// Author: Hakan Gunay
// Date: 2026-04-04
// Base crawler interface - all brand crawlers implement this

package crawler

import (
	"context"

	"minigaraj-scraper/internal/models"
)

// Crawler defines the contract for a brand-specific web crawler
type Crawler interface {
	// BrandName returns the brand this crawler handles (e.g., "Hot Wheels")
	BrandName() string

	// DefaultSeedURLs returns fallback seed URLs when the DB has none for this brand
	DefaultSeedURLs() []string

	// Crawl runs the crawl using the provided seed URLs and sends discovered
	// models to the output channel. It should respect context cancellation.
	Crawl(ctx context.Context, jobID int64, seedURLs []string, out chan<- *models.RawModel) error
}

// Result wraps a crawled model with metadata for the worker
type Result struct {
	Model       *models.RawModel
	IsDuplicate bool
	Err         error
}
