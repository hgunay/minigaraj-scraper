// Author: Hakan Gunay
// Date: 2026-04-04
// Common helper functions shared across all brand crawlers

package shared

import (
	"net/url"
	"regexp"
	"strings"
)

// YearRegex matches years 1960-2029
var YearRegex = regexp.MustCompile(`\b(19[6-9]\d|20[0-2]\d)\b`)

// RefNumRegex matches reference numbers like HW1234, ABC12345Z
var RefNumRegex = regexp.MustCompile(`\b[A-Z]{1,3}\d{3,5}[A-Z]?\b`)

// StrPtr returns a pointer to s, or nil if s is empty
func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Contains checks if s contains any of the given substrings
func Contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// NormalizeScale ensures scale format is "1:64" not "1/64"
func NormalizeScale(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", ":")
	return s
}

// CleanImageURL removes Fandom image resizing parameters
func CleanImageURL(src string) string {
	if idx := strings.Index(src, "/revision/"); idx != -1 {
		return src[:idx]
	}
	return src
}

// ExtractWikiPageName extracts the page name from a fandom.com wiki URL
func ExtractWikiPageName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/wiki/")
	if len(parts) < 2 {
		return ""
	}
	name := parts[len(parts)-1]
	name = strings.ReplaceAll(name, "_", " ")
	name, _ = url.QueryUnescape(name)
	return strings.TrimSpace(name)
}
