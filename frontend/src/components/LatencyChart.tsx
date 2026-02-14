import type { LatencyStats } from "../lib/types.ts";
import { formatDuration } from "../lib/utils.ts";

interface LatencyChartProps {
  data: LatencyStats | undefined;
  isLoading: boolean;
}

export function LatencyChart({ data, isLoading }: LatencyChartProps) {
  if (isLoading) {
    return <ChartSkeleton />;
  }

  if (!data) {
    return <ChartEmpty label="No latency data" />;
  }

  const items = [
    { label: "p50", value: data.p50, color: "text-emerald-400" },
    { label: "p95", value: data.p95, color: "text-amber-400" },
    { label: "p99", value: data.p99, color: "text-red-400" },
  ];

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
      <h3 className="text-sm font-medium text-zinc-300 mb-3">Latency Percentiles</h3>
      <div className="space-y-4 py-4">
        {items.map(({ label, value, color }) => (
          <div key={label} className="flex items-center justify-between">
            <span className="text-xs text-zinc-500 uppercase tracking-wider w-10">{label}</span>
            <div className="flex-1 mx-3 bg-zinc-800 rounded-full h-2 overflow-hidden">
              <div
                className={`h-full rounded-full ${label === "p50" ? "bg-emerald-500" : label === "p95" ? "bg-amber-500" : "bg-red-500"}`}
                style={{ width: `${Math.min(100, data.p99 > 0 ? (value / data.p99) * 100 : 0)}%` }}
              />
            </div>
            <span className={`font-mono text-sm font-semibold ${color}`}>
              {formatDuration(value)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function ChartSkeleton() {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 animate-pulse">
      <div className="h-4 w-24 bg-zinc-800 rounded mb-3" />
      <div className="h-[240px] bg-zinc-800/50 rounded" />
    </div>
  );
}

function ChartEmpty({ label }: { label: string }) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 flex items-center justify-center h-[300px]">
      <p className="text-zinc-500 text-sm">{label}</p>
    </div>
  );
}
