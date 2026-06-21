package services

import (
	"regexp"
	"strings"
)

// DeviceInfo holds parsed user-agent data.
type DeviceInfo struct {
	DeviceType string
	OS         string
	Browser    string
}

var (
	mobileRe  = regexp.MustCompile(`(?i)mobile|android|iphone|ipod|blackberry|windows phone`)
	tabletRe  = regexp.MustCompile(`(?i)ipad|tablet`)
	windowsRe = regexp.MustCompile(`(?i)windows`)
	androidRe = regexp.MustCompile(`(?i)android`)
	iosRe     = regexp.MustCompile(`(?i)iphone|ipad`)
	macRe     = regexp.MustCompile(`(?i)mac os`)
	linuxRe   = regexp.MustCompile(`(?i)linux`)
	edgeRe    = regexp.MustCompile(`(?i)edg/`)
	operaRe   = regexp.MustCompile(`(?i)opr/`)
	chromeRe  = regexp.MustCompile(`(?i)chrome`)
	safariRe  = regexp.MustCompile(`(?i)safari`)
	firefoxRe = regexp.MustCompile(`(?i)firefox`)
)

// ParseDevice extracts device, OS, and browser info from a User-Agent string.
func ParseDevice(userAgent string) DeviceInfo {
	if strings.TrimSpace(userAgent) == "" {
		return DeviceInfo{DeviceType: "unknown", OS: "unknown", Browser: "unknown"}
	}

	deviceType := "desktop"
	if mobileRe.MatchString(userAgent) {
		deviceType = "mobile"
	} else if tabletRe.MatchString(userAgent) {
		deviceType = "tablet"
	}

	os := "unknown"
	switch {
	case windowsRe.MatchString(userAgent):
		os = "Windows"
	case androidRe.MatchString(userAgent):
		os = "Android"
	case iosRe.MatchString(userAgent):
		os = "iOS"
	case macRe.MatchString(userAgent):
		os = "macOS"
	case linuxRe.MatchString(userAgent):
		os = "Linux"
	}

	browser := "unknown"
	switch {
	case edgeRe.MatchString(userAgent):
		browser = "Edge"
	case operaRe.MatchString(userAgent):
		browser = "Opera"
	case chromeRe.MatchString(userAgent):
		browser = "Chrome"
	case safariRe.MatchString(userAgent):
		browser = "Safari"
	case firefoxRe.MatchString(userAgent):
		browser = "Firefox"
	}

	return DeviceInfo{DeviceType: deviceType, OS: os, Browser: browser}
}
