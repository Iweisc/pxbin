import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch, apiFetchPaginated } from "../lib/api.ts";
import type {
  LLMAPIKey,
  ManagementAPIKey,
  CreateKeyRequest,
  CreateKeyResponse,
} from "../lib/types.ts";

const STALE_TIME = 30_000;

export function useLLMKeys(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ["keys", "llm", page, perPage],
    queryFn: () =>
      apiFetchPaginated<LLMAPIKey>(
        `/keys?type=llm&page=${page}&per_page=${perPage}`,
      ),
    staleTime: STALE_TIME,
  });
}

export function useManagementKeys(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ["keys", "management", page, perPage],
    queryFn: () =>
      apiFetchPaginated<ManagementAPIKey>(
        `/keys?type=management&page=${page}&per_page=${perPage}`,
      ),
    staleTime: STALE_TIME,
  });
}

export function useCreateKey(type: "llm" | "management") {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateKeyRequest) =>
      apiFetch<CreateKeyResponse>("/keys", {
        method: "POST",
        body: JSON.stringify({ ...data, type }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] });
    },
  });
}

export function useRevokeKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/keys/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] });
    },
  });
}
