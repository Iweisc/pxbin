import type {
  OverviewStats,
  KeyStats,
  ModelStats,
  TimeSeriesBucket,
  LatencyStats,
  PaginatedResponse,
  Period,
  Interval,
} from "./types.ts";

function getAuthToken(): string | null {
  return localStorage.getItem("pxbin_api_key");
}

class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

interface ApiResponse<T> {
  data: T;
  meta?: {
    total: number;
    page: number;
    per_page: number;
  };
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`/api/v1${path}`, { ...options, headers });

  if (!res.ok) {
    const body = await res.text().catch(() => "Unknown error");
    if (res.status === 401) {
      localStorage.removeItem("pxbin_api_key");
      window.location.href = "/login";
    }
    throw new ApiError(res.status, body);
  }

  const json = (await res.json()) as ApiResponse<T>;
  return json.data;
}

async function apiFetchPaginated<T>(path: string, options?: RequestInit): Promise<PaginatedResponse<T>> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`/api/v1${path}`, { ...options, headers });

  if (!res.ok) {
    const body = await res.text().catch(() => "Unknown error");
    if (res.status === 401) {
      localStorage.removeItem("pxbin_api_key");
      window.location.href = "/login";
    }
    throw new ApiError(res.status, body);
  }

  const json = (await res.json()) as ApiResponse<T[]>;
  return {
    data: json.data ?? [],
    total: json.meta?.total ?? 0,
    page: json.meta?.page ?? 1,
    per_page: json.meta?.per_page ?? 50,
    total_pages: json.meta ? Math.ceil(json.meta.total / json.meta.per_page) : 1,
  };
}

export { apiFetch, apiFetchPaginated, ApiError, getAuthToken };

export function fetchOverviewStats(period: Period) {
  return apiFetch<OverviewStats>(`/stats/overview?period=${period}`);
}

export function fetchStatsByKey(period: Period, page = 1, perPage = 20) {
  return apiFetchPaginated<KeyStats>(
    `/stats/by-key?period=${period}&page=${page}&per_page=${perPage}`,
  );
}

export function fetchStatsByModel(period: Period) {
  return apiFetch<ModelStats[]>(`/stats/by-model?period=${period}`);
}

export async function fetchTimeSeries(period: Period, interval: Interval) {
  const data = await apiFetch<TimeSeriesBucket[]>(
    `/stats/timeseries?period=${period}&interval=${interval}`,
  );
  // Normalize bucket → timestamp for chart compatibility
  return data.map((d) => ({
    ...d,
    timestamp: d.bucket || d.timestamp,
  }));
}

export function fetchLatencyStats(period: Period) {
  return apiFetch<LatencyStats>(
    `/stats/latency?period=${period}`,
  );
}
