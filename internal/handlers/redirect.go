package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"chroniqr-backend/internal/cache"
	"chroniqr-backend/internal/pages"
	"chroniqr-backend/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Queries ───────────────────────────────────────────────────────────────────

const getQRByShortCode = `
	SELECT 
		q.id,
		q.destination_type,
		q.destination_config,
		q.utm_config,
		q.is_active,
		q.expires_at,
		q.ga4_tracking_enabled,
		c.ga4_measurement_id,
		c.ga4_key
	FROM qr_codes q
	LEFT JOIN clients c ON q.client_id = c.id
	WHERE q.short_code = $1
	LIMIT 1
`

// ── QR row shape for redirect ──────────────────────────────────────────────────

type qrRow struct {
	ID                 string         `json:"id"`
	DestinationType    string         `json:"destination_type"`
	DestinationConfig  map[string]any `json:"destination_config"`
	UTMConfig          map[string]any `json:"utm_config"`
	IsActive           bool           `json:"is_active"`
	ExpiresAt          *time.Time     `json:"expires_at"`
	GA4TrackingEnabled bool           `json:"ga4_tracking_enabled"`
	GA4MeasurementID   *string        `json:"ga4_measurement_id"`
	GA4Key             *string        `json:"ga4_key"`
}

// ── Destination result ────────────────────────────────────────────────────────

type destResult struct {
	Kind   string // "redirect", "email", "call", "vcard"
	URL    string
	Config map[string]any
}

func resolveDestination(destType string, cfg map[string]any, utm services.UTMParams) *destResult {
	switch destType {
	case "url":
		rawURL := strVal(cfg, "url")
		rawURL = services.AppendUTMToURL(rawURL, utm)
		return &destResult{Kind: "redirect", URL: rawURL}

	case "time_based":
		tDestType, tDestCfg, err := services.ResolveTimeBased(cfg, time.Now())
		if err != nil {
			log.Printf("[redirect] time_based routing error: %v", err)
			return nil
		}
		return resolveDestination(tDestType, tDestCfg, utm)

	case "whatsapp":
		phone := strVal(cfg, "phone")
		var sb strings.Builder
		for i, ch := range phone {
			if ch >= '0' && ch <= '9' || (i == 0 && ch == '+') {
				sb.WriteRune(ch)
			}
		}
		msg := ""
		if m := strVal(cfg, "message"); m != "" {
			msg = "?text=" + urlEncode(m)
		}
		return &destResult{Kind: "redirect", URL: fmt.Sprintf("https://wa.me/%s%s", sb.String(), msg)}

	case "call":
		return &destResult{Kind: "call", Config: cfg}

	case "email":
		to := strVal(cfg, "to")
		parts := []string{}
		if sub := strVal(cfg, "subject"); sub != "" {
			parts = append(parts, "subject="+urlEncode(sub))
		}
		if body := strVal(cfg, "body"); body != "" {
			parts = append(parts, "body="+urlEncode(body))
		}
		mailtoURL := "mailto:" + to
		if len(parts) > 0 {
			mailtoURL += "?" + strings.Join(parts, "&")
		}
		return &destResult{Kind: "email", URL: mailtoURL}

	case "vcard":
		return &destResult{Kind: "vcard", Config: cfg}
	}
	return nil
}

