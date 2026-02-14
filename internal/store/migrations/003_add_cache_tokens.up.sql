ALTER TABLE request_logs
ADD COLUMN cache_creation_tokens INT DEFAULT 0,
ADD COLUMN cache_read_tokens INT DEFAULT 0;

CREATE INDEX idx_request_logs_cache_read ON request_logs (cache_read_tokens) WHERE cache_read_tokens > 0;
