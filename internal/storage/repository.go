// Author: Hakan Gunay
// Date: 2026-04-04
// Storage repository - all DB operations for scraper schema

package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"minigaraj-scraper/internal/models"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// Repository handles all database operations for the scraper
type Repository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// New creates a new Repository
func New(db *sqlx.DB, logger *zap.Logger) *Repository {
	return &Repository{db: db, logger: logger}
}

// Ping checks database connectivity
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// ============================================================================
// JOB OPERATIONS
// ============================================================================

// CreateJob inserts a new scrape job and returns its ID
func (r *Repository) CreateJob(ctx context.Context, input models.CreateJobInput) (int64, error) {
	const q = `
		INSERT INTO scraper.jobs (brand, status, source_urls)
		VALUES ($1, 'pending', $2)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q, input.Brand, pq.Array(input.SourceURLs)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create job: %w", err)
	}
	return id, nil
}

// GetJob retrieves a job by ID
func (r *Repository) GetJob(ctx context.Context, id int64) (*models.Job, error) {
	const q = `SELECT * FROM scraper.jobs WHERE id = $1`
	var job models.Job
	if err := r.db.GetContext(ctx, &job, q, id); err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &job, nil
}

// StartJob marks a job as running and sets started_at
func (r *Repository) StartJob(ctx context.Context, id int64) error {
	const q = `
		UPDATE scraper.jobs
		SET status = 'running', started_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// CompleteJob marks a job as completed
func (r *Repository) CompleteJob(ctx context.Context, id int64) error {
	const q = `
		UPDATE scraper.jobs
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// FailJob marks a job as failed with an error message
func (r *Repository) FailJob(ctx context.Context, id int64, errMsg string) error {
	const q = `
		UPDATE scraper.jobs
		SET status = 'failed', error_message = $2, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id, errMsg)
	return err
}

// UpdateJobStats updates progress counters for a job
func (r *Repository) UpdateJobStats(ctx context.Context, stats models.JobStats) error {
	const q = `
		UPDATE scraper.jobs SET
			total_pages      = $2,
			scraped_pages    = $3,
			failed_pages     = $4,
			total_models     = $5,
			new_models       = $6,
			duplicate_models = $7,
			updated_at       = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q,
		stats.JobID,
		stats.TotalPages,
		stats.ScrapedPages,
		stats.FailedPages,
		stats.TotalModels,
		stats.NewModels,
		stats.DuplicateModels,
	)
	return err
}

// IncrementJobCounters atomically increments job counters
func (r *Repository) IncrementJobCounters(ctx context.Context, jobID int64, scraped, newModels, dupes, failed int) error {
	const q = `
		UPDATE scraper.jobs SET
			scraped_pages    = scraped_pages + $2,
			new_models       = new_models + $3,
			duplicate_models = duplicate_models + $4,
			total_models     = total_models + $3 + $4,
			failed_pages     = failed_pages + $5,
			updated_at       = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, jobID, scraped, newModels, dupes, failed)
	return err
}

