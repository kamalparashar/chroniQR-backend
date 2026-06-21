package services

import (
	"encoding/json"
	"fmt"
	"time"
	_ "time/tzdata" // embed IANA timezone DB so LoadLocation works on Windows/Docker scratch
)

// ── Structs ───────────────────────────────────────────────────────────────────

// TimeRule is a single routing rule inside a time_based destination_config.
// Days uses Go/JS convention: 0 = Sunday, 1 = Monday … 6 = Saturday.
// StartTime and EndTime are "HH:MM" in 24-hour format.
// Midnight-spanning windows (e.g. "22:00"–"02:00") are supported.
//
// Each rule routes to a full destination type (call, url, whatsapp, email, vcard)
// via DestinationType + DestinationConfig.
// For convenience, a plain URL shorthand is also accepted via Destination.
type TimeRule struct {
	StartTime string `json:"start_time"` // "HH:MM"
	EndTime   string `json:"end_time"`   // "HH:MM"
	Days      []int  `json:"days"`       // 0=Sun … 6=Sat

	// Rich destination — preferred; supports any destination_type
	DestinationType   string         `json:"destination_type"`
	DestinationConfig map[string]any `json:"destination_config"`

	// Simple URL shorthand (backward compat) — used when destination_type is absent
	Destination string `json:"destination"`
}

// TimeBasedConfig is the full structure expected in destination_config when
// destination_type == "time_based".
type TimeBasedConfig struct {
	Type     string     `json:"type"`     // always "time_based"
	Timezone string     `json:"timezone"` // IANA timezone, e.g. "America/New_York"
	Rules    []TimeRule `json:"rules"`

	// Rich default — preferred
	DefaultType   string         `json:"default_type"`
	DefaultConfig map[string]any `json:"default_config"`

	// Simple URL default (backward compat) — used when default_type is absent
	DefaultURL string `json:"default_url"`
}

// ── Resolver ──────────────────────────────────────────────────────────────────

// ResolveTimeBased evaluates the destination_config against the provided time
// (typically time.Now()) and returns the destination type and config to use.
//
// The returned (destType, destCfg) are passed directly into resolveDestination,
// so all existing destination handlers (call, url, whatsapp, email, vcard) work
// without any changes.
//
// Rules are evaluated in declaration order; the first matching rule wins.
// If no rule matches, the default destination is returned.
// now is passed as a parameter to keep the function pure and testable.
func ResolveTimeBased(cfg map[string]any, now time.Time) (destType string, destCfg map[string]any, err error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", nil, fmt.Errorf("time_based: marshal config: %w", err)
	}
	var config TimeBasedConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", nil, fmt.Errorf("time_based: parse config: %w", err)
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil || config.Timezone == "" {
		loc = time.UTC
	}

	localNow := now.In(loc)
	currentDay := int(localNow.Weekday()) // 0=Sun … 6=Sat
	currentMins := localNow.Hour()*60 + localNow.Minute()

	for _, rule := range config.Rules {
		if !dayMatches(currentDay, rule.Days) {
			continue
		}
		startMins, err1 := parseHHMM(rule.StartTime)
		endMins, err2 := parseHHMM(rule.EndTime)
		if err1 != nil || err2 != nil {
			// Skip malformed rule rather than crashing.
			continue
		}
		if timeInWindow(currentMins, startMins, endMins) {
			return ruleDestination(rule)
		}
	}

	return defaultDestination(config)
}

// ruleDestination extracts (destType, destCfg) from a matched rule.
func ruleDestination(rule TimeRule) (string, map[string]any, error) {
	if rule.DestinationType != "" {
		cfg := rule.DestinationConfig
		if cfg == nil {
			cfg = map[string]any{}
		}
		return rule.DestinationType, cfg, nil
	}
	if rule.Destination != "" {
		return "url", map[string]any{"url": rule.Destination}, nil
	}
	return "", nil, fmt.Errorf("time_based: rule has no destination_type or destination")
}

// defaultDestination extracts (destType, destCfg) from the config's default.
func defaultDestination(config TimeBasedConfig) (string, map[string]any, error) {
	if config.DefaultType != "" {
		cfg := config.DefaultConfig
		if cfg == nil {
			cfg = map[string]any{}
		}
		return config.DefaultType, cfg, nil
	}
	if config.DefaultURL != "" {
		return "url", map[string]any{"url": config.DefaultURL}, nil
	}
	return "", nil, fmt.Errorf("time_based: no matching rule and no default destination configured")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseHHMM(s string) (int, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, fmt.Errorf("invalid time %q: %w", s, err)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid time %q: out of range", s)
	}
	return h*60 + m, nil
}

func dayMatches(day int, allowed []int) bool {
	for _, d := range allowed {
		if d == day {
			return true
		}
	}
	return false
}

func timeInWindow(current, start, end int) bool {
	if start <= end {
		return current >= start && current <= end
	}
	return current >= start || current <= end
}

// ── Root Time Constraints ─────────────────────────────────────────────────────

// TimeConstraints represents optional time restrictions on a regular route.
type TimeConstraints struct {
	Timezone  string `json:"timezone"`
	Days      []int  `json:"days"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// HasTimeConstraints checks if a config map contains time constraint fields.
func HasTimeConstraints(cfg map[string]any) bool {
	if cfg == nil {
		return false
	}
	_, hasTz := cfg["timezone"]
	_, hasDays := cfg["days"]
	_, hasStart := cfg["start_time"]
	_, hasEnd := cfg["end_time"]
	return hasTz || hasDays || hasStart || hasEnd
}

// CheckTimeConstraints returns true if the current time satisfies the constraints.
func CheckTimeConstraints(cfg map[string]any, now time.Time) (bool, error) {
	if !HasTimeConstraints(cfg) {
		return true, nil
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		return false, fmt.Errorf("marshal time constraints: %w", err)
	}
	var constraints TimeConstraints
	if err := json.Unmarshal(raw, &constraints); err != nil {
		return false, fmt.Errorf("parse time constraints: %w", err)
	}

	loc, err := time.LoadLocation(constraints.Timezone)
	if err != nil || constraints.Timezone == "" {
		loc = time.UTC
	}

	localNow := now.In(loc)
	currentDay := int(localNow.Weekday()) // 0=Sun … 6=Sat
	currentMins := localNow.Hour()*60 + localNow.Minute()

	if !dayMatches(currentDay, constraints.Days) {
		return false, nil
	}

	startMins, err1 := parseHHMM(constraints.StartTime)
	endMins, err2 := parseHHMM(constraints.EndTime)
	if err1 != nil || err2 != nil {
		return false, fmt.Errorf("malformed time constraints")
	}

	return timeInWindow(currentMins, startMins, endMins), nil
}
