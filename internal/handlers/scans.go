package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"chroniqr-backend/internal/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ScanRecord maps to a row in qr_scans, using column names that match
// what the frontend AnalyticsView expects.
type ScanRecord struct {
	ID          string         `json:"id"`
	QrID        string         `json:"qr_id"`
	ScannedAt   time.Time      `json:"scanned_at"`
	IP          *string        `json:"ip"`
	UserAgent   *string        `json:"user_agent"`
	DeviceType  *string        `json:"device_type"`
	OS          *string        `json:"os"`
	Browser     *string        `json:"browser"`
	Referrer    *string        `json:"referrer"`
	Language    *string        `json:"language"`
	UTMSource   *string        `json:"utm_source"`
	UTMMedium   *string        `json:"utm_medium"`
	UTMCampaign *string        `json:"utm_campaign"`
	UTMTerm     *string        `json:"utm_term"`
	UTMContent  *string        `json:"utm_content"`
	Country     *string        `json:"country"`
	Region      *string        `json:"region"`
	City        *string        `json:"city"`
	Latitude    *float64       `json:"latitude"`
	Longitude   *float64       `json:"longitude"`
	DeviceMeta  map[string]any `json:"device_meta"`
}

// ListScansHandler handles GET /api/scans?qr_id=<id>
// Returns all scan records for a specific QR code owned by the authenticated client.
func ListScansHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		qrID := r.URL.Query().Get("qr_id")
		if qrID == "" {
			http.Error(w, `{"error":"qr_id query parameter is required"}`, http.StatusBadRequest)
			return
		}

		// JOIN against qr_codes ensures the QR belongs to this client (security check)
		// NOTE: s.ip is cast to TEXT because the column is Postgres INET type,
		// which pgx cannot scan directly into *string.
		rows, err := db.Query(r.Context(), `
			SELECT s.id, s.qr_id, s.created_at,
			       s.ip::TEXT, s.user_agent, s.device_type, s.os, s.browser,
			       s.referrer, s.language,
			       s.utm_source, s.utm_medium, s.utm_campaign, s.utm_term, s.utm_content,
			       s.country, s.region, s.city, s.latitude, s.longitude, s.device_meta
			FROM qr_scans s
			INNER JOIN qr_codes q ON q.id = s.qr_id
			WHERE s.qr_id = $1 AND q.client_id = $2
			ORDER BY s.created_at DESC
			LIMIT 1000
		`, qrID, clientID)
		if err != nil {
			log.Printf("[scans] DB query failed in ListScansHandler: %v", err)
			http.Error(w, `{"error":"failed to query scans"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		scans := []ScanRecord{}
		for rows.Next() {
			var s ScanRecord
			err := rows.Scan(
				&s.ID, &s.QrID, &s.ScannedAt,
				&s.IP, &s.UserAgent, &s.DeviceType, &s.OS, &s.Browser,
				&s.Referrer, &s.Language,
				&s.UTMSource, &s.UTMMedium, &s.UTMCampaign, &s.UTMTerm, &s.UTMContent,
				&s.Country, &s.Region, &s.City, &s.Latitude, &s.Longitude, &s.DeviceMeta,
			)
			if err != nil {
				log.Printf("[scans] Row scan failed in ListScansHandler: %v", err)
				http.Error(w, `{"error":"failed to scan record"}`, http.StatusInternalServerError)
				return
			}
			scans = append(scans, s)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(scans)
	}
}

// GetScansCountHandler handles GET /api/scans/count
// Returns { "<qr_id>": <count>, ... } for all QR codes owned by the client.
func GetScansCountHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(r.Context(), `
			SELECT s.qr_id, COUNT(*) AS scan_count
			FROM qr_scans s
			INNER JOIN qr_codes q ON q.id = s.qr_id
			WHERE q.client_id = $1
			GROUP BY s.qr_id
		`, clientID)
		if err != nil {
			log.Printf("[scans] DB query failed in GetScansCountHandler: %v", err)
			http.Error(w, `{"error":"failed to query scan counts"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		counts := map[string]int{}
		for rows.Next() {
			var qrID string
			var count int64
			if err := rows.Scan(&qrID, &count); err != nil {
				log.Printf("[scans] Row scan failed in GetScansCountHandler: %v", err)
				http.Error(w, `{"error":"failed to scan count row"}`, http.StatusInternalServerError)
				return
			}
			counts[qrID] = int(count)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(counts)
	}
}
