import { DataTable, type Column } from "./DataTable.tsx";
import type { RequestLog } from "../lib/types.ts";
import { formatDate, formatCost, formatDuration, formatTokens } from "../lib/utils.ts";

function statusColor(code: number | null): string {
  if (!code) return "text-zinc-500";
  if (code < 400) return "text-emerald-400";
  if (code < 500) return "text-amber-400";
  return "text-red-400";
}

const columns: Column<RequestLog>[] = [
  {
    key: "timestamp",
    header: "Timestamp",
    render: (r) => (
      <span className="font-mono text-xs">{formatDate(r.timestamp)}</span>
    ),
  },
  {
    key: "model",
    header: "Model",
    render: (r) => (
      <span className="font-mono text-xs">{r.model ?? "-"}</span>
    ),
  },
  {
    key: "input_format",
    header: "Format",
    render: (r) => (
      <span className="text-xs uppercase tracking-wide text-zinc-400">
        {r.input_format}
      </span>
    ),
  },
  {
    key: "status_code",
    header: "Status",
    render: (r) => (
      <span className={`font-mono text-xs font-medium ${statusColor(r.status_code)}`}>
        {r.status_code ?? "-"}
      </span>
    ),
  },
  {
    key: "latency_ms",
    header: "Latency",
    render: (r) => (
      <span className="font-mono text-xs">
        {r.latency_ms != null ? formatDuration(r.latency_ms) : "-"}
      </span>
    ),
  },
  {
    key: "tokens_in",
    header: "Tokens In",
    render: (r) => (
      <span className="font-mono text-xs">
        {r.input_tokens != null ? formatTokens(r.input_tokens) : "-"}
      </span>
    ),
  },
  {
    key: "tokens_out",
    header: "Tokens Out",
    render: (r) => (
      <span className="font-mono text-xs">
        {r.output_tokens != null ? formatTokens(r.output_tokens) : "-"}
      </span>
    ),
  },
  {
    key: "cost",
    header: "Cost",
    render: (r) => (
      <span className="font-mono text-xs">
        {r.cost != null ? formatCost(r.cost) : "-"}
      </span>
    ),
  },
];

interface LogsTableProps {
  data: RequestLog[];
  isLoading: boolean;
  onRowClick: (log: RequestLog) => void;
}

export function LogsTable({ data, isLoading, onRowClick }: LogsTableProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      isLoading={isLoading}
      onRowClick={onRowClick}
      emptyMessage="No logs found"
    />
  );
}
