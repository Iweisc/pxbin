# pxbin

A high-performance LLM protocol translation proxy. Routes requests between Anthropic and OpenAI API formats, tracks usage and costs, and provides a management dashboard.

Built for use cases like Claude Code talking to OpenAI-compatible upstreams — pxbin sits in the middle, translates the protocol, and logs everything.

## Features

- **Protocol translation** — Anthropic API to/from OpenAI-compatible format, including streaming (SSE)
- **Multi-upstream routing** — Configure multiple upstream providers with per-model routing and priority
- **Tool use / function calling** — Full translation of tool definitions and tool results between formats
- **Extended thinking** — Anthropic extended thinking blocks are preserved through translation
- **Prompt caching** — Cache control hints are translated; cache read/creation tokens are tracked
- **Cost tracking** — Per-model input/output pricing with automatic cost calculation on every request
- **Async request logging** — Buffered channel (10k capacity) with batch inserts every 500ms
- **API key management** — Two key types: LLM keys (`pxb_`) for proxy access, management keys (`pxm_`) for admin
- **Dashboard** — React frontend for logs, costs, keys, models, and upstream management
- **In-memory caching** — Model and auth key caches with TTL to eliminate per-request DB overhead

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 17
- Node.js (for frontend development)

### 1. Start PostgreSQL

```bash
docker compose up -d postgres
```

### 2. Configure

Create a `config.yaml` or use environment variables (`PXBIN_` prefix):

```yaml
# config.yaml
listen_addr: ":8080"
database_url: "postgres://pxbin:pxbin@localhost:5432/pxbin?sslmode=disable"
database_schema: "public"  # set to a dedicated schema when sharing a PG cluster
management_bootstrap_key: ""  # set via PXBIN_MANAGEMENT_BOOTSTRAP_KEY env var
cors_origins:
  - "http://localhost:5173"
```

### 3. Build and run

```bash
make build
./bin/pxbin
```

Migrations run automatically on startup.

### 4. Bootstrap

Set the `PXBIN_MANAGEMENT_BOOTSTRAP_KEY` env var, then use it to create your first management key:

```bash
curl -X POST http://localhost:8080/api/v1/bootstrap \
  -H "Content-Type: application/json" \
  -d '{"bootstrap_key": "your-bootstrap-secret", "name": "admin"}'
```

### 5. Add an upstream

Use the management key (`pxm_...`) to register an upstream provider:

```bash
curl -X POST http://localhost:8080/api/v1/upstreams \
  -H "x-api-key: pxm_..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "openai",
    "base_url": "https://api.openai.com",
    "api_key": "sk-...",
    "format": "openai"
  }'
```

### 6. Import models

Discover and import models from your upstream:

```bash
# Discover available models
curl -X POST http://localhost:8080/api/v1/models/discover \
  -H "x-api-key: pxm_..." \
  -H "Content-Type: application/json" \
  -d '{"upstream_id": "<upstream-uuid>"}'

# Import them
curl -X POST http://localhost:8080/api/v1/models/import \
  -H "x-api-key: pxm_..." \
  -H "Content-Type: application/json" \
  -d '{"upstream_id": "<upstream-uuid>", "models": ["gpt-4o", "gpt-4o-mini"]}'
```

### 7. Create an LLM key

```bash
curl -X POST http://localhost:8080/api/v1/keys \
  -H "x-api-key: pxm_..." \
  -H "Content-Type: application/json" \
  -d '{"type": "llm", "name": "my-llm-key"}'
```

### 8. Use the proxy

Point your client at pxbin. For Claude Code:

```bash
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_API_KEY=pxb_...
```

All upstream credentials are stored in the database — never in config files.

## API

### Proxy Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/messages` | `pxb_*` | Anthropic-format messages |
| `POST` | `/v1/chat/completions` | `pxb_*` | OpenAI-format chat completions |
| `GET` | `/health` | none | Health check |

Authentication via `Authorization: Bearer <key>` or `x-api-key` header.

### Management Endpoints

All require a `pxm_*` management key.

