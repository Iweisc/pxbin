import { useQuery } from "@tanstack/react-query";
import {
  fetchOverviewStats,
  fetchStatsByKey,
  fetchStatsByModel,
  fetchTimeSeries,
  fetchLatencyStats,
} from "../lib/api.ts";
import type { Period, Interval } from "../lib/types.ts";

const STALE_TIME = 30_000;

export function useOverviewStats(period: Period) {
  return useQuery({
    queryKey: ["overview-stats", period],
    queryFn: () => fetchOverviewStats(period),
    staleTime: STALE_TIME,
    refetchInterval: STALE_TIME,
  });
}

export function useStatsByKey(period: Period, page = 1) {
  return useQuery({
    queryKey: ["stats-by-key", period, page],
    queryFn: () => fetchStatsByKey(period, page),
    staleTime: STALE_TIME,
    refetchInterval: STALE_TIME,
  });
}

export function useStatsByModel(period: Period) {
  return useQuery({
    queryKey: ["stats-by-model", period],
    queryFn: () => fetchStatsByModel(period),
    staleTime: STALE_TIME,
    refetchInterval: STALE_TIME,
  });
}

export function useTimeSeries(period: Period, interval: Interval) {
  return useQuery({
    queryKey: ["timeseries", period, interval],
    queryFn: () => fetchTimeSeries(period, interval),
    staleTime: STALE_TIME,
    refetchInterval: STALE_TIME,
  });
}

export function useLatencyStats(period: Period) {
  return useQuery({
    queryKey: ["latency-stats", period],
    queryFn: () => fetchLatencyStats(period),
    staleTime: STALE_TIME,
    refetchInterval: STALE_TIME,
  });
}