// ListJobs returns all jobs ordered by creation date (newest first)
func (r *Repository) ListJobs(ctx context.Context, limit, offset int) ([]models.Job, error) {
	const q = `SELECT * FROM scraper.jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	var jobs []models.Job
	if err := r.db.SelectContext(ctx, &jobs, q, limit, offset); err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return jobs, nil
}

// ============================================================================
// RAW MODEL OPERATIONS
// ============================================================================

// contentHash generates a deduplication hash for a model
// identityHash determines if two records represent the same logical model.
// Uses: brand + name + year + series + reference_number
func identityHash(brand, name string, year *int, series, refNumber *string) string {
	yearStr := ""
	if year != nil {
		yearStr = fmt.Sprintf("%d", *year)
	}
	key := strings.ToLower(fmt.Sprintf("%s|%s|%s|%s|%s",
		brand, name, yearStr, deref(series), deref(refNumber)))
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}

// fullContentHash captures all parsed fields — detects data changes for the same model.
func fullContentHash(m *models.RawModel) string {
	key := strings.ToLower(fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		m.Brand, deref(m.Name), intStr(m.Year), deref(m.Series),
		deref(m.SubSeries), deref(m.ReferenceNumber), deref(m.Scale),
		deref(m.Color), deref(m.Material), deref(m.WheelType),
		deref(m.Origin), deref(m.Description)))
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}

// SaveRawModel inserts or updates a scraped model using two-tier deduplication.
// - Identity match + same content → skip (true duplicate)
// - Identity match + different content → update existing record
// - No identity match → insert new record
// Returns (id, isDuplicate, error)
func (r *Repository) SaveRawModel(ctx context.Context, m *models.RawModel) (int64, bool, error) {
	idHash := identityHash(m.Brand, deref(m.Name), m.Year, m.Series, m.ReferenceNumber)
	cHash := fullContentHash(m)
	m.IdentityHash = &idHash
	m.ContentHash = &cHash

	// Check for existing record with same identity
	var existingID int64
	var existingContentHash string
	const checkQ = `SELECT id, COALESCE(content_hash, '') FROM scraper.raw_models WHERE identity_hash = $1 LIMIT 1`
	err := r.db.QueryRowContext(ctx, checkQ, idHash).Scan(&existingID, &existingContentHash)

	if err == nil && existingID > 0 {
		// Identity match found
		if existingContentHash == cHash {
			// Same content — true duplicate, skip
			return existingID, true, nil
		}

		// Content changed — update existing record with new data
		rawJSON, err := json.Marshal(m.RawData)
		if err != nil {
			return 0, false, fmt.Errorf("marshal raw_data: %w", err)
		}

		const updateQ = `
			UPDATE scraper.raw_models SET
				job_id = $2, source_url = $3, source_domain = $4,
				name = $5, year = $6, series = $7, sub_series = $8,
				reference_number = $9, scale = $10, color = $11, material = $12,
				wheel_type = $13, origin = $14, description = $15,
				image_urls = $16, raw_data = $17, content_hash = $18,
				scraped_at = NOW(), updated_at = NOW()
			WHERE id = $1`
		_, err = r.db.ExecContext(ctx, updateQ,
			existingID,
			m.JobID, m.SourceURL, m.SourceDomain,
			m.Name, m.Year, m.Series, m.SubSeries,
			m.ReferenceNumber, m.Scale, m.Color, m.Material,
			m.WheelType, m.Origin, m.Description,
			pq.Array(m.ImageURLs), rawJSON, cHash,
		)
		if err != nil {
			return 0, false, fmt.Errorf("update raw_model: %w", err)
		}

		r.logger.Debug("model updated with new content",
			zap.Int64("id", existingID),
			zap.String("name", deref(m.Name)),
		)
		return existingID, false, nil
	}

	// No identity match — insert new record
	rawJSON, err := json.Marshal(m.RawData)
	if err != nil {
		return 0, false, fmt.Errorf("marshal raw_data: %w", err)
	}

	const insertQ = `
		INSERT INTO scraper.raw_models (
			job_id, source_url, source_domain,
			brand, name, year, series, sub_series,
			reference_number, scale, color, material,
			wheel_type, origin, description,
			image_urls, raw_data, identity_hash, content_hash,
			status, scraped_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19,
			'pending', NOW()
		) RETURNING id`

	var id int64
	err = r.db.QueryRowContext(ctx, insertQ,
		m.JobID, m.SourceURL, m.SourceDomain,
		m.Brand, m.Name, m.Year, m.Series, m.SubSeries,
		m.ReferenceNumber, m.Scale, m.Color, m.Material,
		m.WheelType, m.Origin, m.Description,
		pq.Array(m.ImageURLs), rawJSON, idHash, cHash,
	).Scan(&id)
	if err != nil {
		return 0, false, fmt.Errorf("insert raw_model: %w", err)
	}

	return id, false, nil
}

func intStr(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

// ListRawModels returns raw models with optional filters
func (r *Repository) ListRawModels(ctx context.Context, filter RawModelFilter) ([]models.RawModel, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	i := 1

	if filter.Brand != "" {
		where = append(where, fmt.Sprintf("brand = $%d", i))
		args = append(args, filter.Brand)
		i++
	}
	if filter.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", i))
		args = append(args, filter.Status)
		i++
	}
	if filter.JobID != 0 {
		where = append(where, fmt.Sprintf("job_id = $%d", i))
		args = append(args, filter.JobID)
		i++
	}
	if filter.Year != 0 {
		where = append(where, fmt.Sprintf("year = $%d", i))
		args = append(args, filter.Year)
		i++
	}

	whereClause := strings.Join(where, " AND ")

	// Count
	var total int64
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM scraper.raw_models WHERE %s`, whereClause)
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data
	listQ := fmt.Sprintf(`
		SELECT * FROM scraper.raw_models
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, i, i+1)
	args = append(args, filter.Limit, filter.Offset)

	var items []models.RawModel
	if err := r.db.SelectContext(ctx, &items, listQ, args...); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// ApproveRawModel marks a model as approved and sets reviewed_at
func (r *Repository) ApproveRawModel(ctx context.Context, id int64) error {
	now := time.Now()
	const q = `
		UPDATE scraper.raw_models
		SET status = 'approved', reviewed_at = $2, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id, now)
	return err
}

