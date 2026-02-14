import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "../lib/api.ts";
import type {
  Model,
  CreateModelRequest,
  DiscoveredModel,
  DiscoverModelsRequest,
  ImportModelsRequest,
  ImportModelsResponse,
} from "../lib/types.ts";

const STALE_TIME = 30_000;

export function useModels() {
  return useQuery({
    queryKey: ["models"],
    queryFn: () => apiFetch<Model[]>("/models"),
    staleTime: STALE_TIME,
  });
}

export function useCreateModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateModelRequest) =>
      apiFetch<Model>("/models", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["models"] });
    },
  });
}

export function useUpdateModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: Partial<Model> & { id: string }) =>
      apiFetch<Model>(`/models/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["models"] });
    },
  });
}

export function useDeleteModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/models/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["models"] });
    },
  });
}

export function useBulkDeleteModels() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      apiFetch<{ deleted: number }>("/models/bulk-delete", {
        method: "POST",
        body: JSON.stringify({ ids }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["models"] });
    },
  });
}

export function useDiscoverModels() {
  return useMutation({
    mutationFn: (data: DiscoverModelsRequest) =>
      apiFetch<DiscoveredModel[]>("/models/discover", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  });
}

export function useImportModels() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ImportModelsRequest) =>
      apiFetch<ImportModelsResponse>("/models/import", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["models"] });
      qc.invalidateQueries({ queryKey: ["upstreams"] });
    },
  });
}

export function useSyncPricing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<{ models_updated: number; models_not_found: number; total_models: number }>(
        "/models/sync-pricing",
        { method: "POST" },
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["models"] });
    },
  });
}
