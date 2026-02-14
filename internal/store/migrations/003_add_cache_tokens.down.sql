DROP INDEX IF EXISTS idx_request_logs_cache_read;

ALTER TABLE request_logs
DROP COLUMN IF EXISTS cache_creation_tokens,
DROP COLUMN IF EXISTS cache_read_tokens;
