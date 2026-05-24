DROP INDEX IF EXISTS scraper.idx_raw_models_identity_hash;
ALTER TABLE scraper.raw_models DROP COLUMN IF EXISTS identity_hash;
