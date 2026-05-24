-- ============================================================================
-- MiniGaraj Scraper - Initial Schema
-- ============================================================================
-- Creates the scraper schema with jobs and raw_models tables.
-- This runs in the SCRAPER database (separate from minigaraj-api DB).
--
-- Author: Hakan Gunay
-- Date: 2026-04-04
-- ============================================================================

-- Create scraper schema
CREATE SCHEMA IF NOT EXISTS scraper;

-- ============================================================================
-- Job status enum
-- ============================================================================
DO $$ BEGIN
    CREATE TYPE scraper.job_status AS ENUM (
        'pending',
        'running',
        'paused',
        'completed',
        'failed'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ============================================================================
-- Raw model status enum
-- ============================================================================
DO $$ BEGIN
    CREATE TYPE scraper.model_status AS ENUM (
        'pending',
        'approved',
        'rejected',
        'duplicate',
        'imported'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ============================================================================
-- scraper.jobs - Crawl job tracking
-- ============================================================================
CREATE TABLE IF NOT EXISTS scraper.jobs (
    id              BIGSERIAL PRIMARY KEY,
    brand           VARCHAR(100)            NOT NULL,
    status          scraper.job_status      NOT NULL DEFAULT 'pending',
    source_urls     TEXT[]                  NOT NULL DEFAULT '{}',
    -- Progress tracking
    total_pages     INT                     NOT NULL DEFAULT 0,
    scraped_pages   INT                     NOT NULL DEFAULT 0,
    failed_pages    INT                     NOT NULL DEFAULT 0,
    total_models    INT                     NOT NULL DEFAULT 0,
    new_models      INT                     NOT NULL DEFAULT 0,
    duplicate_models INT                   NOT NULL DEFAULT 0,
    -- Error info
    error_message   TEXT,
    -- Timing
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_brand        ON scraper.jobs(brand);
CREATE INDEX IF NOT EXISTS idx_jobs_status       ON scraper.jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at   ON scraper.jobs(created_at DESC);

-- ============================================================================
-- scraper.raw_models - Scraped vehicle data (before import to catalog)
-- ============================================================================
CREATE TABLE IF NOT EXISTS scraper.raw_models (
    id                  BIGSERIAL PRIMARY KEY,
    job_id              BIGINT              REFERENCES scraper.jobs(id) ON DELETE SET NULL,

    -- Source info
    source_url          TEXT                NOT NULL,
    source_domain       VARCHAR(100)        NOT NULL,

    -- Parsed vehicle fields (mapped to catalog.models structure)
    brand               VARCHAR(100)        NOT NULL,
    name                VARCHAR(255),
    year                INT,
    series              VARCHAR(255),
    sub_series          VARCHAR(255),
    reference_number    VARCHAR(100),
    scale               VARCHAR(50),
    color               VARCHAR(100),
    material            VARCHAR(100),
    wheel_type          VARCHAR(100),
    origin              VARCHAR(100),
    description         TEXT,
    image_urls          TEXT[]              NOT NULL DEFAULT '{}',

    -- Complete raw scraped data (everything found on page)
    raw_data            JSONB               NOT NULL DEFAULT '{}',

    -- Review workflow
    status              scraper.model_status NOT NULL DEFAULT 'pending',
    catalog_model_id    BIGINT,             -- populated after import to catalog
    rejection_reason    TEXT,

    -- Duplicate detection
    content_hash        VARCHAR(64),        -- SHA256 of (brand+name+year+series)

    -- Timing
    scraped_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    reviewed_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_raw_models_job_id        ON scraper.raw_models(job_id);
CREATE INDEX IF NOT EXISTS idx_raw_models_brand         ON scraper.raw_models(brand);
CREATE INDEX IF NOT EXISTS idx_raw_models_status        ON scraper.raw_models(status);
CREATE INDEX IF NOT EXISTS idx_raw_models_year          ON scraper.raw_models(year);
CREATE INDEX IF NOT EXISTS idx_raw_models_content_hash  ON scraper.raw_models(content_hash);
CREATE INDEX IF NOT EXISTS idx_raw_models_created_at    ON scraper.raw_models(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_raw_models_raw_data_gin  ON scraper.raw_models USING GIN(raw_data);

-- ============================================================================
-- scraper.crawl_queue - URL queue for distributed crawling
-- ============================================================================
CREATE TABLE IF NOT EXISTS scraper.crawl_queue (
    id          BIGSERIAL PRIMARY KEY,
    job_id      BIGINT          NOT NULL REFERENCES scraper.jobs(id) ON DELETE CASCADE,
    url         TEXT            NOT NULL,
    depth       INT             NOT NULL DEFAULT 0,
    priority    INT             NOT NULL DEFAULT 0,
    status      VARCHAR(20)     NOT NULL DEFAULT 'pending',   -- pending, processing, done, failed
    attempts    INT             NOT NULL DEFAULT 0,
    error_msg   TEXT,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_crawl_queue_job_status   ON scraper.crawl_queue(job_id, status);
CREATE INDEX IF NOT EXISTS idx_crawl_queue_priority     ON scraper.crawl_queue(priority DESC, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_crawl_queue_url   ON scraper.crawl_queue(job_id, url);