| Method | Path | Description |
|--------|------|-------------|
| `GET/POST` | `/api/v1/keys` | List / create API keys |
| `PATCH/DELETE` | `/api/v1/keys/{id}` | Update / deactivate key |
| `GET/POST` | `/api/v1/models` | List / create models |
| `PATCH/DELETE` | `/api/v1/models/{id}` | Update / delete model |
| `POST` | `/api/v1/models/discover` | Discover models from upstream |
| `POST` | `/api/v1/models/import` | Import discovered models |
| `POST` | `/api/v1/models/sync-pricing` | Sync pricing from upstream |
| `GET/POST` | `/api/v1/upstreams` | List / create upstreams |
| `PATCH/DELETE` | `/api/v1/upstreams/{id}` | Update / delete upstream |
| `GET` | `/api/v1/stats/overview` | Usage overview (period: 24h, 7d, 30d) |
| `GET` | `/api/v1/stats/by-key` | Stats grouped by API key |
| `GET` | `/api/v1/stats/by-model` | Stats grouped by model |
| `GET` | `/api/v1/stats/timeseries` | Time series data |
| `GET` | `/api/v1/stats/latency` | Latency percentiles (p50, p95, p99) |
| `GET` | `/api/v1/logs` | Request logs with filtering |
| `POST` | `/api/v1/bootstrap` | Create initial key (requires bootstrap key) |

## Configuration

| Field | Env Var | Default | Description |
|-------|---------|---------|-------------|
| `listen_addr` | `PXBIN_LISTEN_ADDR` | `:8080` | HTTP listen address |
| `database_url` | `PXBIN_DATABASE_URL` | — | PostgreSQL connection string |
| `database_schema` | `PXBIN_DATABASE_SCHEMA` | `public` | Schema used for all pxbin tables/migrations |
| `log_buffer_size` | `PXBIN_LOG_BUFFER_SIZE` | `10000` | Async log buffer capacity |
| `management_bootstrap_key` | `PXBIN_MANAGEMENT_BOOTSTRAP_KEY` | — | Bootstrap key for initial setup |
| `cors_origins` | `PXBIN_CORS_ORIGINS` | — | Comma-separated allowed origins |
| `encryption_key` | `PXBIN_ENCRYPTION_KEY` | — | AES-256 key for upstream API key encryption |

Upstream providers and their API keys are managed exclusively through the management API and stored in the database.

### Running On A Shared PG17 Cluster

If pxbin shares a production PostgreSQL cluster with other apps, set a dedicated schema so pxbin migrations and unique constraints stay isolated:

```bash
export PXBIN_DATABASE_URL="postgres://user:pass@pg-primary,pg-replica/appdb?sslmode=require&target_session_attrs=read-write"
export PXBIN_DATABASE_SCHEMA="pxbin"
```

The pxbin role must be able to use (or create) the configured schema.

## Frontend

The dashboard is a React SPA at `frontend/`.

```bash
cd frontend
npm install
npm run dev    # Dev server on :5173, proxies API to :8080
npm run build  # Production build to dist/
```

Pages: Dashboard, Logs, Costs, Keys, Models, Upstreams.

## Development

```bash
make dev             # Run backend with go run
make test            # Run tests with -race
make lint            # golangci-lint
make frontend-dev    # Vite dev server
make frontend-build  # Production frontend build
```

## Docker

```bash
docker compose up
```

Starts pxbin and PostgreSQL 17. The backend is available on `:8080`.

## Architecture

```
Client (Claude Code, etc.)
  │
  ▼
pxbin proxy (:8080)
  ├── /v1/messages         → Translates to upstream format → Upstream Provider
  ├── /v1/chat/completions → Routes to upstream            → Upstream Provider
  ├── /api/v1/*            → Management API
  └── Async logger ───────→ PostgreSQL (request_logs)
```

Key design decisions:
- **net/http + chi** over heavier frameworks for minimal proxy overhead
- **Async logging** via buffered channel — proxy responses are never blocked by DB writes
- **In-memory caches** with TTL for model lookups and auth key validation
- **Connection pooling** (25 max connections) via pgxpool
- **SHA-256 key hashing** — plaintext keys are never stored
- **No secrets in config** — upstream API keys are managed via the API and stored in the database

## License

All rights reserved.
