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

const TOOLTIP_STYLE = {
  background: "rgba(9, 9, 11, 0.95)",
  border: "1px solid rgba(63, 63, 70, 0.5)",
  borderRadius: 10,
  fontSize: 12,
  padding: "8px 12px",
};

export function CostChart({ data, isLoading }: CostChartProps) {
  if (isLoading) {
    return <ChartSkeleton />;
  }

  if (!data || data.length === 0) {
    return <ChartEmpty label="No cost data" />;
  }

  const totalCost = data.reduce((sum, d) => sum + d.cost, 0);

  return (
    <div
      className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl p-4"
      style={{ animation: "fadeIn 0.5s ease-out forwards" }}
    >
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-xs font-medium text-zinc-400 uppercase tracking-wider">
          Cost
        </h3>
        <span className="text-xs font-mono text-emerald-400">
          {formatCost(totalCost)}
        </span>
      </div>
      <ResponsiveContainer width="100%" height={280}>
        <AreaChart data={data}>
          <defs>
            <linearGradient id="costGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#10b981" stopOpacity={0.25} />
              <stop offset="50%" stopColor="#10b981" stopOpacity={0.08} />
              <stop offset="100%" stopColor="#10b981" stopOpacity={0} />
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
            tickFormatter={(v: number) => formatCost(v)}
            stroke="#3f3f46"
            fontSize={10}
            axisLine={false}
            tickLine={false}
            width={50}
          />
          <Tooltip
            contentStyle={TOOLTIP_STYLE}
            labelFormatter={(label) => formatDate(String(label))}
            formatter={(value) => [formatCost(Number(value)), "Cost"]}
          />
          <Area
            type="monotone"
            dataKey="cost"
            stroke="#10b981"
            fill="url(#costGradient)"
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 3, fill: "#10b981" }}
          />
        </AreaChart>
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
