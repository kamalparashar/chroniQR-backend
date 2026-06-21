package services

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// GeoInfo holds geographic data resolved from an IP address.
type GeoInfo struct {
	Country   string
	Region    string
	City      string
	Latitude  float64
	Longitude float64
}

var (
	geoDB     *geoip2.Reader
	geoDBOnce sync.Once
)

// InitGeoIP loads the MaxMind GeoLite2 .mmdb file once at startup.
// Call this from main() before starting the HTTP server.
// If the file is missing or unreadable, geo lookups will return empty
// GeoInfo silently — the server will still start and run normally.
func InitGeoIP(path string) error {
	var initErr error
	geoDBOnce.Do(func() {
		db, err := geoip2.Open(path)
		if err != nil {
			initErr = err
			log.Printf("[geoip] Failed to load DB at %q: %v — geo fields will be empty", path, err)
			return
		}
		geoDB = db
		log.Printf("[geoip] Loaded MaxMind DB from %q", path)
	})
	return initErr
}

// CloseGeoIP releases the MaxMind DB file handle.
// Call via defer in main() for clean shutdown.
func CloseGeoIP() {
	if geoDB != nil {
		geoDB.Close()
	}
}

// LookupGeo returns geographic info for a given IP string.
// Returns an empty GeoInfo (no error) when:
//   - the DB is not loaded
//   - the IP string is invalid
//   - the IP is a private/loopback address (local dev)
//   - the IP is not found in the DB
func LookupGeo(ipStr string) GeoInfo {
	if geoDB == nil {
		return fallbackLookup(ipStr)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil || isPrivateIP(ip) {
		return GeoInfo{}
	}

	record, err := geoDB.City(ip)
	if err != nil {
		return fallbackLookup(ipStr)
	}

	city := record.City.Names["en"]

	region := ""
	if len(record.Subdivisions) > 0 {
		region = record.Subdivisions[0].Names["en"]
	}

	return GeoInfo{
		Country:   record.Country.Names["en"],
		Region:    region,
		City:      city,
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
	}
}

// ── Fallback API ──────────────────────────────────────────────────────────

func fallbackLookup(ip string) GeoInfo {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip)
	if err != nil {
		return GeoInfo{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return GeoInfo{}
	}

	var data struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return GeoInfo{}
	}

	if data.Status != "success" {
		return GeoInfo{}
	}

	return GeoInfo{
		Country:   data.Country,
		Region:    data.RegionName,
		City:      data.City,
		Latitude:  data.Lat,
		Longitude: data.Lon,
	}
}

// ── private IP ranges ─────────────────────────────────────────────────────

var privateRanges []*net.IPNet

func init() {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16", // link-local
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			privateRanges = append(privateRanges, network)
		}
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, network := range privateRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
