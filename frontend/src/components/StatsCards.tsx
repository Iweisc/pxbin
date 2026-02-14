import type { OverviewStats } from "../lib/types.ts";
import { formatCost, formatTokens, formatDuration } from "../lib/utils.ts";

interface StatsCardsProps {
  data: OverviewStats | undefined;
  isLoading: boolean;
}

interface CardProps {
  label: string;
  value: string;
  sub?: string;
  variant?: "default" | "error";
}

function Card({ label, value, sub, variant = "default" }: CardProps) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
      <p className="text-xs text-zinc-500 uppercase tracking-wider mb-1">
        {label}
      </p>
      <p
        className={`text-2xl font-mono font-semibold ${variant === "error" ? "text-red-400" : "text-zinc-100"}`}
      >
        {value}
      </p>
      {sub && <p className="text-xs text-zinc-500 mt-1">{sub}</p>}
    </div>
  );
}

function Skeleton() {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 animate-pulse">
      <div className="h-3 w-20 bg-zinc-800 rounded mb-3" />
      <div className="h-7 w-24 bg-zinc-800 rounded" />
    </div>
  );
}

export function StatsCards({ data, isLoading }: StatsCardsProps) {
  if (isLoading || !data) {
    return (
      <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3">
        {Array.from({ length: 7 }).map((_, i) => (
          <Skeleton key={i} />
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-7 gap-3">
      <Card
        label="Total Requests"
        value={data.total_requests.toLocaleString()}
      />
      <Card
        label="Input Tokens"
        value={formatTokens(data.total_input_tokens)}
      />
      <Card
        label="Output Tokens"
        value={formatTokens(data.total_output_tokens)}
      />
      <Card
        label="Cache Hits"
        value={formatTokens(data.total_cache_read_tokens)}
        sub={`${(data.cache_hit_rate * 100).toFixed(1)}% hit rate`}
      />
      <Card label="Total Cost" value={formatCost(data.total_cost)} />
      <Card
        label="Error Rate"
        value={`${(data.error_rate * 100).toFixed(1)}%`}
        sub={`${data.error_count} errors`}
        variant={data.error_rate > 0.05 ? "error" : "default"}
      />
      <Card
        label="Avg Latency"
        value={formatDuration(data.avg_latency_ms)}
      />
    </div>
  );
}
