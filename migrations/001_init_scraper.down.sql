-- ============================================================================
-- MiniGaraj Scraper - Rollback Initial Schema
-- ============================================================================

DROP TABLE IF EXISTS scraper.crawl_queue CASCADE;
DROP TABLE IF EXISTS scraper.raw_models CASCADE;
DROP TABLE IF EXISTS scraper.jobs CASCADE;
DROP TYPE IF EXISTS scraper.model_status;
DROP TYPE IF EXISTS scraper.job_status;
DROP SCHEMA IF EXISTS scraper CASCADE;
