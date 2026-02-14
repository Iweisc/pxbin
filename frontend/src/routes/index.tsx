import { useState } from "react";
import { ProtectedRoute } from "../lib/auth.tsx";
import { StatsCards } from "../components/StatsCards.tsx";
import { RequestsChart } from "../components/RequestsChart.tsx";
import { CostChart } from "../components/CostChart.tsx";
import { UsageChart } from "../components/UsageChart.tsx";
import { LatencyChart } from "../components/LatencyChart.tsx";
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
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-zinc-100 tracking-tight">
            Dashboard
          </h1>
          <div className="flex gap-0.5 bg-zinc-900/80 rounded-lg p-0.5 border border-zinc-800/50">
            {PERIOD_OPTIONS.map(({ label, value }) => (
              <button
                key={value}
                onClick={() => setPeriod(value)}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-all duration-150 ${
                  period === value
                    ? "bg-zinc-700/80 text-zinc-100 shadow-sm"
                    : "text-zinc-500 hover:text-zinc-300"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <StatsCards data={overview.data} isLoading={overview.isLoading} />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
          <RequestsChart data={timeSeries.data} isLoading={timeSeries.isLoading} />
          <CostChart data={timeSeries.data} isLoading={timeSeries.isLoading} />
          <UsageChart data={timeSeries.data} isLoading={timeSeries.isLoading} />
          <LatencyChart
            data={timeSeries.data}
            summary={latency.data}
            isLoading={timeSeries.isLoading}
          />
        </div>
      </div>
    </ProtectedRoute>
  );
}
