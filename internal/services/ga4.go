package services

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"
)

const (
	ga4Endpoint      = "https://www.google-analytics.com/mp/collect"
	ga4DebugEndpoint = "https://www.google-analytics.com/debug/mp/collect"
)

// QRScanEvent holds all data to send as a GA4 qr_scan event.
type QRScanEvent struct {
	ScanID          string // Unique per scan — used as GA4 client_id to track individual devices
	ShortCode       string
	DestinationType string
	UTM             UTMParams
	Geo             GeoInfo
	ClientIP        string
	UserAgent       string
	DeviceType      string
	OS              string
	Browser         string
}

type ga4Event struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params"`
}

type ga4Payload struct {
	ClientID           string     `json:"client_id"`
	TimestampMicros    int64      `json:"timestamp_micros"`
	NonPersonalizedAds bool       `json:"non_personalized_ads"`
	IPOverride         string     `json:"ip_override,omitempty"`
	UserAgent          string     `json:"user_agent,omitempty"`
	Events             []ga4Event `json:"events"`
}

func generateSessionID() int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return time.Now().UnixNano()
	}
	return n.Int64()
}

// SendGA4ScanEvent fires a session_start + qr_scan event pair to GA4 via the Measurement Protocol.
func SendGA4ScanEvent(event QRScanEvent, measurementID string, apiSecret string) {
	if measurementID == "" || apiSecret == "" {
		return
	}

	go func() {
		sessionID := generateSessionID()

		utmParams := map[string]any{}
		if event.UTM.Source != "" {
			utmParams["source"] = event.UTM.Source
		}
		if event.UTM.Medium != "" {
			utmParams["medium"] = event.UTM.Medium
		}
		if event.UTM.Campaign != "" {
			utmParams["campaign"] = event.UTM.Campaign
		}
		if event.UTM.Term != "" {
			utmParams["term"] = event.UTM.Term
		}
		if event.UTM.Content != "" {
			utmParams["content"] = event.UTM.Content
		}

		scanParams := map[string]any{
			"session_id":           sessionID,
			"engagement_time_msec": 1,
			"short_code":           event.ShortCode,
			"destination_type":     event.DestinationType,
		}
		for k, v := range utmParams {
			scanParams[k] = v
		}
		if event.Geo.Country != "" {
			scanParams["country"] = event.Geo.Country
		}
		if event.Geo.Region != "" {
			scanParams["region"] = event.Geo.Region
		}
		if event.Geo.City != "" {
			scanParams["city"] = event.Geo.City
		}

		payload := ga4Payload{
			ClientID:        event.ScanID,
			TimestampMicros: time.Now().UnixMicro(),
			IPOverride:      event.ClientIP,
			UserAgent:       event.UserAgent,
			Events: []ga4Event{
				{Name: "qr_scan", Params: scanParams},
			},
		}

		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[ga4] marshal error: %v", err)
			return
		}

		endpoint := ga4Endpoint
		if os.Getenv("GA4_DEBUG") == "true" {
			endpoint = ga4DebugEndpoint
		}

		url := fmt.Sprintf("%s?measurement_id=%s&api_secret=%s",
			endpoint, measurementID, apiSecret)

		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("[ga4] send error: %v", err)
			return
		}
		defer resp.Body.Close()

		if os.Getenv("GA4_DEBUG") == "true" && resp.StatusCode == http.StatusOK {
			debugBody, _ := io.ReadAll(resp.Body)
			log.Printf("[ga4] debug response: %s", string(debugBody))
		}

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			log.Printf("[ga4] unexpected status: %d", resp.StatusCode)
		}
	}()
}
