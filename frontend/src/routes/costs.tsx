import { useState, useMemo } from "react";
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Cell,
} from "recharts";
import { ProtectedRoute } from "../lib/auth.tsx";
import { useStatsByKey, useStatsByModel } from "../hooks/useStats.ts";
import { formatCost, formatTokens, formatDuration } from "../lib/utils.ts";
import type { Period, KeyStats, ModelStats } from "../lib/types.ts";

const PERIOD_OPTIONS: { label: string; value: Period }[] = [
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
  { label: "30d", value: "30d" },
];

const TOOLTIP_STYLE = {
  background: "rgba(9, 9, 11, 0.95)",
  border: "1px solid rgba(63, 63, 70, 0.5)",
  borderRadius: 10,
  fontSize: 12,
  padding: "8px 12px",
};

const BAR_COLORS = [
  "#10b981", "#0ea5e9", "#8b5cf6", "#f59e0b", "#ef4444",
  "#14b8a6", "#3b82f6", "#a855f7", "#f97316", "#ec4899",
];

interface CostRow {
  name: string;
  cost: number;
  requests: number;
  input_tokens: number;
  output_tokens: number;
  error_count: number;
  avg_latency_ms: number;
}

function buildRows(
  tab: "key" | "model",
  keyData: KeyStats[],
  modelData: ModelStats[],
): CostRow[] {
  if (tab === "key") {
    return keyData.map((k) => ({
      name: k.key_name || k.key_prefix,
      cost: k.total_cost,
      requests: k.total_requests,
      input_tokens: k.total_input_tokens,
      output_tokens: k.total_output_tokens,
      error_count: k.error_count,
      avg_latency_ms: k.avg_latency_ms,
    }));
  }
  return modelData.map((m) => ({
    name: m.model,
    cost: m.total_cost,
    requests: m.total_requests,
    input_tokens: m.total_input_tokens,
    output_tokens: m.total_output_tokens,
    error_count: m.error_count,
    avg_latency_ms: m.avg_latency_ms,
  }));
}

