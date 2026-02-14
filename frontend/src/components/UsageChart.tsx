import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from "recharts";
import type { TimeSeriesBucket } from "../lib/types.ts";
import { formatDate, formatTokens } from "../lib/utils.ts";

interface UsageChartProps {
  data: TimeSeriesBucket[] | undefined;
  isLoading: boolean;
}

export function UsageChart({ data, isLoading }: UsageChartProps) {
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
        <p className="text-zinc-500 text-sm">No usage data</p>
      </div>
    );
  }

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
      <h3 className="text-sm font-medium text-zinc-300 mb-3">Token Usage</h3>
      <ResponsiveContainer width="100%" height={240}>
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatDate}
            stroke="#52525b"
            fontSize={11}
          />
          <YAxis
            tickFormatter={(v: number) => formatTokens(v)}
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
            formatter={(value, name) => [
              formatTokens(Number(value)),
              name,
            ]}
          />
          <Legend
            wrapperStyle={{ fontSize: 12, color: "#a1a1aa" }}
          />
          <Bar
            dataKey="input_tokens"
            name="Input"
            fill="#3b82f6"
            stackId="tokens"
            radius={[0, 0, 0, 0]}
          />
          <Bar
            dataKey="output_tokens"
            name="Output"
            fill="#8b5cf6"
            stackId="tokens"
            radius={[2, 2, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
