import { useState } from "react";
import { ProtectedRoute } from "../lib/auth.tsx";
import { StatsCards } from "../components/StatsCards.tsx";
import { LatencyChart } from "../components/LatencyChart.tsx";
import { CostChart } from "../components/CostChart.tsx";
import { UsageChart } from "../components/UsageChart.tsx";
import { useOverviewStats, useTimeSeries, useLatencyStats } from "../hooks/useStats.ts";
import type { Period, Interval } from "../lib/types.ts";

const PERIOD_OPTIONS: { label: string; value: Period }[] = [
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
  { label: "30d", value: "30d" },
];

function intervalForPeriod(period: Period): Interval {
  switch (period) {
    case "24h":
      return "5m";
    case "7d":
      return "1h";
    case "30d":
      return "1d";
  }
}

export function DashboardPage() {
  const [period, setPeriod] = useState<Period>("24h");
  const interval = intervalForPeriod(period);

  const overview = useOverviewStats(period);
  const timeSeries = useTimeSeries(period, interval);
  const latency = useLatencyStats(period);

  return (
    <ProtectedRoute>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-zinc-100">Dashboard</h1>
          <div className="flex gap-1 bg-zinc-900 rounded-lg p-0.5 border border-zinc-800">
            {PERIOD_OPTIONS.map(({ label, value }) => (
              <button
                key={value}
                onClick={() => setPeriod(value)}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                  period === value
                    ? "bg-zinc-700 text-zinc-100"
                    : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <StatsCards data={overview.data} isLoading={overview.isLoading} />

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
          <LatencyChart data={latency.data} isLoading={latency.isLoading} />
          <CostChart data={timeSeries.data} isLoading={timeSeries.isLoading} />
          <UsageChart data={timeSeries.data} isLoading={timeSeries.isLoading} />
        </div>
      </div>
    </ProtectedRoute>
  );
}
