import { useState } from "react";
import { Plus, X, Activity, Check, Loader2 } from "lucide-react";
import { ProtectedRoute } from "../lib/auth.tsx";
import { UpstreamsTable } from "../components/UpstreamsTable.tsx";
import { useUpstreams, useCreateUpstream, useHealthCheckUpstream } from "../hooks/useUpstreams.ts";
import type { HealthCheckResult } from "../lib/types.ts";

export function UpstreamsPage() {
  const [showCreate, setShowCreate] = useState(false);
  const upstreams = useUpstreams();

  return (
    <ProtectedRoute>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-zinc-100 tracking-tight">Upstreams</h1>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-emerald-600 hover:bg-emerald-500 rounded-lg text-white font-medium transition-all duration-150"
          >
            <Plus size={13} />
            Add Upstream
          </button>
        </div>

        <UpstreamsTable
          data={upstreams.data ?? []}
          isLoading={upstreams.isLoading}
        />

        {showCreate && (
          <CreateUpstreamDialog onClose={() => setShowCreate(false)} />
        )}
      </div>
    </ProtectedRoute>
  );
}

function CreateUpstreamDialog({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [format, setFormat] = useState("openai");
  const [priority, setPriority] = useState("0");
  const create = useCreateUpstream();
  const healthCheck = useHealthCheckUpstream();
  const [healthResult, setHealthResult] = useState<HealthCheckResult | null>(null);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    create.mutate(
      {
        name,
        base_url: baseUrl,
        api_key: apiKey,
        format,
        priority: Number(priority),
      },
      { onSuccess: () => onClose() },
    );
  }

  function handleTestConnection() {
    setHealthResult(null);
    healthCheck.mutate(
      { base_url: baseUrl, api_key: apiKey, format },
      { onSuccess: (data) => setHealthResult(data) },
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-[2px]" onClick={onClose} />
      <div
        className="relative bg-zinc-900/95 border border-zinc-800/40 rounded-xl shadow-2xl w-full max-w-md m-4"
        style={{ animation: "fadeInUp 0.25s ease-out forwards" }}
      >
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800/60">
          <h2 className="text-sm font-semibold text-zinc-100">Add Upstream</h2>
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
              placeholder="anthropic-primary"
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 transition-colors"
            />
          </div>
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
              Base URL
            </label>
            <input
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              required
              placeholder="https://api.anthropic.com"
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
              required
              type="password"
              placeholder="sk-ant-..."
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
              Priority (lower = higher priority)
            </label>
            <input
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
              type="number"
              min="0"
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 font-mono transition-colors"
            />
          </div>
          {healthResult && (
            <div className={`flex items-center gap-2 px-3 py-2 rounded-lg text-xs ${
              healthResult.healthy
                ? "bg-emerald-950/40 border border-emerald-800/30"
                : "bg-red-950/40 border border-red-800/30"
            }`}>
              {healthResult.healthy ? (
                <>
                  <Check size={13} className="text-emerald-400 shrink-0" />
                  <span className="text-emerald-300">
                    Healthy — {healthResult.models_found} models, tested <span className="font-mono">{healthResult.tested_model}</span> in {healthResult.latency_ms}ms
                  </span>
                </>
              ) : (
                <span className="text-red-300">{healthResult.error ?? "Health check failed"}</span>
              )}
            </div>
          )}
          {healthCheck.isError && (
            <p className="text-xs text-red-400">
              {healthCheck.error?.message ?? "Health check request failed"}
            </p>
          )}
          {create.isError && (
            <p className="text-xs text-red-400">
              {create.error?.message ?? "Failed to create upstream"}
            </p>
          )}
          <div className="flex gap-2">
            <button
              type="button"
              onClick={handleTestConnection}
              disabled={!baseUrl || !apiKey || healthCheck.isPending}
              className="flex items-center justify-center gap-1.5 flex-1 py-2 text-xs bg-zinc-800/80 hover:bg-zinc-700/80 border border-zinc-700/50 disabled:opacity-40 disabled:cursor-not-allowed rounded-lg text-zinc-300 font-medium transition-all duration-150"
            >
              {healthCheck.isPending ? (
                <Loader2 size={13} className="animate-spin" />
              ) : (
                <Activity size={13} />
              )}
              {healthCheck.isPending ? "Testing..." : "Test Connection"}
            </button>
            <button
              type="submit"
              disabled={!name || !baseUrl || !apiKey || create.isPending}
              className="flex-1 py-2 text-xs bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 disabled:cursor-not-allowed rounded-lg text-white font-medium transition-all duration-150"
            >
              {create.isPending ? "Creating..." : "Create Upstream"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
