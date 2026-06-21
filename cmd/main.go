package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"chroniqr-backend/internal/auth"
	"chroniqr-backend/internal/cache"
	"chroniqr-backend/internal/config"
	"chroniqr-backend/internal/db"
	"chroniqr-backend/internal/handlers"
	"chroniqr-backend/internal/services"
)

func withLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func withRecovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[panic] %v", rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func withRealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
			r.RemoteAddr = strings.TrimSpace(cfIP)
		} else if tcIP := r.Header.Get("True-Client-IP"); tcIP != "" {
			r.RemoteAddr = strings.TrimSpace(tcIP)
		} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
			r.RemoteAddr = strings.TrimSpace(xri)
		} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			r.RemoteAddr = strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
		}
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Println("Initializing chroniQR backend...")

	cfg := config.Load()
	services.InitCrypto(cfg.EncryptionKey)
	auth.InitAuth(cfg.JWTSecret)

	// ── Database Connection ───────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	database, err := db.Connect(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("Fatal: Database connection failed: %v", err)
	}
	defer db.Close()

	// ── Redis Cache Connection ────────────────────────────────────────────
	cache.Init(cfg.RedisURL)
	defer cache.Close()

	// ── GeoIP maxmind DB initialization ───────────────────────────────────
	if err := services.InitGeoIP(cfg.GeoIPDBPath); err != nil {
		log.Printf("⚠️  GeoIP unavailable (%v) — geo fields will be empty", err)
	}
	defer services.CloseGeoIP()

	// ── Routing ───────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	// Protected endpoints helper
	protected := func(handler http.HandlerFunc) http.Handler {
		return auth.Middleware(handler)
	}

	// QR Code CRUD REST endpoints
	mux.Handle("POST /api/qr", protected(handlers.CreateQRHandler(database, cfg.RedirectBaseURL)))
	mux.Handle("GET /api/qr", protected(handlers.ListQRHandler(database)))
	mux.Handle("GET /api/qr/{id}", protected(handlers.GetQRHandler(database)))
	mux.Handle("PUT /api/qr/{id}", protected(handlers.UpdateQRHandler(database)))
	mux.Handle("DELETE /api/qr/{id}", protected(handlers.DeleteQRHandler(database)))

	// GA4 Credentials endpoints
	mux.Handle("GET /api/client/ga4", protected(handlers.GetGA4CredentialsHandler(database)))
	mux.Handle("POST /api/client/ga4", protected(handlers.UpsertGA4CredentialsHandler(database)))
	mux.Handle("DELETE /api/client/ga4", protected(handlers.DeleteGA4CredentialsHandler(database)))

	// Scan analytics endpoints
	mux.Handle("GET /api/scans/count", protected(handlers.GetScansCountHandler(database)))
	mux.Handle("GET /api/scans", protected(handlers.ListScansHandler(database)))

	// Public Redirect router scanned by users
	mux.HandleFunc("GET /{short_code}", handlers.RedirectHandler(database))

	// Precise geolocation callback from redirection interstitial page
	mux.HandleFunc("POST /api/scan-location", handlers.ScanLocationHandler(database))

	// 404 fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"not found"}`)
	})

	handler := withCORS(withLogging(withRecovery(withRealIP(mux))))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf(`
  ✅  chroniQR Backend (Go 1.24+)
  ──────────────────────────────────────
  Local:     http://localhost%s
  Health:    http://localhost%s/health
  Redirects: http://localhost%s/{code}
`, addr, addr, addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