// RedirectHandler handles GET /{short_code}.
func RedirectHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortCode := r.PathValue("short_code")
		if shortCode == "" {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			shortCode = parts[len(parts)-1]
		}

		cacheKey := cache.QRKey(shortCode)

		raw, found, err := cache.GetOrLoad(r.Context(), cacheKey, cache.DefaultTTL,
			func() (string, bool, error) {
				var qr qrRow
				err := db.QueryRow(r.Context(), getQRByShortCode, shortCode).Scan(
					&qr.ID, &qr.DestinationType, &qr.DestinationConfig, &qr.UTMConfig,
					&qr.IsActive, &qr.ExpiresAt, &qr.GA4TrackingEnabled, &qr.GA4MeasurementID, &qr.GA4Key,
				)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return "", false, nil
					}
					return "", false, err
				}
				b, err := json.Marshal(qr)
				if err != nil {
					return "", false, err
				}
				return string(b), true, nil
			})

		if err != nil {
			log.Printf("DB lookup failed in redirect: %v", err)
			sendHTML(w, http.StatusNotFound, pages.NotFound())
			return
		}
		if !found {
			sendHTML(w, http.StatusNotFound, pages.NotFound())
			return
		}

		var qr qrRow
		if err := json.Unmarshal([]byte(raw), &qr); err != nil {
			log.Printf("[redirect] failed to decode QR record for %s: %v", shortCode, err)
			sendHTML(w, http.StatusInternalServerError, pages.Error())
			return
		}

		// 2. Inactive
		if !qr.IsActive {
			sendHTML(w, http.StatusGone, pages.Inactive())
			return
		}

		// 3. Expired
		if qr.ExpiresAt != nil && qr.ExpiresAt.Before(time.Now()) {
			go func() {
				_, err := db.Exec(context.Background(),
					"UPDATE qr_codes SET is_active = false WHERE id = $1",
					qr.ID,
				)
				if err != nil {
					log.Printf("[redirect] failed to auto-deactivate expired QR %s: %v", qr.ID, err)
				}
			}()
			cache.Delete(r.Context(), cacheKey)
			sendHTML(w, http.StatusGone, pages.Inactive())
			return
		}

		// 3.5 Time Constraints for Normal Routes
		if qr.DestinationType != "time_based" {
			isActive, err := services.CheckTimeConstraints(qr.DestinationConfig, time.Now())
			if err != nil {
				log.Printf("[redirect] time constraints check failed for %s: %v", shortCode, err)
				sendHTML(w, http.StatusGone, pages.Inactive())
				return
			}
			if !isActive {
				sendHTML(w, http.StatusGone, pages.Inactive())
				return
			}
		}

		// 4. Extract UTM from inbound scan URL
		inboundUTM := services.ExtractUTM(r)

		// 5. Merge with stored utm_config — stored config wins over inbound
		storedUTM := qr.UTMConfig
		if storedUTM == nil {
			storedUTM = map[string]any{}
		}
		finalUTM := services.MergeUTM(storedUTM, inboundUTM)

		// 6. Geo-IP lookup
		ip := extractIP(r)
		geo := services.LookupGeo(ip)

		// 7. Resolve destination
		dest := resolveDestination(qr.DestinationType, qr.DestinationConfig, finalUTM)
		if dest == nil {
			sendHTML(w, http.StatusInternalServerError, pages.Error())
			return
		}

		log.Printf("✅ QR Scanned: %s (destination: %s)", shortCode, dest.Kind)

		// 8. Log scan to Database SYNCHRONOUSLY (need scan_id for geolocation page)
		ua := r.Header.Get("User-Agent")
		ref := r.Header.Get("Referer")
		lang := r.Header.Get("Accept-Language")
		scanID, scanErr := services.LogScan(r.Context(), db, qr.ID, services.ScanInfo{
			IP:        ip,
			UserAgent: ua,
			Referrer:  ref,
			Language:  lang,
			UTM:       finalUTM,
			Geo:       geo,
		})
		if scanErr != nil {
			log.Printf("[redirect] scan log failed: %v (continuing without geolocation)", scanErr)
		}

		// 9. Async: fire GA4 event
		device := services.ParseDevice(ua)

		if qr.GA4TrackingEnabled && qr.GA4MeasurementID != nil && qr.GA4Key != nil {
			go func() {
				measurementID, err1 := services.Decrypt(*qr.GA4MeasurementID)
				apiSecret, err2 := services.Decrypt(*qr.GA4Key)

				if err1 == nil && err2 == nil && measurementID != "" && apiSecret != "" {
					services.SendGA4ScanEvent(services.QRScanEvent{
						ScanID:          scanID,
						ShortCode:       shortCode,
						DestinationType: qr.DestinationType,
						UTM:             finalUTM,
						Geo:             geo,
						ClientIP:        ip,
						UserAgent:       ua,
						DeviceType:      device.DeviceType,
						OS:              device.OS,
						Browser:         device.Browser,
					}, measurementID, apiSecret)
				} else {
					log.Printf("[redirect] failed to decrypt GA4 credentials or they are empty")
				}
			}()
		}

		// 10. Send response
		switch dest.Kind {
		case "redirect":
			if scanID != "" {
				sendHTML(w, http.StatusOK, pages.GeolocationPage(scanID, dest.URL))
			} else {
				http.Redirect(w, r, dest.URL, http.StatusFound)
			}
		case "email":
			sendHTML(w, http.StatusOK, pages.EmailLanding(dest.URL)+pages.GeoBackgroundScript(scanID))
		case "call":
			sendHTML(w, http.StatusOK, pages.CallLanding(strVal(dest.Config, "caller_number"), strVal(dest.Config, "landing_page_text"))+pages.GeoBackgroundScript(scanID))
		case "vcard":
			name := strVal(dest.Config, "name")
			filename := strings.ReplaceAll(name, " ", "_") + ".vcf"
			w.Header().Set("Content-Type", "text/vcard")
			w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
			w.Write([]byte(buildVCard(dest.Config)))
		default:
			sendHTML(w, http.StatusInternalServerError, pages.Error())
			return
		}
	}
}

