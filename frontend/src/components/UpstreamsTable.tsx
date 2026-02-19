import { useState } from "react";
import { Pencil, Trash2, X, Activity, Check, Loader2 } from "lucide-react";
import type { Upstream, HealthCheckResult } from "../lib/types.ts";
import { useUpdateUpstream, useDeleteUpstream, useBulkDeleteUpstreams, useHealthCheckUpstream } from "../hooks/useUpstreams.ts";

interface UpstreamsTableProps {
  data: Upstream[];
  isLoading: boolean;
}

function EditUpstreamDialog({ upstream, onClose }: { upstream: Upstream; onClose: () => void }) {
  const [name, setName] = useState(upstream.name);
  const [baseUrl, setBaseUrl] = useState(upstream.base_url);
  const [apiKey, setApiKey] = useState("");
  const [format, setFormat] = useState(upstream.format);
  const [priority, setPriority] = useState(String(upstream.priority));
  const [isActive, setIsActive] = useState(upstream.is_active);
  const update = useUpdateUpstream();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const data: Record<string, unknown> = {
      id: upstream.id,
      name,
      base_url: baseUrl,
      format,
      priority: Number(priority),
      is_active: isActive,
    };
    if (apiKey) {
      data.api_key = apiKey;
    }
    update.mutate(data as Parameters<typeof update.mutate>[0], {
      onSuccess: () => onClose(),
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-[2px]" onClick={onClose} />
      <div
        className="relative bg-zinc-900/95 border border-zinc-800/40 rounded-xl shadow-2xl w-full max-w-md m-4"
        style={{ animation: "fadeInUp 0.25s ease-out forwards" }}
      >
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800/60">
          <h2 className="text-sm font-semibold text-zinc-100">Edit Upstream</h2>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition-colors"
          >
            <X size={15} />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 transition-colors"
            />
          </div>
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Base URL</label>
            <input
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              required
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 font-mono transition-colors"
            />
          </div>
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
              API Key
            </label>
            <input
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              type="password"
              placeholder="Leave blank to keep existing"
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 font-mono transition-colors"
            />
          </div>
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
              API Format
            </label>
            <select
              value={format}
              onChange={(e) => setFormat(e.target.value)}
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 focus:outline-none focus:border-zinc-500 transition-colors"
            >
              <option value="openai">OpenAI Compatible</option>
              <option value="anthropic">Native Anthropic</option>
            </select>
          </div>
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
              Priority
            </label>
            <input
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
              type="number"
              min="0"
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 font-mono transition-colors"
            />
          </div>
          <div className="flex items-center gap-3">
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider">Status</label>
            <button
              type="button"
              onClick={() => setIsActive(!isActive)}
              className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                isActive ? "bg-emerald-600" : "bg-zinc-700"
              }`}
            >
              <span
                className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                  isActive ? "translate-x-4.5" : "translate-x-0.5"
                }`}
              />
            </button>
            <span className={`text-xs font-medium ${isActive ? "text-emerald-400" : "text-zinc-500"}`}>
              {isActive ? "Active" : "Inactive"}
            </span>
          </div>
          {update.isError && (
            <p className="text-xs text-red-400">
              {update.error?.message ?? "Failed to update upstream"}
            </p>
          )}
          <div className="flex gap-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2 text-xs bg-zinc-800/80 hover:bg-zinc-700/80 border border-zinc-700/50 rounded-lg text-zinc-300 font-medium transition-all duration-150"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!name || !baseUrl || update.isPending}
              className="flex-1 py-2 text-xs bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 disabled:cursor-not-allowed rounded-lg text-white font-medium transition-all duration-150"
            >
              {update.isPending ? "Saving..." : "Save Changes"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export function UpstreamsTable({ data, isLoading }: UpstreamsTableProps) {
  const [editUpstream, setEditUpstream] = useState<Upstream | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmBulk, setConfirmBulk] = useState(false);
  const [healthResults, setHealthResults] = useState<Record<string, HealthCheckResult>>({});
  const [healthChecking, setHealthChecking] = useState<Set<string>>(new Set());
  const [bulkHealthChecking, setBulkHealthChecking] = useState(false);
  const del = useDeleteUpstream();
  const bulkDel = useBulkDeleteUpstreams();
  const healthCheck = useHealthCheckUpstream();

  function handleDelete(id: string) {
    del.mutate(id, { onSettled: () => setConfirmDeleteId(null) });
  }

  function handleHealthCheck(id: string) {
    setHealthChecking((prev) => new Set(prev).add(id));
    healthCheck.mutate(
      { upstream_id: id },
      {
        onSuccess: (result) => {
          setHealthResults((prev) => ({ ...prev, [id]: result }));
        },
        onSettled: () => {
          setHealthChecking((prev) => {
            const next = new Set(prev);
            next.delete(id);
            return next;
          });
        },
      },
    );
  }

  async function handleBulkHealthCheck() {
    setBulkHealthChecking(true);
    const ids = [...selected];
    for (const id of ids) {
      setHealthChecking((prev) => new Set(prev).add(id));
      try {
        const result = await healthCheck.mutateAsync({ upstream_id: id });
        setHealthResults((prev) => ({ ...prev, [id]: result }));
      } catch {
        // individual failure is shown via missing result
      } finally {
        setHealthChecking((prev) => {
          const next = new Set(prev);
          next.delete(id);
          return next;
        });
      }
    }
    setBulkHealthChecking(false);
  }

  function toggleSelect(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleAll() {
    if (selected.size === data.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(data.map((u) => u.id)));
    }
  }

  function handleBulkDelete() {
    bulkDel.mutate([...selected], {
      onSuccess: () => {
        setSelected(new Set());
        setConfirmBulk(false);
      },
    });
  }

  if (isLoading) {
    return (
      <div className="bg-zinc-900/40 border border-zinc-800/30 rounded-xl overflow-hidden">
        <div className="animate-pulse p-4 space-y-3">
          <div className="h-4 bg-zinc-800/60 rounded w-full" />
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-8 bg-zinc-800/30 rounded w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl flex items-center justify-center py-16">
        <p className="text-zinc-600 text-sm">No upstreams configured</p>
      </div>
    );
  }

  return (
    <div>
      {selected.size > 0 && (
        <div className="flex items-center gap-3 mb-3 px-4 py-2 bg-zinc-900/60 border border-zinc-800/40 rounded-xl">
          <span className="text-[10px] text-zinc-400 font-medium">
            {selected.size} selected
          </span>
          {confirmBulk ? (
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-zinc-500">Delete {selected.size} upstream{selected.size !== 1 ? "s" : ""}?</span>
              <button
                onClick={handleBulkDelete}
                disabled={bulkDel.isPending}
                className="text-[10px] text-red-400 hover:text-red-300 font-medium transition-colors"
              >
                {bulkDel.isPending ? "Deleting..." : "Confirm"}
              </button>
              <button
                onClick={() => setConfirmBulk(false)}
                className="text-[10px] text-zinc-600 hover:text-zinc-400 transition-colors"
              >
                Cancel
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-3">
              <button
                onClick={handleBulkHealthCheck}
                disabled={bulkHealthChecking}
                className="flex items-center gap-1 text-[10px] text-zinc-400 hover:text-zinc-200 font-medium transition-colors disabled:opacity-40"
              >
                {bulkHealthChecking ? <Loader2 size={11} className="animate-spin" /> : <Activity size={11} />}
                {bulkHealthChecking ? "Checking..." : "Check Health"}
              </button>
              <button
                onClick={() => setConfirmBulk(true)}
                className="flex items-center gap-1 text-[10px] text-red-400 hover:text-red-300 font-medium transition-colors"
              >
                <Trash2 size={11} />
                Delete Selected
              </button>
            </div>
          )}
          <button
            onClick={() => setSelected(new Set())}
            className="ml-auto text-[10px] text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            Clear
          </button>
        </div>
      )}
      <div
        className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl overflow-hidden"
        style={{ animation: "fadeIn 0.4s ease-out forwards" }}
      >
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-800/60">
                <th className="px-4 py-2.5 w-10">
                  <input
                    type="checkbox"
                    checked={data.length > 0 && selected.size === data.length}
                    onChange={toggleAll}
                    className="rounded border-zinc-600 bg-zinc-800 text-emerald-600 focus:ring-0 focus:ring-offset-0 cursor-pointer"
                  />
                </th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Name</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Base URL</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Format</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Priority</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Status</th>
                <th className="px-4 py-2.5"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800/30">
              {data.map((u) => (
                <tr key={u.id} className="text-zinc-200 hover:bg-zinc-800/20 transition-colors">
                  <td className="px-4 py-2.5 w-10">
                    <input
                      type="checkbox"
                      checked={selected.has(u.id)}
                      onChange={() => toggleSelect(u.id)}
                      className="rounded border-zinc-600 bg-zinc-800 text-emerald-600 focus:ring-0 focus:ring-offset-0 cursor-pointer"
                    />
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="text-xs font-medium text-zinc-200">{u.name}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="font-mono text-xs text-zinc-500">{u.base_url}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span
                      className={`text-[10px] font-medium px-1.5 py-0.5 rounded-md ${
                        u.format === "anthropic"
                          ? "bg-amber-900/20 text-amber-400/80"
                          : "bg-blue-900/20 text-blue-400/80"
                      }`}
                    >
                      {u.format}
                    </span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="font-mono text-xs text-zinc-400">{u.priority}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {u.is_active ? (
                      <span className="text-[10px] font-medium text-emerald-400">Active</span>
                    ) : (
                      <span className="text-[10px] font-medium text-zinc-600">Inactive</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {confirmDeleteId === u.id ? (
                      <div className="flex items-center gap-2">
                        <button
                          onClick={(e) => { e.stopPropagation(); handleDelete(u.id); }}
                          className="text-[10px] text-red-400 hover:text-red-300 font-medium transition-colors"
                        >
                          Confirm
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setConfirmDeleteId(null); }}
                          className="text-[10px] text-zinc-600 hover:text-zinc-400 transition-colors"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        {healthChecking.has(u.id) ? (
                          <Loader2 size={13} className="text-zinc-500 animate-spin" />
                        ) : healthResults[u.id] ? (
                          <button
                            onClick={(e) => { e.stopPropagation(); handleHealthCheck(u.id); }}
                            className="relative group"
                            title={healthResults[u.id].healthy
                              ? `Healthy — ${healthResults[u.id].models_found} models, ${healthResults[u.id].tested_model} (${healthResults[u.id].latency_ms}ms)`
                              : healthResults[u.id].error ?? "Unhealthy"}
                          >
                            {healthResults[u.id].healthy ? (
                              <Check size={13} className="text-emerald-400" />
                            ) : (
                              <Activity size={13} className="text-red-400" />
                            )}
                          </button>
                        ) : (
                          <button
                            onClick={(e) => { e.stopPropagation(); handleHealthCheck(u.id); }}
                            className="text-zinc-600 hover:text-zinc-400 transition-colors"
                            title="Check health"
                          >
                            <Activity size={13} />
                          </button>
                        )}
                        <button
                          onClick={(e) => { e.stopPropagation(); setEditUpstream(u); }}
                          className="text-zinc-600 hover:text-zinc-400 transition-colors"
                        >
                          <Pencil size={13} />
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setConfirmDeleteId(u.id); }}
                          className="text-zinc-600 hover:text-red-400 transition-colors"
                        >
                          <Trash2 size={13} />
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {editUpstream && (
        <EditUpstreamDialog
          upstream={editUpstream}
          onClose={() => setEditUpstream(null)}
        />
      )}
    </div>
  );
}
