// Author: Hakan Gunay
// Date: 2026-04-04
// Prometheus metrics for the scraper

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// PagesTotal counts total pages scraped per brand and status
	PagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scraper",
		Name:      "pages_total",
		Help:      "Total pages scraped",
	}, []string{"brand", "status"}) // status: success, error

	// ModelsTotal counts total models discovered per brand and status
	ModelsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scraper",
		Name:      "models_total",
		Help:      "Total models discovered",
	}, []string{"brand", "status"}) // status: new, duplicate, error

	// ActiveJobs tracks currently running crawl jobs
	ActiveJobs = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "scraper",
		Name:      "active_jobs",
		Help:      "Number of currently running crawl jobs",
	})

	// RequestDuration tracks HTTP request duration per domain
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "scraper",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"domain"})

	// ErrorsTotal counts errors per brand and type
	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scraper",
		Name:      "errors_total",
		Help:      "Total scraper errors",
	}, []string{"brand", "type"}) // type: network, parse, save

	// RetriesTotal counts retry attempts
	RetriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scraper",
		Name:      "retries_total",
		Help:      "Total retry attempts",
	}, []string{"domain", "status_code"})

	// JobDuration tracks total job duration per brand
	JobDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "scraper",
		Name:      "job_duration_seconds",
		Help:      "Total crawl job duration in seconds",
		Buckets:   []float64{10, 30, 60, 120, 300, 600, 1800, 3600},
	}, []string{"brand", "status"}) // status: completed, failed, cancelled
)
