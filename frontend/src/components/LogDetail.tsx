import { X } from "lucide-react";
import type { RequestLog } from "../lib/types.ts";
import { formatDate, formatCost, formatDuration, formatTokens } from "../lib/utils.ts";

interface LogDetailProps {
  log: RequestLog;
  onClose: () => void;
}

function Field({ label, value }: { label: string; value: string | null | undefined }) {
  return (
    <div>
      <dt className="text-xs text-zinc-500 mb-0.5">{label}</dt>
      <dd className="text-sm font-mono text-zinc-200">{value ?? "-"}</dd>
    </div>
  );
}

export function LogDetail({ log, onClose }: LogDetailProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl w-full max-w-2xl max-h-[80vh] overflow-y-auto m-4">
        <div className="flex items-center justify-between px-5 py-4 border-b border-zinc-800 sticky top-0 bg-zinc-900">
          <h2 className="text-sm font-semibold text-zinc-100">Request Detail</h2>
          <button
            onClick={onClose}
            className="text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            <X size={16} />
          </button>
        </div>
        <div className="p-5 space-y-5">
          <div className="grid grid-cols-2 gap-4">
            <Field label="ID" value={log.id} />
            <Field label="Timestamp" value={formatDate(log.timestamp)} />
            <Field label="Method" value={log.method} />
            <Field label="Path" value={log.path} />
            <Field label="Model" value={log.model} />
            <Field label="Format" value={log.input_format} />
            <Field
              label="Status"
              value={log.status_code != null ? String(log.status_code) : null}
            />
            <Field
              label="Latency"
              value={log.latency_ms != null ? formatDuration(log.latency_ms) : null}
            />
            <Field
              label="Input Tokens"
              value={log.input_tokens != null ? formatTokens(log.input_tokens) : null}
            />
            <Field
              label="Output Tokens"
              value={log.output_tokens != null ? formatTokens(log.output_tokens) : null}
            />
            <Field
              label="Cost"
              value={log.cost != null ? formatCost(log.cost) : null}
            />
            <Field label="Key ID" value={log.llm_key_id} />
            <Field label="Upstream ID" value={log.upstream_id} />
          </div>
          {log.error_message && (
            <div>
              <dt className="text-xs text-zinc-500 mb-1">Error</dt>
              <dd className="text-sm font-mono text-red-400 bg-zinc-950 rounded-md p-3 break-all">
                {log.error_message}
              </dd>
            </div>
          )}
          {log.request_metadata &&
            Object.keys(log.request_metadata).length > 0 && (
              <div>
                <dt className="text-xs text-zinc-500 mb-1">Metadata</dt>
                <dd className="text-xs font-mono text-zinc-300 bg-zinc-950 rounded-md p-3 whitespace-pre-wrap break-all">
                  {JSON.stringify(log.request_metadata, null, 2)}
                </dd>
              </div>
            )}
        </div>
      </div>
    </div>
  );
}
