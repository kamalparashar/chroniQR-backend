package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"chroniqr-backend/internal/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ScanRecord represents a single scan event from qr_scans.
type ScanRecord struct {
	ID              string     `json:"id"`
	QrID            string     `json:"qr_id"`
	ScannedAt       time.Time  `json:"scanned_at"`
	IPAddress       string     `json:"ip_address"`
	Country         string     `json:"country"`
	City            string     `json:"city"`
	DeviceType      string     `json:"device_type"`
	OS              string     `json:"os"`
	Browser         string     `json:"browser"`
	DestinationURL  string     `json:"destination_url"`
	Referrer        string     `json:"referrer"`
	Lat             *float64   `json:"lat,omitempty"`
	Lng             *float64   `json:"lng,omitempty"`
	LocationUpdated bool       `json:"location_updated"`
}

// ListScansHandler handles GET /api/scans?qr_id=<id>
// Returns all scan records for a specific QR code that belongs to the authenticated client.
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

		// Verify the QR code belongs to the requesting client before returning its scans
		rows, err := db.Query(r.Context(), `
			SELECT s.id, s.qr_id, s.scanned_at, s.ip_address, s.country, s.city,
			       s.device_type, s.os, s.browser, s.destination_url, s.referrer,
			       s.lat, s.lng, s.location_updated
			FROM qr_scans s
			INNER JOIN qr_codes q ON q.id = s.qr_id
			WHERE s.qr_id = $1 AND q.client_id = $2
			ORDER BY s.scanned_at DESC
			LIMIT 1000
		`, qrID, clientID)
		if err != nil {
			http.Error(w, `{"error":"failed to query scans"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		scans := []ScanRecord{}
		for rows.Next() {
			var s ScanRecord
			err := rows.Scan(
				&s.ID, &s.QrID, &s.ScannedAt, &s.IPAddress, &s.Country, &s.City,
				&s.DeviceType, &s.OS, &s.Browser, &s.DestinationURL, &s.Referrer,
				&s.Lat, &s.Lng, &s.LocationUpdated,
			)
			if err != nil {
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
// Returns a map of { "<qr_id>": <total_scan_count>, ... } for all QR codes owned by the client.
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
			http.Error(w, `{"error":"failed to query scan counts"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		counts := map[string]int{}
		for rows.Next() {
			var qrID string
			var count int
			if err := rows.Scan(&qrID, &count); err != nil {
				http.Error(w, `{"error":"failed to scan count row"}`, http.StatusInternalServerError)
				return
			}
			counts[qrID] = count
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(counts)
	}
}