// RejectRawModel marks a model as rejected
func (r *Repository) RejectRawModel(ctx context.Context, id int64, reason string) error {
	now := time.Now()
	const q = `
		UPDATE scraper.raw_models
		SET status = 'rejected', rejection_reason = $2, reviewed_at = $3, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id, reason, now)
	return err
}

// MarkImported marks a model as imported and stores the catalog model ID
func (r *Repository) MarkImported(ctx context.Context, id int64, catalogModelID int64) error {
	const q = `
		UPDATE scraper.raw_models
		SET status = 'imported', catalog_model_id = $2, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id, catalogModelID)
	return err
}

// ============================================================================
// CRAWL QUEUE OPERATIONS
// ============================================================================

// EnqueueURL adds a URL to the crawl queue (ignores duplicates)
func (r *Repository) EnqueueURL(ctx context.Context, jobID int64, url string, depth, priority int) error {
	const q = `
		INSERT INTO scraper.crawl_queue (job_id, url, depth, priority, status)
		VALUES ($1, $2, $3, $4, 'pending')
		ON CONFLICT (job_id, url) DO NOTHING`
	_, err := r.db.ExecContext(ctx, q, jobID, url, depth, priority)
	return err
}

// ============================================================================
// SEED URL OPERATIONS
// ============================================================================

// GetActiveSeedURLs returns active seed URLs for a brand, ordered by priority (desc)
func (r *Repository) GetActiveSeedURLs(ctx context.Context, brand string) ([]models.SeedURL, error) {
	const q = `
		SELECT * FROM scraper.seed_urls
		WHERE brand = $1 AND is_active = true
		ORDER BY priority DESC, created_at`
	var seeds []models.SeedURL
	if err := r.db.SelectContext(ctx, &seeds, q, brand); err != nil {
		return nil, fmt.Errorf("get seed urls: %w", err)
	}
	return seeds, nil
}

// ListSeedURLs returns all seed URLs with optional brand filter
func (r *Repository) ListSeedURLs(ctx context.Context, brand string) ([]models.SeedURL, error) {
	if brand != "" {
		const q = `SELECT * FROM scraper.seed_urls WHERE brand = $1 ORDER BY brand, priority DESC, created_at`
		var seeds []models.SeedURL
		if err := r.db.SelectContext(ctx, &seeds, q, brand); err != nil {
			return nil, fmt.Errorf("list seed urls: %w", err)
		}
		return seeds, nil
	}

	const q = `SELECT * FROM scraper.seed_urls ORDER BY brand, priority DESC, created_at`
	var seeds []models.SeedURL
	if err := r.db.SelectContext(ctx, &seeds, q); err != nil {
		return nil, fmt.Errorf("list seed urls: %w", err)
	}
	return seeds, nil
}

// CreateSeedURL inserts a new seed URL
func (r *Repository) CreateSeedURL(ctx context.Context, input models.CreateSeedURLInput) (int64, error) {
	const q = `
		INSERT INTO scraper.seed_urls (brand, url, label, category, priority)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	var id int64
	err := r.db.QueryRowContext(ctx, q, input.Brand, input.URL, input.Label, input.Category, input.Priority).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create seed url: %w", err)
	}
	return id, nil
}

// UpsertSeedURL inserts or updates a seed URL (used by auto-discovery)
func (r *Repository) UpsertSeedURL(ctx context.Context, input models.CreateSeedURLInput) error {
	const q = `
		INSERT INTO scraper.seed_urls (brand, url, label, category, priority)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (brand, url) DO UPDATE SET
			label = COALESCE(EXCLUDED.label, scraper.seed_urls.label),
			category = COALESCE(EXCLUDED.category, scraper.seed_urls.category)`
	_, err := r.db.ExecContext(ctx, q, input.Brand, input.URL, input.Label, input.Category, input.Priority)
	return err
}

// ToggleSeedURL activates or deactivates a seed URL
func (r *Repository) ToggleSeedURL(ctx context.Context, id int64, isActive bool) error {
	const q = `UPDATE scraper.seed_urls SET is_active = $2 WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id, isActive)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("seed url %d not found", id)
	}
	return nil
}

// DeleteSeedURL removes a seed URL
func (r *Repository) DeleteSeedURL(ctx context.Context, id int64) error {
	const q = `DELETE FROM scraper.seed_urls WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("seed url %d not found", id)
	}
	return nil
}

// UpdateSeedURLLastCrawled updates the last_crawled_at timestamp
func (r *Repository) UpdateSeedURLLastCrawled(ctx context.Context, brand string, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	const q = `
		UPDATE scraper.seed_urls
		SET last_crawled_at = NOW()
		WHERE brand = $1 AND url = ANY($2)`
	_, err := r.db.ExecContext(ctx, q, brand, pq.Array(urls))
	return err
}

// ============================================================================
// HELPERS
// ============================================================================

// RawModelFilter defines optional filters for listing raw models
type RawModelFilter struct {
	Brand  string
	Status string
	JobID  int64
	Year   int
	Limit  int
	Offset int
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
