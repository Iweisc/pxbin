import { useState } from "react";
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from "recharts";
import { ProtectedRoute } from "../lib/auth.tsx";
import { DataTable, type Column } from "../components/DataTable.tsx";
import { useStatsByKey, useStatsByModel } from "../hooks/useStats.ts";
import { formatCost, formatTokens } from "../lib/utils.ts";
import type { Period, KeyStats, ModelStats } from "../lib/types.ts";

const PERIOD_OPTIONS: { label: string; value: Period }[] = [
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
  { label: "30d", value: "30d" },
];

const keyColumns: Column<KeyStats>[] = [
  {
    key: "key_name",
    header: "Key",
    render: (k) => (
      <div>
        <span className="text-sm">{k.key_name}</span>
        <span className="text-xs text-zinc-500 ml-2 font-mono">
          {k.key_prefix}...
        </span>
      </div>
    ),
  },
  {
    key: "total_requests",
    header: "Requests",
    render: (k) => (
      <span className="font-mono text-xs">
        {k.total_requests.toLocaleString()}
      </span>
    ),
  },
  {
    key: "total_input_tokens",
    header: "Input Tokens",
    render: (k) => (
      <span className="font-mono text-xs">
        {formatTokens(k.total_input_tokens)}
      </span>
    ),
  },
  {
    key: "total_output_tokens",
    header: "Output Tokens",
    render: (k) => (
      <span className="font-mono text-xs">
        {formatTokens(k.total_output_tokens)}
      </span>
    ),
  },
  {
    key: "total_cost",
    header: "Cost",
    render: (k) => (
      <span className="font-mono text-xs font-medium text-emerald-400">
        {formatCost(k.total_cost)}
      </span>
    ),
  },
];

const modelColumns: Column<ModelStats>[] = [
  {
    key: "model",
    header: "Model",
    render: (m) => <span className="font-mono text-xs">{m.model}</span>,
  },
  {
    key: "total_requests",
    header: "Requests",
    render: (m) => (
      <span className="font-mono text-xs">
        {m.total_requests.toLocaleString()}
      </span>
    ),
  },
  {
    key: "total_input_tokens",
    header: "Input Tokens",
    render: (m) => (
      <span className="font-mono text-xs">
        {formatTokens(m.total_input_tokens)}
      </span>
    ),
  },
  {
    key: "total_output_tokens",
    header: "Output Tokens",
    render: (m) => (
      <span className="font-mono text-xs">
        {formatTokens(m.total_output_tokens)}
      </span>
    ),
  },
  {
    key: "total_cost",
    header: "Cost",
    render: (m) => (
      <span className="font-mono text-xs font-medium text-emerald-400">
        {formatCost(m.total_cost)}
      </span>
    ),
  },
];

export function CostsPage() {
  const [period, setPeriod] = useState<Period>("24h");
  const [tab, setTab] = useState<"key" | "model">("key");

  const byKey = useStatsByKey(period);
  const byModel = useStatsByModel(period);

  const chartData =
    tab === "key"
      ? (byKey.data?.data ?? []).map((k) => ({
          name: k.key_name || k.key_prefix,
          cost: k.total_cost,
        }))
      : (byModel.data ?? []).map((m) => ({
          name: m.model,
          cost: m.total_cost,
        }));

  return (
    <ProtectedRoute>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-zinc-100">Costs</h1>
          <div className="flex gap-1 bg-zinc-900 rounded-lg p-0.5 border border-zinc-800">
            {PERIOD_OPTIONS.map(({ label, value }) => (
              <button
                key={value}
                onClick={() => setPeriod(value)}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                  period === value
                    ? "bg-zinc-700 text-zinc-100"
                    : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex gap-1 bg-zinc-900 rounded-lg p-0.5 border border-zinc-800 w-fit">
          <button
            onClick={() => setTab("key")}
            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
              tab === "key"
                ? "bg-zinc-700 text-zinc-100"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            By Key
          </button>
          <button
            onClick={() => setTab("model")}
            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
              tab === "model"
                ? "bg-zinc-700 text-zinc-100"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            By Model
          </button>
        </div>

        {chartData.length > 0 && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
            <h3 className="text-sm font-medium text-zinc-300 mb-3">
              Cost Breakdown
            </h3>
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
                <XAxis
                  dataKey="name"
                  stroke="#52525b"
                  fontSize={11}
                  interval={0}
                  angle={-30}
                  textAnchor="end"
                  height={60}
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
                  formatter={(value) => [formatCost(Number(value)), "Cost"]}
                />
                <Bar dataKey="cost" fill="#22c55e" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

        {tab === "key" ? (
          <DataTable
            columns={keyColumns}
            data={byKey.data?.data ?? []}
            isLoading={byKey.isLoading}
            emptyMessage="No cost data for this period"
          />
        ) : (
          <DataTable
            columns={modelColumns}
            data={byModel.data ?? []}
            isLoading={byModel.isLoading}
            emptyMessage="No cost data for this period"
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
