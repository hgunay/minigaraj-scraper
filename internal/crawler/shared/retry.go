// Author: Hakan Gunay
// Date: 2026-04-04
// Retry middleware for colly collectors — exponential backoff with jitter

package shared

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"minigaraj-scraper/internal/metrics"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// RetryConfig holds retry middleware configuration
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig(maxRetries int) RetryConfig {
	return RetryConfig{
		MaxRetries: maxRetries,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

// retryState tracks per-URL retry attempts
type retryState struct {
	mu       sync.Mutex
	attempts map[string]int
}

func newRetryState() *retryState {
	return &retryState{attempts: make(map[string]int)}
}

func (rs *retryState) increment(url string) int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.attempts[url]++
	return rs.attempts[url]
}

func (rs *retryState) clear(url string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.attempts, url)
}

// AttachRetry adds retry-on-error behavior to a colly collector.
// It retries on 429 (rate limit), 5xx (server errors), and network errors.
func AttachRetry(col *colly.Collector, cfg RetryConfig, logger *zap.Logger) {
	if cfg.MaxRetries <= 0 {
		return
	}

	state := newRetryState()

	// Clear state on successful requests
	col.OnResponse(func(r *colly.Response) {
		state.clear(r.Request.URL.String())
	})

	col.OnError(func(r *colly.Response, err error) {
		url := r.Request.URL.String()
		attempt := state.increment(url)

		if attempt > cfg.MaxRetries {
			logger.Warn("max retries exceeded",
				zap.String("url", url),
				zap.Int("attempts", attempt),
				zap.Error(err),
			)
			state.clear(url)
			return
		}

		if !isRetryable(r.StatusCode) {
			state.clear(url)
			return
		}

		delay := backoffDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)

		logger.Debug("retrying request",
			zap.String("url", url),
			zap.Int("attempt", attempt),
			zap.Int("status", r.StatusCode),
			zap.Duration("delay", delay),
		)

		metrics.RetriesTotal.WithLabelValues(
			r.Request.URL.Hostname(),
			fmt.Sprintf("%d", r.StatusCode),
		).Inc()

		time.Sleep(delay)
		_ = r.Request.Retry()
	})
}

// isRetryable returns true for status codes that warrant a retry
func isRetryable(statusCode int) bool {
	switch {
	case statusCode == 0:
		// Network error (timeout, DNS, connection refused)
		return true
	case statusCode == http.StatusTooManyRequests: // 429
		return true
	case statusCode >= 500 && statusCode < 600:
		// Server errors (500, 502, 503, 504)
		return true
	case statusCode == http.StatusRequestTimeout: // 408
		return true
	default:
		return false
	}
}

// backoffDelay calculates exponential backoff with jitter
func backoffDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Exponential: baseDelay * 2^(attempt-1)
	exp := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(baseDelay) * exp)

	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter: ±25%
	jitter := time.Duration(float64(delay) * (0.5*rand.Float64() - 0.25))
	delay += jitter

	if delay < 0 {
		delay = baseDelay
	}

	return delay
}
