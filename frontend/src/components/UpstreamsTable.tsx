import { useState } from "react";
import { Pencil, Trash2, Check, X } from "lucide-react";
import type { Upstream } from "../lib/types.ts";
import { useUpdateUpstream, useDeleteUpstream, useBulkDeleteUpstreams } from "../hooks/useUpstreams.ts";

interface UpstreamsTableProps {
  data: Upstream[];
  isLoading: boolean;
}

export function UpstreamsTable({ data, isLoading }: UpstreamsTableProps) {
  const [editId, setEditId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editPriority, setEditPriority] = useState("");
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmBulk, setConfirmBulk] = useState(false);
  const update = useUpdateUpstream();
  const del = useDeleteUpstream();
  const bulkDel = useBulkDeleteUpstreams();

  function startEdit(u: Upstream) {
    setEditId(u.id);
    setEditName(u.name);
    setEditPriority(String(u.priority));
  }

  function saveEdit(id: string) {
    update.mutate(
      { id, name: editName, priority: Number(editPriority) },
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
                    {editId === u.id ? (
                      <input
                        value={editName}
                        onChange={(e) => setEditName(e.target.value)}
                        className="w-32 bg-zinc-800/60 border border-zinc-600/50 rounded-lg text-xs text-zinc-200 px-2 py-1 focus:outline-none focus:border-zinc-500"
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <span className="text-xs font-medium text-zinc-200">{u.name}</span>
                    )}
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
                    {editId === u.id ? (
                      <input
                        value={editPriority}
                        onChange={(e) => setEditPriority(e.target.value)}
                        type="number"
                        min="0"
                        className="w-16 bg-zinc-800/60 border border-zinc-600/50 rounded-lg text-xs text-zinc-200 px-2 py-1 font-mono focus:outline-none focus:border-zinc-500"
                        onClick={(e) => e.stopPropagation()}
                      />
                    ) : (
                      <span className="font-mono text-xs text-zinc-400">{u.priority}</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {u.is_active ? (
                      <span className="text-[10px] font-medium text-emerald-400">Active</span>
                    ) : (
                      <span className="text-[10px] font-medium text-zinc-600">Inactive</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 whitespace-nowrap">
                    {editId === u.id ? (
                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={(e) => { e.stopPropagation(); saveEdit(u.id); }}
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
                    ) : confirmDeleteId === u.id ? (
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
                        <button
                          onClick={(e) => { e.stopPropagation(); startEdit(u); }}
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
    </div>
  );
}
