import { useState, useMemo } from "react";
import { Pencil, Trash2, Check, X } from "lucide-react";
import type { Model, Upstream } from "../lib/types.ts";
import { useUpdateModel, useDeleteModel, useBulkDeleteModels } from "../hooks/useModels.ts";

interface ModelsTableProps {
  data: Model[];
  isLoading: boolean;
  upstreams: Upstream[];
}

export function ModelsTable({ data, isLoading, upstreams }: ModelsTableProps) {
  const [editId, setEditId] = useState<string | null>(null);
  const [editInputCost, setEditInputCost] = useState("");
  const [editOutputCost, setEditOutputCost] = useState("");
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmBulk, setConfirmBulk] = useState(false);
  const update = useUpdateModel();
  const del = useDeleteModel();
  const bulkDel = useBulkDeleteModels();

  const upstreamMap = useMemo(() => {
    const map = new Map<string, Upstream>();
    for (const u of upstreams) {
      map.set(u.id, u);
    }
    return map;
  }, [upstreams]);

  function startEdit(m: Model) {
    setEditId(m.id);
    setEditInputCost(String(m.input_cost_per_million));
    setEditOutputCost(String(m.output_cost_per_million));
  }

  function saveEdit(id: string) {
    update.mutate(
      {
        id,
        input_cost_per_million: Number(editInputCost),
        output_cost_per_million: Number(editOutputCost),
      },
      { onSuccess: () => setEditId(null) },
    );
  }

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
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
        <div className="animate-pulse p-4 space-y-3">
          <div className="h-4 bg-zinc-800 rounded w-full" />
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-8 bg-zinc-800/50 rounded w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg flex items-center justify-center py-16">
        <p className="text-zinc-500 text-sm">No models configured</p>
      </div>
    );
  }

  return (
    <div>
      {selected.size > 0 && (
        <div className="flex items-center gap-3 mb-3 px-4 py-2.5 bg-zinc-900 border border-zinc-800 rounded-lg">
          <span className="text-xs text-zinc-300">
            {selected.size} selected
          </span>
          {confirmBulk ? (
            <div className="flex items-center gap-2">
              <span className="text-xs text-zinc-400">Delete {selected.size} model{selected.size !== 1 ? "s" : ""}?</span>
              <button
                onClick={handleBulkDelete}
                disabled={bulkDel.isPending}
                className="text-xs text-red-400 hover:text-red-300 font-medium transition-colors"
              >
                {bulkDel.isPending ? "Deleting..." : "Confirm"}
              </button>
              <button
                onClick={() => setConfirmBulk(false)}
                className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
              >
                Cancel
              </button>
            </div>
          ) : (
            <button
              onClick={() => setConfirmBulk(true)}
              className="flex items-center gap-1 text-xs text-red-400 hover:text-red-300 font-medium transition-colors"
            >
              <Trash2 size={12} />
              Delete Selected
            </button>
          )}
          <button
            onClick={() => setSelected(new Set())}
            className="ml-auto text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
          >
            Clear
          </button>
        </div>
      )}
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-800">
                <th className="px-4 py-3 w-10">
                  <input
                    type="checkbox"
                    checked={data.length > 0 && selected.size === data.length}
                    onChange={toggleAll}
                    className="rounded border-zinc-600 bg-zinc-800 text-emerald-600 focus:ring-0 focus:ring-offset-0 cursor-pointer"
                  />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider">Name</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider">Display Name</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider">Provider</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider">Upstream</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider">Input $/M</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider">Output $/M</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider">Status</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800/50">
              {data.map((m) => (
                <tr key={m.id} className="text-zinc-100 hover:bg-zinc-800/30 transition-colors">
                  <td className="px-4 py-3 w-10">
                    <input
                      type="checkbox"
                      checked={selected.has(m.id)}
                      onChange={() => toggleSelect(m.id)}
                      className="rounded border-zinc-600 bg-zinc-800 text-emerald-600 focus:ring-0 focus:ring-offset-0 cursor-pointer"
                    />
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span className="font-mono text-xs">{m.name}</span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span className="text-sm">{m.display_name ?? "-"}</span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span className="text-xs text-zinc-400 uppercase">{m.provider}</span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    {!m.upstream_id ? (
                      <span className="text-xs text-zinc-500">-</span>
                    ) : (
                      <span className="text-xs text-zinc-300">
                        {upstreamMap.get(m.upstream_id)?.name ?? "unknown"}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    {editId === m.id ? (
                      <input
                        value={editInputCost}
                        onChange={(e) => setEditInputCost(e.target.value)}
                        type="number"
                        step="0.01"
                        className="w-20 bg-zinc-800 border border-zinc-600 rounded text-xs text-zinc-200 px-2 py-1 font-mono"
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <span className="font-mono text-xs">${m.input_cost_per_million.toFixed(2)}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    {editId === m.id ? (
                      <input
                        value={editOutputCost}
                        onChange={(e) => setEditOutputCost(e.target.value)}
                        type="number"
                        step="0.01"
                        className="w-20 bg-zinc-800 border border-zinc-600 rounded text-xs text-zinc-200 px-2 py-1 font-mono"
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <span className="font-mono text-xs">${m.output_cost_per_million.toFixed(2)}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    {m.is_active ? (
                      <span className="text-xs font-medium text-emerald-400">Active</span>
                    ) : (
                      <span className="text-xs font-medium text-zinc-500">Inactive</span>
                    )}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    {editId === m.id ? (
                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={(e) => { e.stopPropagation(); saveEdit(m.id); }}
                          className="text-emerald-400 hover:text-emerald-300 transition-colors"
                        >
                          <Check size={14} />
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setEditId(null); }}
                          className="text-zinc-500 hover:text-zinc-300 transition-colors"
                        >
                          <X size={14} />
                        </button>
                      </div>
                    ) : confirmDeleteId === m.id ? (
                      <div className="flex items-center gap-2">
                        <button
                          onClick={(e) => { e.stopPropagation(); handleDelete(m.id); }}
                          className="text-xs text-red-400 hover:text-red-300 font-medium transition-colors"
                        >
                          Confirm
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setConfirmDeleteId(null); }}
                          className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={(e) => { e.stopPropagation(); startEdit(m); }}
                          className="text-zinc-500 hover:text-zinc-300 transition-colors"
                        >
                          <Pencil size={14} />
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setConfirmDeleteId(m.id); }}
                          className="text-zinc-500 hover:text-red-400 transition-colors"
                        >
                          <Trash2 size={14} />
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
    </div>
  );
}
