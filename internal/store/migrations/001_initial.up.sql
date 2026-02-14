CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE upstreams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    base_url        TEXT NOT NULL,
    api_key_encrypted TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    priority        INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE models (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL UNIQUE,
    display_name            TEXT,
    provider                TEXT NOT NULL DEFAULT 'openai',
    input_cost_per_million  NUMERIC(12,6) NOT NULL DEFAULT 0,
    output_cost_per_million NUMERIC(12,6) NOT NULL DEFAULT 0,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE llm_api_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash      TEXT NOT NULL UNIQUE,
    key_prefix    TEXT NOT NULL,
    name          TEXT NOT NULL DEFAULT '',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    rate_limit    INT,
    last_used_at  TIMESTAMPTZ,
    metadata      JSONB DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE management_api_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash      TEXT NOT NULL UNIQUE,
    key_prefix    TEXT NOT NULL,
    name          TEXT NOT NULL DEFAULT '',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    permissions   TEXT[] NOT NULL DEFAULT ARRAY['read'],
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE request_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    llm_key_id        UUID REFERENCES llm_api_keys(id),
    timestamp         TIMESTAMPTZ NOT NULL DEFAULT now(),
    method            TEXT NOT NULL,
    path              TEXT NOT NULL,
    model             TEXT,
    input_format      TEXT NOT NULL CHECK (input_format IN ('anthropic', 'openai')),
    upstream_id       UUID REFERENCES upstreams(id),
    status_code       INT,
    latency_ms        INT,
    input_tokens      INT,
    output_tokens     INT,
    cost              NUMERIC(12,8),
    error_message     TEXT,
    request_metadata  JSONB DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_request_logs_timestamp ON request_logs (timestamp DESC);
CREATE INDEX idx_request_logs_llm_key_id ON request_logs (llm_key_id);
CREATE INDEX idx_request_logs_model ON request_logs (model);
CREATE INDEX idx_request_logs_status_code ON request_logs (status_code);
CREATE INDEX idx_request_logs_input_format ON request_logs (input_format);
CREATE INDEX idx_llm_api_keys_key_hash ON llm_api_keys (key_hash);
CREATE INDEX idx_management_api_keys_key_hash ON management_api_keys (key_hash);