type scanLocationRequest struct {
	ScanID    string  `json:"scan_id"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`

	ScreenWidth    int     `json:"screen_width,omitempty"`
	ScreenHeight   int     `json:"screen_height,omitempty"`
	Timezone       string  `json:"timezone,omitempty"`
	Language       string  `json:"language,omitempty"`
	Platform       string  `json:"platform,omitempty"`
	ColorDepth     int     `json:"color_depth,omitempty"`
	PixelRatio     float64 `json:"pixel_ratio,omitempty"`
	TouchSupport   bool    `json:"touch_support,omitempty"`
	ConnectionType string  `json:"connection_type,omitempty"`
	OnLine         bool    `json:"online,omitempty"`
	CookiesEnabled bool    `json:"cookies_enabled,omitempty"`
}

// ScanLocationHandler handles POST /api/scan-location.
func ScanLocationHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req scanLocationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		if req.ScanID == "" {
			http.Error(w, `{"error":"scan_id required"}`, http.StatusBadRequest)
			return
		}
		if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
			http.Error(w, `{"error":"invalid coordinates"}`, http.StatusBadRequest)
			return
		}

		meta := &services.DeviceMeta{
			ScreenWidth:    req.ScreenWidth,
			ScreenHeight:   req.ScreenHeight,
			Timezone:       req.Timezone,
			Language:       req.Language,
			Platform:       req.Platform,
			ColorDepth:     req.ColorDepth,
			PixelRatio:     req.PixelRatio,
			TouchSupport:   req.TouchSupport,
			ConnectionType: req.ConnectionType,
			OnLine:         req.OnLine,
			CookiesEnabled: req.CookiesEnabled,
		}

		if err := services.UpdateScanGeo(r.Context(), db, req.ScanID, req.Latitude, req.Longitude, meta); err != nil {
			log.Printf("[scan-location] Failed to update scan %s: %v", req.ScanID, err)
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[scan-location] ✅ Updated scan %s — GPS: %.6f, %.6f | Screen: %dx%d | TZ: %s | Lang: %s",
			req.ScanID, req.Latitude, req.Longitude, req.ScreenWidth, req.ScreenHeight, req.Timezone, req.Language)
		w.WriteHeader(http.StatusNoContent)
	}
}

// ── IP extractor ──────────────────────────────────────────────────────────────

func extractIP(r *http.Request) string {
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}
	if tcIP := r.Header.Get("True-Client-IP"); tcIP != "" {
		return strings.TrimSpace(tcIP)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}
	return r.RemoteAddr
}

func sendHTML(w http.ResponseWriter, status int, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(html))
}

func buildVCard(cfg map[string]any) string {
	lines := []string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"FN:" + strVal(cfg, "name"),
		"N:" + strVal(cfg, "name") + ";;;",
	}
	if v := strVal(cfg, "phone"); v != "" {
		lines = append(lines, "TEL:"+v)
	}
	if v := strVal(cfg, "email"); v != "" {
		lines = append(lines, "EMAIL:"+v)
	}
	if v := strVal(cfg, "company"); v != "" {
		lines = append(lines, "ORG:"+v)
	}
	if v := strVal(cfg, "website"); v != "" {
		lines = append(lines, "URL:"+v)
	}
	if v := strVal(cfg, "note"); v != "" {
		lines = append(lines, "NOTE:"+v)
	}
	lines = append(lines, "END:VCARD")
	return strings.Join(lines, "\r\n")
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func urlEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
