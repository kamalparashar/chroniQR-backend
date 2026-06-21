package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"chroniqr-backend/internal/auth"
	"chroniqr-backend/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GA4CredentialsRequest represents the request payload for setting credentials.
type GA4CredentialsRequest struct {
	MeasurementID string `json:"ga4_measurement_id"`
	Key           string `json:"ga4_key"`
}

// GA4CredentialsResponse represents the decrypted response.
type GA4CredentialsResponse struct {
	MeasurementID string `json:"ga4_measurement_id"`
	Key           string `json:"ga4_key"`
}

// GetGA4CredentialsHandler handles GET /api/client/ga4.
func GetGA4CredentialsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var encMeasurementID *string
		var encKey *string

		err := db.QueryRow(r.Context(),
			"SELECT ga4_measurement_id, ga4_key FROM clients WHERE id = $1",
			clientID,
		).Scan(&encMeasurementID, &encKey)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, `{"error":"client not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}

		var measurementID, key string
		if encMeasurementID != nil {
			measurementID, _ = services.Decrypt(*encMeasurementID)
		}
		if encKey != nil {
			key, _ = services.Decrypt(*encKey)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GA4CredentialsResponse{
			MeasurementID: measurementID,
			Key:           key,
		})
	}
}

// UpsertGA4CredentialsHandler handles POST /api/client/ga4.
func UpsertGA4CredentialsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req GA4CredentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
			return
		}

		if req.MeasurementID == "" || req.Key == "" {
			http.Error(w, `{"error":"ga4_measurement_id and ga4_key are required"}`, http.StatusBadRequest)
			return
		}

		encMeasurementID, err1 := services.Encrypt(req.MeasurementID)
		encKey, err2 := services.Encrypt(req.Key)
		if err1 != nil || err2 != nil {
			http.Error(w, `{"error":"failed to encrypt credentials"}`, http.StatusInternalServerError)
			return
		}

		_, err := db.Exec(r.Context(),
			"UPDATE clients SET ga4_measurement_id = $1, ga4_key = $2, updated_at = NOW() WHERE id = $3",
			encMeasurementID, encKey, clientID,
		)

		if err != nil {
			http.Error(w, `{"error":"failed to save credentials"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

// DeleteGA4CredentialsHandler handles DELETE /api/client/ga4.
func DeleteGA4CredentialsHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := auth.GetClientID(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		_, err := db.Exec(r.Context(),
			"UPDATE clients SET ga4_measurement_id = NULL, ga4_key = NULL, updated_at = NOW() WHERE id = $1",
			clientID,
		)

		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
