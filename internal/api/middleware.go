// Author: Hakan Gunay
// Date: 2026-04-04
// HTTP middleware for authentication and request processing

package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// apiKeyAuth returns a middleware that validates the API key from
// Authorization header (Bearer <key>) or X-API-Key header.
// If apiKey is empty, authentication is disabled (development mode).
func apiKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if no API key is configured
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for health and metrics endpoints
			if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			provided := extractAPIKey(r)
			if provided == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "missing API key",
				})
				return
			}

			if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid API key",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractAPIKey(r *http.Request) string {
	// Check X-API-Key header first
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}

	// Check Authorization: Bearer <key>
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}
