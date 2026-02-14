import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "../lib/api.ts";
import type { Upstream, CreateUpstreamRequest } from "../lib/types.ts";

const STALE_TIME = 30_000;

export function useUpstreams() {
  return useQuery({
    queryKey: ["upstreams"],
    queryFn: () => apiFetch<Upstream[]>("/upstreams"),
    staleTime: STALE_TIME,
  });
}

export function useCreateUpstream() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateUpstreamRequest) =>
      apiFetch<Upstream>("/upstreams", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["upstreams"] });
    },
  });
}

export function useUpdateUpstream() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Upstream> & { id: string }) =>
      apiFetch<Upstream>(`/upstreams/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["upstreams"] });
    },
  });
}

export function useDeleteUpstream() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/upstreams/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["upstreams"] });
    },
  });
}

export function useBulkDeleteUpstreams() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      apiFetch<{ deleted: number }>("/upstreams/bulk-delete", {
        method: "POST",
        body: JSON.stringify({ ids }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["upstreams"] });
      qc.invalidateQueries({ queryKey: ["models"] });
    },
  });
}
