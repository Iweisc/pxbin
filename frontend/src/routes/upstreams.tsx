import { useState } from "react";
import { Plus, X } from "lucide-react";
import { ProtectedRoute } from "../lib/auth.tsx";
import { UpstreamsTable } from "../components/UpstreamsTable.tsx";
import { useUpstreams, useCreateUpstream } from "../hooks/useUpstreams.ts";

export function UpstreamsPage() {
  const [showCreate, setShowCreate] = useState(false);
  const upstreams = useUpstreams();

  return (
    <ProtectedRoute>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-zinc-100">Upstreams</h1>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-emerald-600 hover:bg-emerald-500 rounded-md text-white font-medium transition-colors"
          >
            <Plus size={14} />
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl w-full max-w-md m-4">
        <div className="flex items-center justify-between px-5 py-4 border-b border-zinc-800">
          <h2 className="text-sm font-semibold text-zinc-100">Add Upstream</h2>
          <button
            onClick={onClose}
            className="text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            <X size={16} />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <div>
            <label className="block text-xs text-zinc-400 mb-1.5">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              placeholder="anthropic-primary"
              className="w-full bg-zinc-800 border border-zinc-700 rounded-md text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-600"
            />
          </div>
          <div>
            <label className="block text-xs text-zinc-400 mb-1.5">
              Base URL
            </label>
            <input
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              required
              placeholder="https://api.anthropic.com"
              className="w-full bg-zinc-800 border border-zinc-700 rounded-md text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-600 font-mono"
            />
          </div>
          <div>
            <label className="block text-xs text-zinc-400 mb-1.5">
              API Key
            </label>
            <input
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              required
              type="password"
              placeholder="sk-ant-..."
              className="w-full bg-zinc-800 border border-zinc-700 rounded-md text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-600 font-mono"
            />
          </div>
          <div>
            <label className="block text-xs text-zinc-400 mb-1.5">
              API Format
            </label>
            <select
              value={format}
              onChange={(e) => setFormat(e.target.value)}
              className="w-full bg-zinc-800 border border-zinc-700 rounded-md text-sm text-zinc-200 px-3 py-2 focus:outline-none focus:border-zinc-600"
            >
              <option value="openai">OpenAI Compatible</option>
              <option value="anthropic">Native Anthropic</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-zinc-400 mb-1.5">
              Priority (lower = higher priority)
            </label>
            <input
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
              type="number"
              min="0"
              className="w-full bg-zinc-800 border border-zinc-700 rounded-md text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-600 font-mono"
            />
          </div>
          {create.isError && (
            <p className="text-xs text-red-400">
              {create.error?.message ?? "Failed to create upstream"}
            </p>
          )}
          <button
            type="submit"
            disabled={!name || !baseUrl || !apiKey || create.isPending}
            className="w-full py-2 text-sm bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed rounded-md text-white font-medium transition-colors"
          >
            {create.isPending ? "Creating..." : "Create Upstream"}
          </button>
        </form>
      </div>
    </div>
  );
}
