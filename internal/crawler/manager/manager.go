// Author: Hakan Gunay
// Date: 2026-04-04
// Crawler manager - orchestrates all brand crawlers

package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/crawler"
	"minigaraj-scraper/internal/crawler/hotwheels"
	"minigaraj-scraper/internal/crawler/matchbox"
	"minigaraj-scraper/internal/crawler/minigt"
	"minigaraj-scraper/internal/metrics"
	"minigaraj-scraper/internal/models"
	"minigaraj-scraper/internal/storage"

	"go.uber.org/zap"
)

// Manager orchestrates crawl jobs across all registered brand crawlers
type Manager struct {
	crawlers map[string]crawler.Crawler
	repo     *storage.Repository
	logger   *zap.Logger

	mu         sync.Mutex
	activeJobs map[int64]context.CancelFunc
}

// New creates a new Manager and registers all available crawlers
func New(cfg config.CrawlerConfig, repo *storage.Repository, logger *zap.Logger) *Manager {
	m := &Manager{
		crawlers:   make(map[string]crawler.Crawler),
		repo:       repo,
		logger:     logger,
		activeJobs: make(map[int64]context.CancelFunc),
	}

	// Register all brand crawlers here
	m.register(hotwheels.New(cfg, logger))
	m.register(matchbox.New(cfg, logger))
	m.register(minigt.New(cfg, logger))
	// Future: m.register(majorette.New(cfg, logger))
	// Future: m.register(tarmacworks.New(cfg, logger))
	// Future: m.register(inno64.New(cfg, logger))

	return m
}

// register adds a crawler to the registry
func (m *Manager) register(c crawler.Crawler) {
	m.crawlers[c.BrandName()] = c
}

// AvailableBrands returns all registered brand names
func (m *Manager) AvailableBrands() []string {
	brands := make([]string, 0, len(m.crawlers))
	for b := range m.crawlers {
		brands = append(brands, b)
	}
	return brands
}

// StartJob creates and runs a scrape job for the given brand
func (m *Manager) StartJob(ctx context.Context, brand string) (int64, error) {
	c, ok := m.crawlers[brand]
	if !ok {
		return 0, fmt.Errorf("no crawler registered for brand: %s", brand)
	}

	// Load seed URLs from DB, fall back to crawler defaults
	seeds, err := m.repo.GetActiveSeedURLs(ctx, brand)
	if err != nil {
		m.logger.Warn("failed to load seed URLs from DB, using defaults",
			zap.String("brand", brand), zap.Error(err))
	}
	var seedURLs []string
	if len(seeds) > 0 {
		for _, s := range seeds {
			seedURLs = append(seedURLs, s.URL)
		}
	} else {
		seedURLs = c.DefaultSeedURLs()
	}

	// Create job record
	jobID, err := m.repo.CreateJob(ctx, models.CreateJobInput{
		Brand:      brand,
		SourceURLs: seedURLs,
	})
	if err != nil {
		return 0, fmt.Errorf("create job: %w", err)
	}

	// Create a cancellable context for this job
	jobCtx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.activeJobs[jobID] = cancel
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.activeJobs, jobID)
			m.mu.Unlock()
		}()
		m.runJob(jobCtx, jobID, seedURLs, c)
	}()

	m.logger.Info("job started",
		zap.Int64("job_id", jobID),
		zap.String("brand", brand),
	)

	return jobID, nil
}

// CancelJob cancels a running job by its ID
func (m *Manager) CancelJob(jobID int64) error {
	m.mu.Lock()
	cancel, ok := m.activeJobs[jobID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("job %d is not active", jobID)
	}

	cancel()
	m.logger.Info("job cancelled", zap.Int64("job_id", jobID))
	return nil
}

// Shutdown cancels all active jobs and waits briefly for them to finish
func (m *Manager) Shutdown() {
	m.mu.Lock()
	for jobID, cancel := range m.activeJobs {
		m.logger.Info("shutting down job", zap.Int64("job_id", jobID))
		cancel()
	}
	m.mu.Unlock()
}

// runJob executes a crawl job, saving results to the database
func (m *Manager) runJob(ctx context.Context, jobID int64, seedURLs []string, c crawler.Crawler) {
	brandName := c.BrandName()
	startTime := time.Now()
	metrics.ActiveJobs.Inc()
	defer metrics.ActiveJobs.Dec()

	if err := m.repo.StartJob(ctx, jobID); err != nil {
		m.logger.Error("failed to start job", zap.Int64("job_id", jobID), zap.Error(err))
		return
	}

	out := make(chan *models.RawModel, 100)
	var stats jobStats

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.consumeModels(ctx, jobID, brandName, out, &stats)
	}()

	crawlErr := c.Crawl(ctx, jobID, seedURLs, out)
	close(out)
	wg.Wait()

	duration := time.Since(startTime).Seconds()

	// Check if the job was cancelled
	if ctx.Err() != nil {
		m.logger.Info("job was cancelled", zap.Int64("job_id", jobID))
		_ = m.repo.FailJob(context.Background(), jobID, "cancelled by user")
		metrics.JobDuration.WithLabelValues(brandName, "cancelled").Observe(duration)
		return
	}

	if crawlErr != nil {
		m.logger.Error("crawl failed",
			zap.Int64("job_id", jobID),
			zap.Error(crawlErr),
		)
		_ = m.repo.FailJob(context.Background(), jobID, crawlErr.Error())
		metrics.JobDuration.WithLabelValues(brandName, "failed").Observe(duration)
		metrics.ErrorsTotal.WithLabelValues(brandName, "crawl").Inc()
		return
	}

	_ = m.repo.UpdateJobStats(context.Background(), models.JobStats{
		JobID:           jobID,
		TotalModels:     stats.newModels + stats.dupes,
		NewModels:       stats.newModels,
		DuplicateModels: stats.dupes,
		ScrapedPages:    stats.pages,
	})

	_ = m.repo.CompleteJob(context.Background(), jobID)
	_ = m.repo.UpdateSeedURLLastCrawled(context.Background(), brandName, seedURLs)
	metrics.JobDuration.WithLabelValues(brandName, "completed").Observe(duration)

	m.logger.Info("job completed",
		zap.Int64("job_id", jobID),
		zap.Int("new_models", stats.newModels),
		zap.Int("duplicates", stats.dupes),
	)
}

// consumeModels drains the output channel and saves models to DB
func (m *Manager) consumeModels(ctx context.Context, jobID int64, brand string, out <-chan *models.RawModel, stats *jobStats) {
	for model := range out {
		if model == nil {
			continue
		}

		_, isDup, err := m.repo.SaveRawModel(ctx, model)
		if err != nil {
			m.logger.Warn("save raw model failed",
				zap.String("url", model.SourceURL),
				zap.Error(err),
			)
			metrics.ErrorsTotal.WithLabelValues(brand, "save").Inc()
			continue
		}

		if isDup {
			stats.dupes++
			metrics.ModelsTotal.WithLabelValues(brand, "duplicate").Inc()
		} else {
			stats.newModels++
			metrics.ModelsTotal.WithLabelValues(brand, "new").Inc()
		}
		stats.pages++
		metrics.PagesTotal.WithLabelValues(brand, "success").Inc()

		if (stats.newModels+stats.dupes)%100 == 0 {
			m.logger.Info("crawl progress",
				zap.Int64("job_id", jobID),
				zap.Int("new", stats.newModels),
				zap.Int("dupes", stats.dupes),
			)
		}
	}
}

type jobStats struct {
	pages     int
	newModels int
	dupes     int
}
