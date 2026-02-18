import { useState, useMemo } from "react";
import { Pencil, Trash2, X } from "lucide-react";
import type { Model, Upstream } from "../lib/types.ts";
import { useUpdateModel, useDeleteModel, useBulkDeleteModels } from "../hooks/useModels.ts";

interface ModelsTableProps {
  data: Model[];
  isLoading: boolean;
  upstreams: Upstream[];
}

function EditModelDialog({ model, upstreams, onClose }: { model: Model; upstreams: Upstream[]; onClose: () => void }) {
  const [name, setName] = useState(model.name);
  const [displayName, setDisplayName] = useState(model.display_name ?? "");
  const [provider, setProvider] = useState(model.provider);
  const [upstreamId, setUpstreamId] = useState(model.upstream_id ?? "");
  const [inputCost, setInputCost] = useState(String(model.input_cost_per_million));
  const [outputCost, setOutputCost] = useState(String(model.output_cost_per_million));
  const [isActive, setIsActive] = useState(model.is_active);
  const update = useUpdateModel();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    update.mutate(
      {
        id: model.id,
        name,
        display_name: displayName || null,
        provider,
        upstream_id: upstreamId || null,
        input_cost_per_million: Number(inputCost),
        output_cost_per_million: Number(outputCost),
        is_active: isActive,
      } as Parameters<typeof update.mutate>[0],
      { onSuccess: () => onClose() },
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
          <h2 className="text-sm font-semibold text-zinc-100">Edit Model</h2>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition-colors"
          >
            <X size={15} />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Model Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 font-mono transition-colors"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Provider</label>
              <input
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                required
                className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 transition-colors"
              />
            </div>
            <div>
              <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Upstream</label>
              <select
                value={upstreamId}
                onChange={(e) => setUpstreamId(e.target.value)}
                className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 focus:outline-none focus:border-zinc-500 transition-colors"
              >
                <option value="">None</option>
                {upstreams.map((u) => (
                  <option key={u.id} value={u.id}>{u.name}</option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Display Name</label>
            <input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Optional display name"
              className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 transition-colors"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Input $/M</label>
              <input
                value={inputCost}
                onChange={(e) => setInputCost(e.target.value)}
                type="number"
                step="0.01"
                min="0"
                className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 font-mono transition-colors"
              />
            </div>
            <div>
              <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Output $/M</label>
              <input
                value={outputCost}
                onChange={(e) => setOutputCost(e.target.value)}
                type="number"
                step="0.01"
                min="0"
                className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-500 font-mono transition-colors"
              />
            </div>
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
              {update.error?.message ?? "Failed to update model"}
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
              disabled={update.isPending}
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

export function ModelsTable({ data, isLoading, upstreams }: ModelsTableProps) {
  const [editModel, setEditModel] = useState<Model | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmBulk, setConfirmBulk] = useState(false);
  const del = useDeleteModel();
  const bulkDel = useBulkDeleteModels();

  const upstreamMap = useMemo(() => {
    const map = new Map<string, Upstream>();
    for (const u of upstreams) {
      map.set(u.id, u);
    }
    return map;
  }, [upstreams]);

  function handleDelete(id: string) {
    del.mutate(id, { onSettled: () => setConfirmDeleteId(null) });
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
      setSelected(new Set(data.map((m) => m.id)));
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
        <p className="text-zinc-600 text-sm">No models configured</p>
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
              <span className="text-[10px] text-zinc-500">Delete {selected.size} model{selected.size !== 1 ? "s" : ""}?</span>
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
            <button
              onClick={() => setConfirmBulk(true)}
              className="flex items-center gap-1 text-[10px] text-red-400 hover:text-red-300 font-medium transition-colors"
            >
              <Trash2 size={11} />
              Delete Selected
            </button>
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
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Display Name</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Provider</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Upstream</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Input $/M</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Output $/M</th>
                <th className="px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider">Status</th>
                <th className="px-4 py-2.5"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800/30">
              {data.map((m) => (
                <tr key={m.id} className="text-zinc-200 hover:bg-zinc-800/20 transition-colors">
                  <td className="px-4 py-2.5 w-10">
                    <input
                      type="checkbox"
                      checked={selected.has(m.id)}
                      onChange={() => toggleSelect(m.id)}
                      className="rounded border-zinc-600 bg-zinc-800 text-emerald-600 focus:ring-0 focus:ring-offset-0 cursor-pointer"
                    />
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="font-mono text-xs">{m.name}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="text-xs text-zinc-400">{m.display_name ?? "-"}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="text-[10px] text-zinc-500 uppercase tracking-wide">{m.provider}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {!m.upstream_id ? (
                      <span className="text-xs text-zinc-600">-</span>
                    ) : (
                      <span className="text-xs text-zinc-400">
                        {upstreamMap.get(m.upstream_id)?.name ?? "unknown"}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="font-mono text-xs text-zinc-300">${m.input_cost_per_million.toFixed(2)}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    <span className="font-mono text-xs text-zinc-300">${m.output_cost_per_million.toFixed(2)}</span>
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {m.is_active ? (
                      <span className="text-[10px] font-medium text-emerald-400">Active</span>
                    ) : (
                      <span className="text-[10px] font-medium text-zinc-600">Inactive</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {confirmDeleteId === m.id ? (
                      <div className="flex items-center gap-2">
                        <button
                          onClick={(e) => { e.stopPropagation(); handleDelete(m.id); }}
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
                        <button
                          onClick={(e) => { e.stopPropagation(); setEditModel(m); }}
                          className="text-zinc-600 hover:text-zinc-400 transition-colors"
                        >
                          <Pencil size={13} />
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setConfirmDeleteId(m.id); }}
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

      {editModel && (
        <EditModelDialog
          model={editModel}
          upstreams={upstreams}
          onClose={() => setEditModel(null)}
        />
      )}
    </div>
  );
}
