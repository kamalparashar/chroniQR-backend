package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"chroniqr-backend/internal/auth"
	"chroniqr-backend/internal/cache"
	"chroniqr-backend/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QRRow represents the QR code schema mapped to Go.
type QRRow struct {
	ID                 string         `json:"id"`
	ClientID           string         `json:"client_id"`
	ShortCode          string         `json:"short_code"`
	ShortURL           string         `json:"short_url"`
	Name               string         `json:"name"`
	DestinationType    string         `json:"destination_type"`
	DestinationConfig  map[string]any `json:"destination_config"`
	UTMConfig          map[string]any `json:"utm_config"`
	StyleConfig        map[string]any `json:"style_config"`
	Tags               []string       `json:"tags"`
	IsActive           bool           `json:"is_active"`
	ExpiresAt          *time.Time     `json:"expires_at"`
	GA4TrackingEnabled bool           `json:"ga4_tracking_enabled"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// CreateQRRequest defines the incoming payload for creating a QR code.
type CreateQRRequest struct {
	Name               string         `json:"name"`
	DestinationType    string         `json:"destination_type"`
	DestinationConfig  map[string]any `json:"destination_config"`
	UTMConfig          map[string]any `json:"utm_config"`
	StyleConfig        map[string]any `json:"style_config"`
	Tags               []string       `json:"tags"`
	IsActive           *bool          `json:"is_active,omitempty"`
	ExpiresAt          *time.Time     `json:"expires_at,omitempty"`
	GA4TrackingEnabled *bool          `json:"ga4_tracking_enabled,omitempty"`
}

// UpdateQRRequest defines the incoming payload for updating a QR code.
type UpdateQRRequest struct {
	Name               *string         `json:"name,omitempty"`
	DestinationType    *string         `json:"destination_type,omitempty"`
	DestinationConfig  map[string]any  `json:"destination_config,omitempty"`
	UTMConfig          map[string]any  `json:"utm_config,omitempty"`
	StyleConfig        map[string]any  `json:"style_config,omitempty"`
	Tags               []string        `json:"tags,omitempty"`
	IsActive           *bool           `json:"is_active,omitempty"`
	ExpiresAt          **time.Time     `json:"expires_at,omitempty"` // double pointer to support null/removal
	GA4TrackingEnabled *bool           `json:"ga4_tracking_enabled,omitempty"`
}

// CreateQRHandler handles POST /api/qr.
func CreateQRHandler(db *pgxpool.Pool, redirectBaseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req CreateQRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.DestinationType == "" {
			http.Error(w, `{"error":"name and destination_type are required"}`, http.StatusBadRequest)
			return
		}

		if req.DestinationConfig == nil {
			req.DestinationConfig = map[string]any{}
		}
		if req.UTMConfig == nil {
			req.UTMConfig = map[string]any{}
		}
		if req.StyleConfig == nil {
			req.StyleConfig = map[string]any{}
		}
		if req.Tags == nil {
			req.Tags = []string{}
		}

		// Validate destination config
		res := services.ValidateDestinationConfig(req.DestinationType, req.DestinationConfig)
		if !res.OK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "validation failed", "details": res.Errors})
			return
		}

		// Generate unique short code
		code, err := services.GenerateShortCode(r.Context(), db)
		if err != nil {
			http.Error(w, `{"error":"failed to generate short code"}`, http.StatusInternalServerError)
			return
		}

		shortURL := fmt.Sprintf("%s/%s", redirectBaseURL, code)

		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		ga4Enabled := false
		if req.GA4TrackingEnabled != nil {
			ga4Enabled = *req.GA4TrackingEnabled
		}

		var qr QRRow
		err = db.QueryRow(r.Context(), `
			INSERT INTO qr_codes (
				client_id, short_code, short_url, name, destination_type,
				destination_config, utm_config, style_config, tags, is_active, expires_at, ga4_tracking_enabled
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id, client_id, short_code, short_url, name, destination_type, destination_config, utm_config, style_config, tags, is_active, expires_at, ga4_tracking_enabled, created_at, updated_at
		`, clientID, code, shortURL, req.Name, req.DestinationType, req.DestinationConfig, req.UTMConfig, req.StyleConfig, req.Tags, isActive, req.ExpiresAt, ga4Enabled).Scan(
			&qr.ID, &qr.ClientID, &qr.ShortCode, &qr.ShortURL, &qr.Name, &qr.DestinationType, &qr.DestinationConfig, &qr.UTMConfig, &qr.StyleConfig, &qr.Tags, &qr.IsActive, &qr.ExpiresAt, &qr.GA4TrackingEnabled, &qr.CreatedAt, &qr.UpdatedAt,
		)

		if err != nil {
			http.Error(w, `{"error":"failed to save qr code"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(qr)
	}
}

// ListQRHandler handles GET /api/qr.
func ListQRHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(r.Context(), `
			SELECT id, client_id, short_code, short_url, name, destination_type, destination_config, utm_config, style_config, tags, is_active, expires_at, ga4_tracking_enabled, created_at, updated_at
			FROM qr_codes
			WHERE client_id = $1
			ORDER BY created_at DESC
		`, clientID)
		if err != nil {
			http.Error(w, `{"error":"failed to query qr codes"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		qrs := []QRRow{}
		for rows.Next() {
			var qr QRRow
			err := rows.Scan(
				&qr.ID, &qr.ClientID, &qr.ShortCode, &qr.ShortURL, &qr.Name, &qr.DestinationType, &qr.DestinationConfig, &qr.UTMConfig, &qr.StyleConfig, &qr.Tags, &qr.IsActive, &qr.ExpiresAt, &qr.GA4TrackingEnabled, &qr.CreatedAt, &qr.UpdatedAt,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to scan qr code"}`, http.StatusInternalServerError)
				return
			}
			qrs = append(qrs, qr)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(qrs)
	}
}

