import {
  ResponsiveContainer,
  ComposedChart,
  Area,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from "recharts";
import type { TimeSeriesBucket } from "../lib/types.ts";
import { formatDate } from "../lib/utils.ts";

interface RequestsChartProps {
  data: TimeSeriesBucket[] | undefined;
  isLoading: boolean;
}

const TOOLTIP_STYLE = {
  background: "rgba(9, 9, 11, 0.95)",
  border: "1px solid rgba(63, 63, 70, 0.5)",
  borderRadius: 10,
  fontSize: 12,
  padding: "8px 12px",
};

export function RequestsChart({ data, isLoading }: RequestsChartProps) {
  if (isLoading) {
    return <ChartSkeleton />;
  }

  if (!data || data.length === 0) {
    return <ChartEmpty label="No request data" />;
  }

  const totalRequests = data.reduce((sum, d) => sum + d.requests, 0);
  const totalErrors = data.reduce((sum, d) => sum + d.errors, 0);
  const hasErrors = totalErrors > 0;

  return (
    <div
      className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl p-4"
      style={{ animation: "fadeIn 0.5s ease-out forwards" }}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <h3 className="text-xs font-medium text-zinc-400 uppercase tracking-wider">
            Requests
          </h3>
          <div className="flex items-center gap-3 text-[10px] text-zinc-600">
            <span className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-sky-500" />
              Requests
            </span>
            {hasErrors && (
              <span className="flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
                Errors
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-3 text-xs">
          <span className="font-mono text-zinc-300">
            {totalRequests.toLocaleString()}
          </span>
          {hasErrors && (
            <span className="font-mono text-red-400/80">
              {totalErrors} err
            </span>
          )}
        </div>
      </div>
      <ResponsiveContainer width="100%" height={280}>
        <ComposedChart data={data}>
          <defs>
            <linearGradient
              id="requestsGradient"
              x1="0"
              y1="0"
              x2="0"
              y2="1"
            >
              <stop offset="0%" stopColor="#0ea5e9" stopOpacity={0.2} />
              <stop offset="100%" stopColor="#0ea5e9" stopOpacity={0} />
            </linearGradient>
          </defs>
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
            yAxisId="left"
            tickFormatter={(v: number) => v.toLocaleString()}
            stroke="#3f3f46"
            fontSize={10}
            axisLine={false}
            tickLine={false}
            width={45}
          />
          {hasErrors && (
            <YAxis
              yAxisId="right"
              orientation="right"
              tickFormatter={(v: number) => v.toLocaleString()}
              stroke="#3f3f46"
              fontSize={10}
              axisLine={false}
              tickLine={false}
              width={35}
            />
          )}
          <Tooltip
            contentStyle={TOOLTIP_STYLE}
            labelFormatter={(label) => formatDate(String(label))}
            formatter={(value, name) => [
              (Number(value) || 0).toLocaleString(),
              name === "requests" ? "Requests" : "Errors",
            ]}
          />
          <Area
            yAxisId="left"
            type="monotone"
            dataKey="requests"
            stroke="#0ea5e9"
            fill="url(#requestsGradient)"
            strokeWidth={1.5}
            dot={false}
            activeDot={{ r: 3, fill: "#0ea5e9" }}
          />
          {hasErrors && (
            <Line
              yAxisId="right"
              type="monotone"
              dataKey="errors"
              stroke="#ef4444"
              strokeWidth={1.5}
              dot={false}
              activeDot={{ r: 3, fill: "#ef4444" }}
            />
          )}
        </ComposedChart>
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
