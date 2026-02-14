import type { OverviewStats } from "../lib/types.ts";
import { formatCost, formatTokens, formatDuration } from "../lib/utils.ts";

interface StatsCardsProps {
  data: OverviewStats | undefined;
  isLoading: boolean;
}

interface StatConfig {
  label: string;
  value: string;
  sub?: string;
  accent: string;
  variant?: "error";
}

function StatCard({
  label,
  value,
  sub,
  accent,
  variant,
  index,
}: StatConfig & { index: number }) {
  return (
    <div
      className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl px-3.5 py-3 hover:bg-zinc-800/40 transition-all duration-200"
      style={{
        borderTopWidth: 2,
        borderTopColor: accent,
        animation: `fadeInUp 0.35s ease-out ${index * 50}ms forwards`,
        opacity: 0,
      }}
    >
      <p
        className={`text-xl font-mono font-semibold tracking-tight ${
          variant === "error" ? "text-red-400" : "text-zinc-100"
        }`}
      >
        {value}
      </p>
      <p className="text-[10px] text-zinc-500 uppercase tracking-wider mt-1 font-medium">
        {label}
      </p>
      {sub && (
        <p className="text-[10px] text-zinc-600 mt-0.5 font-mono">{sub}</p>
      )}
    </div>
  );
}

function Skeleton() {
  return (
    <div
      className="bg-zinc-900/40 border border-zinc-800/30 rounded-xl px-3.5 py-3 animate-pulse"
      style={{
        borderTopWidth: 2,
        borderTopColor: "#3f3f46",
      }}
    >
      <div className="h-6 w-16 bg-zinc-800/60 rounded mb-2" />
      <div className="h-2.5 w-20 bg-zinc-800/40 rounded" />
    </div>
  );
}

export function StatsCards({ data, isLoading }: StatsCardsProps) {
  if (isLoading || !data) {
    return (
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-2.5">
        {Array.from({ length: 7 }).map((_, i) => (
          <Skeleton key={i} />
        ))}
      </div>
    );
  }

  const stats: StatConfig[] = [
    {
      label: "Requests",
      value: data.total_requests.toLocaleString(),
      accent: "#0ea5e9",
    },
    {
      label: "Input Tokens",
      value: formatTokens(data.total_input_tokens),
      accent: "#3b82f6",
    },
    {
      label: "Output Tokens",
      value: formatTokens(data.total_output_tokens),
      accent: "#8b5cf6",
    },
    {
      label: "Cache Hits",
      value: formatTokens(data.total_cache_read_tokens),
      sub: `${(data.cache_hit_rate * 100).toFixed(1)}% hit rate`,
      accent: "#14b8a6",
    },
    {
      label: "Total Cost",
      value: formatCost(data.total_cost),
      accent: "#10b981",
    },
    {
      label: "Error Rate",
      value: `${(data.error_rate * 100).toFixed(1)}%`,
      sub: `${data.error_count} errors`,
      accent: "#ef4444",
      variant: data.error_rate > 0.05 ? "error" : undefined,
    },
    {
      label: "Avg Latency",
      value: formatDuration(data.avg_latency_ms),
      accent: "#f59e0b",
    },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-2.5">
      {stats.map((stat, i) => (
        <StatCard key={stat.label} {...stat} index={i} />
      ))}
    </div>
  );
}
