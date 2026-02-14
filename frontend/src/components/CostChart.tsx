import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from "recharts";
import type { TimeSeriesBucket } from "../lib/types.ts";
import { formatDate, formatCost } from "../lib/utils.ts";

interface CostChartProps {
  data: TimeSeriesBucket[] | undefined;
  isLoading: boolean;
}

export function CostChart({ data, isLoading }: CostChartProps) {
  if (isLoading) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 animate-pulse">
        <div className="h-4 w-24 bg-zinc-800 rounded mb-3" />
        <div className="h-[240px] bg-zinc-800/50 rounded" />
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 flex items-center justify-center h-[300px]">
        <p className="text-zinc-500 text-sm">No cost data</p>
      </div>
    );
  }

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
      <h3 className="text-sm font-medium text-zinc-300 mb-3">Cost</h3>
      <ResponsiveContainer width="100%" height={240}>
        <AreaChart data={data}>
          <defs>
            <linearGradient id="costGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#22c55e" stopOpacity={0.3} />
              <stop offset="100%" stopColor="#22c55e" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatDate}
            stroke="#52525b"
            fontSize={11}
          />
          <YAxis
            tickFormatter={(v: number) => formatCost(v)}
            stroke="#52525b"
            fontSize={11}
          />
          <Tooltip
            contentStyle={{
              background: "#18181b",
              border: "1px solid #3f3f46",
              borderRadius: 8,
              fontSize: 12,
            }}
            labelFormatter={(label) => formatDate(String(label))}
            formatter={(value) => [formatCost(Number(value)), "Cost"]}
          />
          <Area
            type="monotone"
            dataKey="cost"
            stroke="#22c55e"
            fill="url(#costGradient)"
            strokeWidth={1.5}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
