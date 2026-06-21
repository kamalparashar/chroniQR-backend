package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	Port            int
	DBURL           string
	RedisURL        string
	SupabaseURL     string
	JWTSecret       string // legacy HS256 fallback
	EncryptionKey   string
	RedirectBaseURL string
	GeoIPDBPath     string
	ShortCodeLength int
	GA4Debug        bool
}

// loadDotEnv reads a .env file and sets environment variables (does not override existing).
func loadDotEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("Required env var %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Load reads configuration from .env and environment variables.
func Load() *Config {
	loadDotEnv(".env")

	port, err := strconv.Atoi(getEnv("PORT", "3000"))
	if err != nil {
		port = 3000
	}

	shortCodeLength, err := strconv.Atoi(getEnv("SHORT_CODE_LENGTH", "8"))
	if err != nil {
		shortCodeLength = 8
	}

	ga4Debug := getEnv("GA4_DEBUG", "false") == "true"

	return &Config{
		Port:            port,
		DBURL:           mustEnv("DB_URL"),
		RedisURL:        getEnv("REDIS_URL", ""),
		SupabaseURL:     getEnv("SUPABASE_URL", ""),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", "default-insecure-32-byte-secret-key!"),
		RedirectBaseURL: mustEnv("REDIRECT_BASE_URL"),
		GeoIPDBPath:     getEnv("GEOIP_DB_PATH", "./assets/GeoLite2-City.mmdb"),
		ShortCodeLength: shortCodeLength,
		GA4Debug:        ga4Debug,
	}
}
