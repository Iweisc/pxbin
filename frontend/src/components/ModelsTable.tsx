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
                    {editId === m.id ? (
                      <input
                        value={editInputCost}
                        onChange={(e) => setEditInputCost(e.target.value)}
                        type="number"
                        step="0.01"
                        className="w-20 bg-zinc-800/60 border border-zinc-600/50 rounded-lg text-xs text-zinc-200 px-2 py-1 font-mono focus:outline-none focus:border-zinc-500"
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <span className="font-mono text-xs text-zinc-300">${m.input_cost_per_million.toFixed(2)}</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {editId === m.id ? (
                      <input
                        value={editOutputCost}
                        onChange={(e) => setEditOutputCost(e.target.value)}
                        type="number"
                        step="0.01"
                        className="w-20 bg-zinc-800/60 border border-zinc-600/50 rounded-lg text-xs text-zinc-200 px-2 py-1 font-mono focus:outline-none focus:border-zinc-500"
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <span className="font-mono text-xs text-zinc-300">${m.output_cost_per_million.toFixed(2)}</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {m.is_active ? (
                      <span className="text-[10px] font-medium text-emerald-400">Active</span>
                    ) : (
                      <span className="text-[10px] font-medium text-zinc-600">Inactive</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {editId === m.id ? (
                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={(e) => { e.stopPropagation(); saveEdit(m.id); }}
                          className="text-emerald-400 hover:text-emerald-300 transition-colors"
                        >
                          <Check size={13} />
                        </button>
                        <button
                          onClick={(e) => { e.stopPropagation(); setEditId(null); }}
                          className="text-zinc-600 hover:text-zinc-400 transition-colors"
                        >
                          <X size={13} />
                        </button>
                      </div>
                    ) : confirmDeleteId === m.id ? (
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
                          onClick={(e) => { e.stopPropagation(); startEdit(m); }}
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
    </div>
  );
}
