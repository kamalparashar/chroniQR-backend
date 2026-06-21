package services

import (
	"fmt"
	"net/url"
	"regexp"
	"time"
)

var e164Re = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// ValidationResult holds the outcome of destination config validation.
type ValidationResult struct {
	OK     bool
	Errors map[string]string
}

func validateE164(phone string) error {
	if !e164Re.MatchString(phone) {
		return fmt.Errorf("must be E.164 format e.g. +919876543210")
	}
	return nil
}

func validateURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("must be a valid URL")
	}
	return nil
}

// ValidateTimeConstraints checks the validity of optional time constraints on a route.
func ValidateTimeConstraints(cfg map[string]any) map[string]string {
	errs := map[string]string{}

	if !HasTimeConstraints(cfg) {
		return errs
	}

	tz, _ := cfg["timezone"].(string)
	if tz == "" {
		errs["timezone"] = "timezone is required when setting time constraints"
	} else if _, err := time.LoadLocation(tz); err != nil {
		errs["timezone"] = fmt.Sprintf("invalid IANA timezone: %q", tz)
	}

	for _, field := range []string{"start_time", "end_time"} {
		v, _ := cfg[field].(string)
		if v == "" {
			errs[field] = "required (HH:MM)"
		} else {
			var h, m int
			if _, err := fmt.Sscanf(v, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
				errs[field] = fmt.Sprintf("invalid time %q, must be HH:MM (24h)", v)
			}
		}
	}

	days, _ := cfg["days"].([]any)
	if len(days) == 0 {
		errs["days"] = "at least one day is required (0=Sun … 6=Sat)"
	} else {
		for j, d := range days {
			switch v := d.(type) {
			case float64:
				if int(v) < 0 || int(v) > 6 {
					errs[fmt.Sprintf("days[%d]", j)] = "day must be 0–6 (0=Sun)"
				}
			default:
				errs[fmt.Sprintf("days[%d]", j)] = "day must be an integer 0–6"
			}
		}
	}
	return errs
}

