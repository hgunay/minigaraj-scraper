package shared

import (
	"net/http"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{0, true},                              // network error
		{http.StatusTooManyRequests, true},      // 429
		{http.StatusInternalServerError, true},  // 500
		{http.StatusBadGateway, true},           // 502
		{http.StatusServiceUnavailable, true},   // 503
		{http.StatusGatewayTimeout, true},       // 504
		{http.StatusRequestTimeout, true},       // 408
		{http.StatusOK, false},                  // 200
		{http.StatusNotFound, false},            // 404
		{http.StatusForbidden, false},           // 403
		{http.StatusBadRequest, false},          // 400
	}

	for _, tt := range tests {
		got := isRetryable(tt.status)
		if got != tt.want {
			t.Errorf("isRetryable(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestBackoffDelay(t *testing.T) {
	base := 1 * time.Second
	max := 30 * time.Second

	// Attempt 1: ~1s (±25%)
	d1 := backoffDelay(1, base, max)
	if d1 < 750*time.Millisecond || d1 > 1250*time.Millisecond {
		t.Errorf("attempt 1: delay %v outside expected range [750ms, 1250ms]", d1)
	}

	// Attempt 3: ~4s (±25%)
	d3 := backoffDelay(3, base, max)
	if d3 < 3*time.Second || d3 > 5*time.Second {
		t.Errorf("attempt 3: delay %v outside expected range [3s, 5s]", d3)
	}

	// Should not exceed max
	d10 := backoffDelay(10, base, max)
	if d10 > max+max/4 {
		t.Errorf("attempt 10: delay %v exceeds max %v + jitter", d10, max)
	}
}

func TestRetryState(t *testing.T) {
	state := newRetryState()

	if got := state.increment("http://example.com"); got != 1 {
		t.Errorf("first increment = %d, want 1", got)
	}
	if got := state.increment("http://example.com"); got != 2 {
		t.Errorf("second increment = %d, want 2", got)
	}

	state.clear("http://example.com")
	if got := state.increment("http://example.com"); got != 1 {
		t.Errorf("after clear, increment = %d, want 1", got)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig(3)
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", cfg.MaxDelay)
	}
}
