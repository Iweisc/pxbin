import { useQuery } from "@tanstack/react-query";
import { apiFetch, apiFetchPaginated } from "../lib/api.ts";
import type { RequestLog } from "../lib/types.ts";

interface LogFilters {
  page?: number;
  perPage?: number;
  model?: string;
  statusCode?: number;
  inputFormat?: string;
  keyId?: string;
  from?: string;
  to?: string;
}

export function useLogs(params: LogFilters) {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set("page", String(params.page));
  if (params.perPage) searchParams.set("per_page", String(params.perPage));
  if (params.model) searchParams.set("model", params.model);
  if (params.statusCode) searchParams.set("status_code", String(params.statusCode));
  if (params.inputFormat) searchParams.set("input_format", params.inputFormat);
  if (params.keyId) searchParams.set("key_id", params.keyId);
  if (params.from) searchParams.set("from", params.from);
  if (params.to) searchParams.set("to", params.to);

  return useQuery({
    queryKey: ["logs", params],
    queryFn: () => apiFetchPaginated<RequestLog>(`/logs?${searchParams}`),
    staleTime: 10_000,
  });
}

export function useLogDetail(id: string) {
  return useQuery({
    queryKey: ["log", id],
    queryFn: () => apiFetch<RequestLog>(`/logs/${id}`),
    enabled: !!id,
  });
}
