-- ============================================================================
-- MiniGaraj Scraper - Two-tier deduplication
-- ============================================================================
-- Splits content_hash into identity_hash (same model?) and content_hash
-- (data changed?). Allows updating existing records when new data arrives.
--
-- Author: Hakan Gunay
-- Date: 2026-04-04
-- ============================================================================

-- Add identity_hash column
ALTER TABLE scraper.raw_models ADD COLUMN IF NOT EXISTS identity_hash VARCHAR(64);

-- Populate identity_hash from existing content_hash for existing rows
UPDATE scraper.raw_models SET identity_hash = content_hash WHERE identity_hash IS NULL;

-- Create index on identity_hash
CREATE INDEX IF NOT EXISTS idx_raw_models_identity_hash ON scraper.raw_models(identity_hash);