// ValidateDestinationConfig validates destination_config for a given destination_type.
func ValidateDestinationConfig(destType string, cfg map[string]any) ValidationResult {
	var errs map[string]string
	if destType != "time_based" {
		errs = ValidateTimeConstraints(cfg)
	} else {
		errs = map[string]string{}
	}

	switch destType {
	case "call":
		agentID, _ := cfg["agent_id"].(string)
		if agentID == "" {
			errs["agent_id"] = "agent_id must be a valid UUID"
		}
		if phone, ok := cfg["caller_number"].(string); ok && phone != "" {
			if err := validateE164(phone); err != nil {
				errs["caller_number"] = err.Error()
			}
		}
		if text, ok := cfg["landing_page_text"].(string); ok && len(text) > 120 {
			errs["landing_page_text"] = "must be 120 characters or fewer"
		}

	case "whatsapp":
		phone, _ := cfg["phone"].(string)
		if phone == "" {
			errs["phone"] = "phone is required"
		} else if err := validateE164(phone); err != nil {
			errs["phone"] = err.Error()
		}
		if msg, ok := cfg["message"].(string); ok && len(msg) > 4096 {
			errs["message"] = "must be 4096 characters or fewer"
		}

	case "url":
		rawURL, _ := cfg["url"].(string)
		if rawURL == "" {
			errs["url"] = "url is required"
		} else if err := validateURL(rawURL); err != nil {
			errs["url"] = err.Error()
		}

	case "vcard":
		name, _ := cfg["name"].(string)
		if name == "" {
			errs["name"] = "name is required"
		}
		if phone, ok := cfg["phone"].(string); ok && phone != "" {
			if err := validateE164(phone); err != nil {
				errs["phone"] = err.Error()
			}
		}
		if website, ok := cfg["website"].(string); ok && website != "" {
			if err := validateURL(website); err != nil {
				errs["website"] = err.Error()
			}
		}

	case "email":
		to, _ := cfg["to"].(string)
		if to == "" {
			errs["to"] = "to is required"
		}
		if subj, ok := cfg["subject"].(string); ok && len(subj) > 255 {
			errs["subject"] = "must be 255 characters or fewer"
		}
		if body, ok := cfg["body"].(string); ok && len(body) > 4096 {
			errs["body"] = "must be 4096 characters or fewer"
		}

	case "time_based":
		tz, _ := cfg["timezone"].(string)
		if tz != "" {
			if _, err := time.LoadLocation(tz); err != nil {
				errs["timezone"] = fmt.Sprintf("invalid IANA timezone: %q", tz)
			}
		}
		
		defaultType, _ := cfg["default_type"].(string)
		defaultCfg, _ := cfg["default_config"].(map[string]any)
		defaultURL, _ := cfg["default_url"].(string)

		if defaultType != "" {
			if defaultType == "time_based" {
				errs["default_type"] = "cannot nest time_based routing"
			} else {
				if defaultCfg == nil {
					defaultCfg = map[string]any{}
				}
				res := ValidateDestinationConfig(defaultType, defaultCfg)
				if !res.OK {
					for k, v := range res.Errors {
						errs["default_config."+k] = v
					}
				}
			}
		} else if defaultURL != "" {
			if err := validateURL(defaultURL); err != nil {
				errs["default_url"] = err.Error()
			}
		} else {
			errs["default_destination"] = "must provide default_type/default_config or default_url"
		}

		rules, _ := cfg["rules"].([]any)
		if len(rules) == 0 {
			errs["rules"] = "at least one rule is required"
		}
		for i, r := range rules {
			rule, ok := r.(map[string]any)
			if !ok {
				errs[fmt.Sprintf("rules[%d]", i)] = "must be an object"
				continue
			}
			
			destType, _ := rule["destination_type"].(string)
			destCfg, _ := rule["destination_config"].(map[string]any)
			destURL, _ := rule["destination"].(string)

			if destType != "" {
				if destType == "time_based" {
					errs[fmt.Sprintf("rules[%d].destination_type", i)] = "cannot nest time_based routing"
				} else {
					if destCfg == nil {
						destCfg = map[string]any{}
					}
					res := ValidateDestinationConfig(destType, destCfg)
					if !res.OK {
						for k, v := range res.Errors {
							errs[fmt.Sprintf("rules[%d].destination_config.%s", i, k)] = v
						}
					}
				}
			} else if destURL != "" {
				if err := validateURL(destURL); err != nil {
					errs[fmt.Sprintf("rules[%d].destination", i)] = err.Error()
				}
			} else {
				errs[fmt.Sprintf("rules[%d].destination", i)] = "must provide destination_type/destination_config or destination"
			}

			for _, field := range []string{"start_time", "end_time"} {
				v, _ := rule[field].(string)
				if v == "" {
					errs[fmt.Sprintf("rules[%d].%s", i, field)] = "required (HH:MM)"
				} else {
					var h, m int
					if _, err := fmt.Sscanf(v, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
						errs[fmt.Sprintf("rules[%d].%s", i, field)] = fmt.Sprintf("invalid time %q, must be HH:MM (24h)", v)
					}
				}
			}
			
			days, _ := rule["days"].([]any)
			if len(days) == 0 {
				errs[fmt.Sprintf("rules[%d].days", i)] = "at least one day is required (0=Sun … 6=Sat)"
			} else {
				for j, d := range days {
					switch v := d.(type) {
					case float64:
						if int(v) < 0 || int(v) > 6 {
							errs[fmt.Sprintf("rules[%d].days[%d]", i, j)] = "day must be 0–6 (0=Sun)"
						}
					default:
						errs[fmt.Sprintf("rules[%d].days[%d]", i, j)] = "day must be an integer 0–6"
					}
				}
			}
		}

	default:
		return ValidationResult{OK: false, Errors: map[string]string{"destination_type": fmt.Sprintf("Unknown type: %s", destType)}}
	}

	if len(errs) > 0 {
		return ValidationResult{OK: false, Errors: errs}
	}
	return ValidationResult{OK: true}
}
