package services

import (
	"net/http"
	"net/url"
)

// UTMParams holds all five standard UTM tracking parameters.
type UTMParams struct {
	Source   string
	Medium   string
	Campaign string
	Term     string
	Content  string
}

// IsEmpty returns true if none of the core UTM params are set.
func (u UTMParams) IsEmpty() bool {
	return u.Source == "" && u.Medium == "" && u.Campaign == ""
}

// ExtractUTM reads utm_* query parameters from an inbound scan request.
// Called at the start of RedirectHandler before any other processing.
func ExtractUTM(r *http.Request) UTMParams {
	q := r.URL.Query()
	return UTMParams{
		Source:   q.Get("utm_source"),
		Medium:   q.Get("utm_medium"),
		Campaign: q.Get("utm_campaign"),
		Term:     q.Get("utm_term"),
		Content:  q.Get("utm_content"),
	}
}

// MergeUTM combines the stored utm_config from the QR record with the UTM
// params present on the inbound scan URL. The stored config always wins —
// this lets clients pre-configure UTMs at QR creation time and have them
// apply to every scan regardless of what the scanner's URL contains.
func MergeUTM(stored map[string]any, inbound UTMParams) UTMParams {
	get := func(key1, key2, fallback string) string {
		if v, ok := stored[key1].(string); ok && v != "" {
			return v
		}
		if v, ok := stored[key2].(string); ok && v != "" {
			return v
		}
		return fallback
	}
	return UTMParams{
		Source:   get("source", "utm_source", inbound.Source),
		Medium:   get("medium", "utm_medium", inbound.Medium),
		Campaign: get("campaign", "utm_campaign", inbound.Campaign),
		Term:     get("term", "utm_term", inbound.Term),
		Content:  get("content", "utm_content", inbound.Content),
	}
}

// AppendUTMToURL appends non-empty UTM params to a destination URL.
// Existing query params on the destination are preserved.
// Only applied to destination_type = "url" — WhatsApp, vCard, and email
// destinations do not support query parameters.
func AppendUTMToURL(rawURL string, utm UTMParams) string {
	if utm.IsEmpty() {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	if utm.Source != "" {
		q.Set("utm_source", utm.Source)
	}
	if utm.Medium != "" {
		q.Set("utm_medium", utm.Medium)
	}
	if utm.Campaign != "" {
		q.Set("utm_campaign", utm.Campaign)
	}
	if utm.Term != "" {
		q.Set("utm_term", utm.Term)
	}
	if utm.Content != "" {
		q.Set("utm_content", utm.Content)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
