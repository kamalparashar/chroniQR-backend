package services

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ScanInfo holds all request metadata to persist for a QR scan.
type ScanInfo struct {
	IP        string
	UserAgent string
	Referrer  string
	Language  string
	UTM       UTMParams
	Geo       GeoInfo
}

// DeviceMeta holds client-side device info.
type DeviceMeta struct {
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

// LogScan persists a scan event.
func LogScan(ctx context.Context, db *pgxpool.Pool, qrID string, info ScanInfo) (string, error) {
	device := ParseDevice(info.UserAgent)

	var scanID string
	err := db.QueryRow(ctx, `
		INSERT INTO qr_scans (
			qr_id, ip, user_agent, device_type, os, browser, referrer, language,
			utm_source, utm_medium, utm_campaign, utm_term, utm_content,
			country, region, city, latitude, longitude
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id
	`,
		qrID,
		nullableStr(info.IP),
		nullableStr(info.UserAgent),
		device.DeviceType,
		device.OS,
		device.Browser,
		nullableStr(info.Referrer),
		nullableStr(info.Language),
		nullableStr(info.UTM.Source),
		nullableStr(info.UTM.Medium),
		nullableStr(info.UTM.Campaign),
		nullableStr(info.UTM.Term),
		nullableStr(info.UTM.Content),
		nullableStr(info.Geo.Country),
		nullableStr(info.Geo.Region),
		nullableStr(info.Geo.City),
		nullableFloat(info.Geo.Latitude),
		nullableFloat(info.Geo.Longitude),
	).Scan(&scanID)

	if err != nil {
		log.Printf("[scanLogger] Failed to log scan: %v", err)
		return "", err
	}
	return scanID, nil
}

// UpdateScanGeo updates scan with precise location and device info.
func UpdateScanGeo(ctx context.Context, db *pgxpool.Pool, scanID string, lat, lng float64, meta *DeviceMeta) error {
	var metaJSON []byte
	var err error
	if meta != nil {
		metaJSON, err = json.Marshal(meta)
		if err != nil {
			metaJSON = []byte("{}")
		}
	} else {
		metaJSON = []byte("{}")
	}

	hasGPS := lat != 0 || lng != 0
	if hasGPS {
		_, err = db.Exec(ctx, `
			UPDATE qr_scans
			SET latitude = $1, longitude = $2, device_meta = $3
			WHERE id = $4
		`, lat, lng, metaJSON, scanID)
	} else {
		_, err = db.Exec(ctx, `
			UPDATE qr_scans
			SET device_meta = $1
			WHERE id = $2
		`, metaJSON, scanID)
	}
	return err
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullableFloat(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}