export function CostsPage() {
  const [period, setPeriod] = useState<Period>("24h");
  const [tab, setTab] = useState<"key" | "model">("key");

  const byKey = useStatsByKey(period);
  const byModel = useStatsByModel(period);

  const isLoading = tab === "key" ? byKey.isLoading : byModel.isLoading;

  const rows = useMemo(
    () => buildRows(tab, byKey.data?.data ?? [], byModel.data ?? []),
    [tab, byKey.data, byModel.data],
  );

  const totalCost = rows.reduce((s, r) => s + r.cost, 0);
  const totalRequests = rows.reduce((s, r) => s + r.requests, 0);

  const chartData = rows
    .slice()
    .sort((a, b) => b.cost - a.cost)
    .slice(0, 10);

  return (
    <ProtectedRoute>
      <div className="space-y-4">
        {/* Header row: title + tab toggle + period selector */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <h1 className="text-lg font-semibold text-zinc-100 tracking-tight">
              Costs
            </h1>
            <div className="flex gap-0.5 bg-zinc-900/80 rounded-lg p-0.5 border border-zinc-800/50">
              <button
                onClick={() => setTab("key")}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-all duration-150 ${
                  tab === "key"
                    ? "bg-zinc-700/80 text-zinc-100 shadow-sm"
                    : "text-zinc-500 hover:text-zinc-300"
                }`}
              >
                By Key
              </button>
              <button
                onClick={() => setTab("model")}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-all duration-150 ${
                  tab === "model"
                    ? "bg-zinc-700/80 text-zinc-100 shadow-sm"
                    : "text-zinc-500 hover:text-zinc-300"
                }`}
              >
                By Model
              </button>
            </div>
          </div>
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

        {/* Summary stats */}
        {!isLoading && rows.length > 0 && (
          <div className="grid grid-cols-3 gap-2.5">
            <SummaryCard
              label="Total Cost"
              value={formatCost(totalCost)}
              accent="#10b981"
            />
            <SummaryCard
              label="Total Requests"
              value={totalRequests.toLocaleString()}
              accent="#0ea5e9"
            />
            <SummaryCard
              label="Avg Cost / Request"
              value={
                totalRequests > 0
                  ? formatCost(totalCost / totalRequests)
                  : "$0.00"
              }
              accent="#8b5cf6"
            />
          </div>
        )}

        {/* Chart */}
        {isLoading ? (
          <div className="bg-zinc-900/40 border border-zinc-800/30 rounded-xl p-4 animate-pulse">
            <div className="h-3 w-28 bg-zinc-800/60 rounded mb-3" />
            <div className="h-[300px] bg-zinc-800/30 rounded-lg" />
          </div>
        ) : chartData.length > 0 ? (
          <div
            className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl p-4"
            style={{ animation: "fadeIn 0.5s ease-out forwards" }}
          >
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-xs font-medium text-zinc-400 uppercase tracking-wider">
                Cost Breakdown
              </h3>
              <span className="text-[10px] text-zinc-600">
                Top {chartData.length}
              </span>
            </div>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart
                data={chartData}
                layout="vertical"
                margin={{ left: 8, right: 16, top: 4, bottom: 4 }}
              >
                <CartesianGrid
                  horizontal={false}
                  vertical
                  stroke="#27272a"
                  strokeOpacity={0.5}
                />
                <XAxis
                  type="number"
                  tickFormatter={(v: number) => formatCost(v)}
                  stroke="#3f3f46"
                  fontSize={10}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  type="category"
                  dataKey="name"
                  stroke="#3f3f46"
                  fontSize={10}
                  axisLine={false}
                  tickLine={false}
                  width={120}
                  tick={{ fill: "#a1a1aa", fontFamily: "var(--font-mono)" }}
                />
                <Tooltip
                  contentStyle={TOOLTIP_STYLE}
                  formatter={(value) => [formatCost(Number(value)), "Cost"]}
                  labelStyle={{ color: "#e4e4e7", fontFamily: "var(--font-mono)", fontSize: 11 }}
                />
                <Bar dataKey="cost" radius={[0, 4, 4, 0]} barSize={20}>
                  {chartData.map((_, i) => (
                    <Cell
                      key={i}
                      fill={BAR_COLORS[i % BAR_COLORS.length]}
                      fillOpacity={0.8}
                    />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        ) : (
          <div className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl p-4 flex items-center justify-center py-16">
            <p className="text-zinc-600 text-sm">No cost data for this period</p>
          </div>
        )}

        {/* Table */}
        {isLoading ? (
          <div className="bg-zinc-900/40 border border-zinc-800/30 rounded-xl overflow-hidden animate-pulse p-4 space-y-3">
            <div className="h-4 bg-zinc-800/60 rounded w-full" />
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="h-8 bg-zinc-800/30 rounded w-full" />
            ))}
          </div>
        ) : rows.length > 0 ? (
          <div
            className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl overflow-hidden"
            style={{ animation: "fadeIn 0.5s ease-out 0.1s forwards", opacity: 0 }}
          >
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-zinc-800/60">
                    <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      {tab === "key" ? "Key" : "Model"}
                    </th>
                    <th className="px-4 py-2.5 text-right text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      Cost
                    </th>
                    <th className="px-4 py-2.5 text-right text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      Share
                    </th>
                    <th className="px-4 py-2.5 text-right text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      Requests
                    </th>
                    <th className="px-4 py-2.5 text-right text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      Input
                    </th>
                    <th className="px-4 py-2.5 text-right text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      Output
                    </th>
                    <th className="px-4 py-2.5 text-right text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      Errors
                    </th>
                    <th className="px-4 py-2.5 text-right text-[10px] font-medium text-zinc-500 uppercase tracking-wider">
                      Avg Latency
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/30">
                  {rows.map((row, i) => {
                    const share =
                      totalCost > 0 ? (row.cost / totalCost) * 100 : 0;
                    return (
                      <tr
                        key={i}
                        className="hover:bg-zinc-800/20 transition-colors"
                      >
                        <td className="px-4 py-2.5 whitespace-nowrap">
                          <span className="font-mono text-xs text-zinc-200">
                            {row.name}
                          </span>
                        </td>
                        <td className="px-4 py-2.5 text-right whitespace-nowrap">
                          <span className="font-mono text-xs font-medium text-emerald-400">
                            {formatCost(row.cost)}
                          </span>
                        </td>
                        <td className="px-4 py-2.5 text-right whitespace-nowrap">
                          <div className="flex items-center justify-end gap-2">
                            <div className="w-12 h-1 bg-zinc-800 rounded-full overflow-hidden">
                              <div
                                className="h-full rounded-full bg-emerald-500/60"
                                style={{ width: `${Math.min(100, share)}%` }}
                              />
                            </div>
                            <span className="font-mono text-[10px] text-zinc-500 w-10 text-right">
                              {share.toFixed(1)}%
                            </span>
                          </div>
                        </td>
                        <td className="px-4 py-2.5 text-right whitespace-nowrap font-mono text-xs text-zinc-300">
                          {row.requests.toLocaleString()}
                        </td>
                        <td className="px-4 py-2.5 text-right whitespace-nowrap font-mono text-xs text-zinc-400">
                          {formatTokens(row.input_tokens)}
                        </td>
                        <td className="px-4 py-2.5 text-right whitespace-nowrap font-mono text-xs text-zinc-400">
                          {formatTokens(row.output_tokens)}
                        </td>
                        <td className="px-4 py-2.5 text-right whitespace-nowrap font-mono text-xs">
                          <span
                            className={
                              row.error_count > 0
                                ? "text-red-400"
                                : "text-zinc-600"
                            }
                          >
                            {row.error_count}
                          </span>
                        </td>
                        <td className="px-4 py-2.5 text-right whitespace-nowrap font-mono text-xs text-zinc-400">
                          {formatDuration(row.avg_latency_ms)}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ) : null}
      </div>
    </ProtectedRoute>
  );
}

function SummaryCard({
  label,
  value,
  accent,
}: {
  label: string;
  value: string;
  accent: string;
}) {
  return (
    <div
      className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl px-3.5 py-3"
      style={{
        borderTopWidth: 2,
        borderTopColor: accent,
        animation: "fadeInUp 0.35s ease-out forwards",
      }}
    >
      <p className="text-xl font-mono font-semibold tracking-tight text-zinc-100">
        {value}
      </p>
      <p className="text-[10px] text-zinc-500 uppercase tracking-wider mt-1 font-medium">
        {label}
      </p>
    </div>
  );
}
