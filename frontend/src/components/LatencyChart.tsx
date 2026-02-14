import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from "recharts";
import type { TimeSeriesBucket, LatencyStats } from "../lib/types.ts";
import { formatDate, formatDuration } from "../lib/utils.ts";

interface LatencyChartProps {
  data: TimeSeriesBucket[] | undefined;
  summary?: LatencyStats;
  isLoading: boolean;
}

const TOOLTIP_STYLE = {
  background: "rgba(9, 9, 11, 0.95)",
  border: "1px solid rgba(63, 63, 70, 0.5)",
  borderRadius: 10,
  fontSize: 12,
  padding: "8px 12px",
};

export function LatencyChart({ data, summary, isLoading }: LatencyChartProps) {
  if (isLoading) {
    return <ChartSkeleton />;
  }

  if (!data || data.length === 0) {
    return <ChartEmpty label="No latency data" />;
  }

  const hasPercentiles = data.some(
    (d) => d.p50_latency_ms != null && d.p50_latency_ms > 0,
  );

  return (
    <div
      className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl p-4"
      style={{ animation: "fadeIn 0.5s ease-out forwards" }}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <h3 className="text-xs font-medium text-zinc-400 uppercase tracking-wider">
            Latency
          </h3>
          <div className="flex items-center gap-3 text-[10px] text-zinc-600">
            {hasPercentiles ? (
              <>
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                  p50
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
                  p95
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
                  p99
                </span>
              </>
            ) : (
              <span className="flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                avg
              </span>
            )}
          </div>
        </div>
        {summary && (
          <div className="flex items-center gap-2.5 text-[10px] font-mono">
            <span className="text-emerald-400">
              p50 {formatDuration(summary.p50)}
            </span>
            <span className="text-amber-400">
              p95 {formatDuration(summary.p95)}
            </span>
            <span className="text-red-400">
              p99 {formatDuration(summary.p99)}
            </span>
          </div>
        )}
      </div>
      <ResponsiveContainer width="100%" height={280}>
        <LineChart data={data}>
          <CartesianGrid
            horizontal
            vertical={false}
            stroke="#27272a"
            strokeOpacity={0.5}
          />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatDate}
            stroke="#3f3f46"
            fontSize={10}
            axisLine={false}
            tickLine={false}
          />
          <YAxis
            tickFormatter={(v: number) => formatDuration(v)}
            stroke="#3f3f46"
            fontSize={10}
            axisLine={false}
            tickLine={false}
            width={50}
          />
          <Tooltip
            contentStyle={TOOLTIP_STYLE}
            labelFormatter={(label) => formatDate(String(label))}
            formatter={(value, name) => {
              const labels: Record<string, string> = {
                p50_latency_ms: "p50",
                p95_latency_ms: "p95",
                p99_latency_ms: "p99",
                avg_latency_ms: "Avg",
              };
              return [formatDuration(Number(value) || 0), labels[String(name)] || name];
            }}
          />
          {hasPercentiles ? (
            <>
              <Line
                type="monotone"
                dataKey="p50_latency_ms"
                stroke="#10b981"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 3, fill: "#10b981" }}
              />
              <Line
                type="monotone"
                dataKey="p95_latency_ms"
                stroke="#f59e0b"
                strokeWidth={1.5}
                dot={false}
                activeDot={{ r: 3, fill: "#f59e0b" }}
              />
              <Line
                type="monotone"
                dataKey="p99_latency_ms"
                stroke="#ef4444"
                strokeWidth={1.5}
                dot={false}
                strokeDasharray="4 2"
                activeDot={{ r: 3, fill: "#ef4444" }}
              />
            </>
          ) : (
            <Line
              type="monotone"
              dataKey="avg_latency_ms"
              stroke="#10b981"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 3, fill: "#10b981" }}
            />
          )}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

function ChartSkeleton() {
  return (
    <div className="bg-zinc-900/40 border border-zinc-800/30 rounded-xl p-4 animate-pulse">
      <div className="h-3 w-20 bg-zinc-800/60 rounded mb-3" />
      <div className="h-[280px] bg-zinc-800/30 rounded-lg" />
    </div>
  );
}

function ChartEmpty({ label }: { label: string }) {
  return (
    <div className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl p-4 flex items-center justify-center h-[340px]">
      <p className="text-zinc-600 text-sm">{label}</p>
    </div>
  );
}