// GetQRHandler handles GET /api/qr/{id}.
func GetQRHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}

		var qr QRRow
		err := db.QueryRow(r.Context(), `
			SELECT id, client_id, short_code, short_url, name, destination_type, destination_config, utm_config, style_config, tags, is_active, expires_at, ga4_tracking_enabled, created_at, updated_at
			FROM qr_codes
			WHERE id = $1 AND client_id = $2
		`, id, clientID).Scan(
			&qr.ID, &qr.ClientID, &qr.ShortCode, &qr.ShortURL, &qr.Name, &qr.DestinationType, &qr.DestinationConfig, &qr.UTMConfig, &qr.StyleConfig, &qr.Tags, &qr.IsActive, &qr.ExpiresAt, &qr.GA4TrackingEnabled, &qr.CreatedAt, &qr.UpdatedAt,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"qr code not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"failed to fetch qr code"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(qr)
	}
}

// UpdateQRHandler handles PUT /api/qr/{id}.
func UpdateQRHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}

		var existing QRRow
		err := db.QueryRow(r.Context(), `
			SELECT id, client_id, short_code, short_url, name, destination_type, destination_config, utm_config, style_config, tags, is_active, expires_at, ga4_tracking_enabled, created_at, updated_at
			FROM qr_codes
			WHERE id = $1 AND client_id = $2
		`, id, clientID).Scan(
			&existing.ID, &existing.ClientID, &existing.ShortCode, &existing.ShortURL, &existing.Name, &existing.DestinationType, &existing.DestinationConfig, &existing.UTMConfig, &existing.StyleConfig, &existing.Tags, &existing.IsActive, &existing.ExpiresAt, &existing.GA4TrackingEnabled, &existing.CreatedAt, &existing.UpdatedAt,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"qr code not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"failed to fetch qr code"}`, http.StatusInternalServerError)
			return
		}

		var req UpdateQRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
			return
		}

		// Apply updates
		if req.Name != nil {
			existing.Name = *req.Name
		}
		if req.DestinationType != nil {
			existing.DestinationType = *req.DestinationType
		}
		if req.DestinationConfig != nil {
			existing.DestinationConfig = req.DestinationConfig
		}
		if req.UTMConfig != nil {
			existing.UTMConfig = req.UTMConfig
		}
		if req.StyleConfig != nil {
			existing.StyleConfig = req.StyleConfig
		}
		if req.Tags != nil {
			existing.Tags = req.Tags
		}
		if req.IsActive != nil {
			existing.IsActive = *req.IsActive
		}
		if req.ExpiresAt != nil {
			existing.ExpiresAt = *req.ExpiresAt
		}
		if req.GA4TrackingEnabled != nil {
			existing.GA4TrackingEnabled = *req.GA4TrackingEnabled
		}

		// Validate config
		res := services.ValidateDestinationConfig(existing.DestinationType, existing.DestinationConfig)
		if !res.OK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "validation failed", "details": res.Errors})
			return
		}

		var qr QRRow
		err = db.QueryRow(r.Context(), `
			UPDATE qr_codes SET
				name = $1,
				destination_type = $2,
				destination_config = $3,
				utm_config = $4,
				style_config = $5,
				tags = $6,
				is_active = $7,
				expires_at = $8,
				ga4_tracking_enabled = $9,
				updated_at = NOW()
			WHERE id = $10 AND client_id = $11
			RETURNING id, client_id, short_code, short_url, name, destination_type, destination_config, utm_config, style_config, tags, is_active, expires_at, ga4_tracking_enabled, created_at, updated_at
		`, existing.Name, existing.DestinationType, existing.DestinationConfig, existing.UTMConfig, existing.StyleConfig, existing.Tags, existing.IsActive, existing.ExpiresAt, existing.GA4TrackingEnabled, id, clientID).Scan(
			&qr.ID, &qr.ClientID, &qr.ShortCode, &qr.ShortURL, &qr.Name, &qr.DestinationType, &qr.DestinationConfig, &qr.UTMConfig, &qr.StyleConfig, &qr.Tags, &qr.IsActive, &qr.ExpiresAt, &qr.GA4TrackingEnabled, &qr.CreatedAt, &qr.UpdatedAt,
		)

		if err != nil {
			http.Error(w, `{"error":"failed to update qr code"}`, http.StatusInternalServerError)
			return
		}

		// Invalidate cache
		cacheKey := cache.QRKey(qr.ShortCode)
		cache.Delete(r.Context(), cacheKey)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(qr)
	}
}

// DeleteQRHandler handles DELETE /api/qr/{id}.
func DeleteQRHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}

		var shortCode string
		err := db.QueryRow(r.Context(), `
			DELETE FROM qr_codes
			WHERE id = $1 AND client_id = $2
			RETURNING short_code
		`, id, clientID).Scan(&shortCode)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"qr code not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"failed to delete qr code"}`, http.StatusInternalServerError)
			return
		}

		// Invalidate cache
		cacheKey := cache.QRKey(shortCode)
		cache.Delete(r.Context(), cacheKey)

		w.WriteHeader(http.StatusNoContent)
	}
}
