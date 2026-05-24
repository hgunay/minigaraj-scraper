// Author: Hakan Gunay
// Date: 2026-04-04
// Domain models for scraper jobs and raw scraped data

package models

import (
	"encoding/json"
	"time"
)

// JobStatus represents the current state of a scrape job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// ModelStatus represents the review state of a raw scraped model
type ModelStatus string

const (
	ModelStatusPending   ModelStatus = "pending"
	ModelStatusApproved  ModelStatus = "approved"
	ModelStatusRejected  ModelStatus = "rejected"
	ModelStatusDuplicate ModelStatus = "duplicate"
	ModelStatusImported  ModelStatus = "imported"
)

// Job represents a scrape job in scraper.jobs
type Job struct {
	ID              int64      `db:"id"`
	Brand           string     `db:"brand"`
	Status          JobStatus  `db:"status"`
	SourceURLs      []string   `db:"source_urls"`
	TotalPages      int        `db:"total_pages"`
	ScrapedPages    int        `db:"scraped_pages"`
	FailedPages     int        `db:"failed_pages"`
	TotalModels     int        `db:"total_models"`
	NewModels       int        `db:"new_models"`
	DuplicateModels int        `db:"duplicate_models"`
	ErrorMessage    *string    `db:"error_message"`
	StartedAt       *time.Time `db:"started_at"`
	CompletedAt     *time.Time `db:"completed_at"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

// RawModel represents a single scraped vehicle in scraper.raw_models
type RawModel struct {
	ID              int64       `db:"id"`
	JobID           *int64      `db:"job_id"`
	SourceURL       string      `db:"source_url"`
	SourceDomain    string      `db:"source_domain"`
	Brand           string      `db:"brand"`
	Name            *string     `db:"name"`
	Year            *int        `db:"year"`
	Series          *string     `db:"series"`
	SubSeries       *string     `db:"sub_series"`
	ReferenceNumber *string     `db:"reference_number"`
	Scale           *string     `db:"scale"`
	Color           *string     `db:"color"`
	Material        *string     `db:"material"`
	WheelType       *string     `db:"wheel_type"`
	Origin          *string     `db:"origin"`
	Description     *string     `db:"description"`
	ImageURLs       []string    `db:"image_urls"`
	RawData         RawDataJSON `db:"raw_data"`
	Status          ModelStatus `db:"status"`
	CatalogModelID  *int64      `db:"catalog_model_id"`
	RejectionReason *string     `db:"rejection_reason"`
	IdentityHash    *string     `db:"identity_hash"`
	ContentHash     *string     `db:"content_hash"`
	ScrapedAt       time.Time   `db:"scraped_at"`
	ReviewedAt      *time.Time  `db:"reviewed_at"`
	CreatedAt       time.Time   `db:"created_at"`
	UpdatedAt       time.Time   `db:"updated_at"`
}

// RawDataJSON holds all raw key-value pairs collected from the source page.
// Stored as JSONB in PostgreSQL.
type RawDataJSON map[string]interface{}

func (r RawDataJSON) Value() ([]byte, error) {
	return json.Marshal(r)
}

// CrawlQueueItem represents a URL in the crawl queue
type CrawlQueueItem struct {
	ID          int64      `db:"id"`
	JobID       int64      `db:"job_id"`
	URL         string     `db:"url"`
	Depth       int        `db:"depth"`
	Priority    int        `db:"priority"`
	Status      string     `db:"status"`
	Attempts    int        `db:"attempts"`
	ErrorMsg    *string    `db:"error_msg"`
	CreatedAt   time.Time  `db:"created_at"`
	ProcessedAt *time.Time `db:"processed_at"`
}

// SeedURL represents a crawler starting point in scraper.seed_urls
type SeedURL struct {
	ID            int64      `db:"id" json:"id"`
	Brand         string     `db:"brand" json:"brand"`
	URL           string     `db:"url" json:"url"`
	Label         *string    `db:"label" json:"label"`
	Category      *string    `db:"category" json:"category"`
	IsActive      bool       `db:"is_active" json:"is_active"`
	Priority      int        `db:"priority" json:"priority"`
	LastCrawledAt *time.Time `db:"last_crawled_at" json:"last_crawled_at"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

// CreateSeedURLInput is the input for adding a new seed URL
type CreateSeedURLInput struct {
	Brand    string  `json:"brand"`
	URL      string  `json:"url"`
	Label    *string `json:"label"`
	Category *string `json:"category"`
	Priority int     `json:"priority"`
}

// CreateJobInput is the input for creating a new scrape job
type CreateJobInput struct {
	Brand      string
	SourceURLs []string
}

// JobStats is used for progress updates
type JobStats struct {
	JobID           int64
	TotalPages      int
	ScrapedPages    int
	FailedPages     int
	TotalModels     int
	NewModels       int
	DuplicateModels int
}
