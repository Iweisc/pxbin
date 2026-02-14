DROP INDEX IF EXISTS idx_models_upstream_id;
ALTER TABLE models DROP COLUMN IF EXISTS upstream_id;
ALTER TABLE upstreams DROP COLUMN IF EXISTS format;
