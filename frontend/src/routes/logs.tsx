import { useState } from "react";
import { ProtectedRoute } from "../lib/auth.tsx";
import { LogsTable } from "../components/LogsTable.tsx";
import { LogDetail } from "../components/LogDetail.tsx";
import { Pagination } from "../components/Pagination.tsx";
import { DateRangePicker } from "../components/DateRangePicker.tsx";
import { useLogs } from "../hooks/useLogs.ts";
import type { RequestLog } from "../lib/types.ts";

const FORMAT_OPTIONS = [
  { label: "All Formats", value: "" },
  { label: "Anthropic", value: "anthropic" },
  { label: "OpenAI", value: "openai" },
];

const STATUS_OPTIONS = [
  { label: "All Status", value: 0 },
  { label: "2xx", value: 200 },
  { label: "4xx", value: 400 },
  { label: "5xx", value: 500 },
];

export function LogsPage() {
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [model, setModel] = useState("");
  const [statusCode, setStatusCode] = useState(0);
  const [inputFormat, setInputFormat] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [selected, setSelected] = useState<RequestLog | null>(null);

  const { data, isLoading } = useLogs({
    page,
    perPage,
    model: model || undefined,
    statusCode: statusCode || undefined,
    inputFormat: inputFormat || undefined,
    from: from || undefined,
    to: to || undefined,
  });

  return (
    <ProtectedRoute>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-zinc-100 tracking-tight">Logs</h1>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          <input
            value={model}
            onChange={(e) => {
              setModel(e.target.value);
              setPage(1);
            }}
            placeholder="Filter by model..."
            className="bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-xs text-zinc-300 px-3 py-1.5 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 w-48 transition-colors"
          />
          <select
            value={inputFormat}
            onChange={(e) => {
              setInputFormat(e.target.value);
              setPage(1);
            }}
            className="bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-xs text-zinc-300 px-2.5 py-1.5 focus:outline-none focus:border-zinc-500 transition-colors"
          >
            {FORMAT_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <select
            value={statusCode}
            onChange={(e) => {
              setStatusCode(Number(e.target.value));
              setPage(1);
            }}
            className="bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-xs text-zinc-300 px-2.5 py-1.5 focus:outline-none focus:border-zinc-500 transition-colors"
          >
            {STATUS_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <DateRangePicker
            from={from}
            to={to}
            onChange={(f, t) => {
              setFrom(f);
              setTo(t);
              setPage(1);
            }}
          />
        </div>

        <LogsTable
          data={data?.data ?? []}
          isLoading={isLoading}
          onRowClick={setSelected}
        />

        {data && (
          <Pagination
            page={data.page}
            perPage={data.per_page}
            total={data.total}
            onPageChange={setPage}
            onPerPageChange={setPerPage}
          />
        )}

        {selected && (
          <LogDetail log={selected} onClose={() => setSelected(null)} />
        )}
      </div>
    </ProtectedRoute>
  );
}
