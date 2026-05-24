package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuth_NoKeyConfigured(t *testing.T) {
	handler := apiKeyAuth("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when no API key configured, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_HealthSkipped(t *testing.T) {
	handler := apiKeyAuth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /health without key, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	handler := apiKeyAuth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when key missing, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	handler := apiKeyAuth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid key, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_ValidXAPIKey(t *testing.T) {
	handler := apiKeyAuth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	req.Header.Set("X-API-Key", "secret-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid X-API-Key, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_ValidBearerToken(t *testing.T) {
	handler := apiKeyAuth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid Bearer token, got %d", rec.Code)
	}
}
