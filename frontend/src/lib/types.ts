export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface OverviewStats {
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_read_tokens: number;
  cache_hit_rate: number;
  total_cost: number;
  error_count: number;
  error_rate: number;
  avg_latency_ms: number;
  avg_overhead_us: number;
}

export interface KeyStats {
  key_id: string;
  key_name: string;
  key_prefix: string;
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cost: number;
  error_count: number;
  avg_latency_ms: number;
}

export interface ModelStats {
  model: string;
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cost: number;
  error_count: number;
  avg_latency_ms: number;
}

export interface TimeSeriesBucket {
  timestamp: string;
  bucket?: string;  // Backend sends this as time.Time
  requests: number;
  input_tokens: number;
  output_tokens: number;
  cost: number;
  errors: number;
  avg_latency_ms: number;
  avg_overhead_us: number;
  p50_latency_ms?: number;
  p95_latency_ms?: number;
  p99_latency_ms?: number;
}

export interface LatencyStats {
  timestamp: string;
  p50: number;
  p95: number;
  p99: number;
  overhead_p50_us: number;
  overhead_p95_us: number;
  overhead_p99_us: number;
}

export interface RequestLog {
  id: string;
  llm_key_id: string | null;
  timestamp: string;
  method: string;
  path: string;
  model: string | null;
  input_format: "anthropic" | "openai";
  upstream_id: string | null;
  status_code: number | null;
  latency_ms: number | null;
  input_tokens: number | null;
  output_tokens: number | null;
  cost: number | null;
  overhead_us: number | null;
  error_message: string | null;
  request_metadata: Record<string, unknown>;
  created_at: string;
}

export interface LLMAPIKey {
  id: string;
  key_prefix: string;
  name: string;
  is_active: boolean;
  rate_limit: number | null;
  last_used_at: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ManagementAPIKey {
  id: string;
  key_prefix: string;
  name: string;
  is_active: boolean;
  permissions: string[];
  last_used_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface Model {
  id: string;
  name: string;
  display_name: string | null;
  provider: string;
  upstream_id: string | null;
  input_cost_per_million: number;
  output_cost_per_million: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Upstream {
  id: string;
  name: string;
  base_url: string;
  format: string;
  is_active: boolean;
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface CreateKeyRequest {
  name: string;
  rate_limit?: number | null;
  metadata?: Record<string, unknown>;
}

export interface CreateKeyResponse {
  id: string;
  key: string;
  key_prefix: string;
  name: string;
}

export interface CreateModelRequest {
  name: string;
  display_name?: string;
  provider: string;
  input_cost_per_million: number;
  output_cost_per_million: number;
}

export interface CreateUpstreamRequest {
  name: string;
  base_url: string;
  api_key: string;
  format?: string;
  priority?: number;
}

export interface DiscoveredModel {
  id: string;
  owned_by: string;
}

export interface DiscoverModelsRequest {
  upstream_id: string;
}

export interface ImportModelsRequest {
  upstream_id: string;
  models: {
    name: string;
    provider: string;
  }[];
}

export interface ImportModelsResponse {
  upstream: Upstream;
  models_created: number;
  models_skipped: number;
}

export interface HealthCheckResult {
  healthy: boolean;
  models_found: number;
  tested_model: string;
  latency_ms: number;
  error: string | null;
}

export type Period = "24h" | "7d" | "30d";
export type Interval = "5m" | "1h" | "1d";
