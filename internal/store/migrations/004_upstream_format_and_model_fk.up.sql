-- Add format column to upstreams (openai or anthropic)
ALTER TABLE upstreams ADD COLUMN format TEXT NOT NULL DEFAULT 'openai'
  CHECK (format IN ('openai', 'anthropic'));

-- Add upstream_id FK to models
ALTER TABLE models ADD COLUMN upstream_id UUID REFERENCES upstreams(id);
CREATE INDEX idx_models_upstream_id ON models (upstream_id);
