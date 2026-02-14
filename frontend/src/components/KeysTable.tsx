import { useState } from "react";
import { Copy, Trash2 } from "lucide-react";
import { DataTable, type Column } from "./DataTable.tsx";
import type { LLMAPIKey, ManagementAPIKey } from "../lib/types.ts";
import { formatDate } from "../lib/utils.ts";
import { useRevokeKey } from "../hooks/useKeys.ts";

type AnyKey = LLMAPIKey | ManagementAPIKey;

interface KeysTableProps {
  data: AnyKey[];
  isLoading: boolean;
}

export function KeysTable({ data, isLoading }: KeysTableProps) {
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const revoke = useRevokeKey();

  function handleRevoke(id: string) {
    revoke.mutate(id, { onSettled: () => setConfirmId(null) });
  }

  const columns: Column<AnyKey>[] = [
    {
      key: "key_prefix",
      header: "Prefix",
      render: (k) => (
        <div className="flex items-center gap-1.5">
          <span className="font-mono text-xs text-zinc-300">{k.key_prefix}...</span>
          <button
            onClick={(e) => {
              e.stopPropagation();
              navigator.clipboard.writeText(k.key_prefix);
            }}
            className="text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            <Copy size={11} />
          </button>
        </div>
      ),
    },
    {
      key: "name",
      header: "Name",
      render: (k) => <span className="text-xs text-zinc-200">{k.name}</span>,
    },
    {
      key: "created_at",
      header: "Created",
      render: (k) => (
        <span className="text-xs text-zinc-500 font-mono">{formatDate(k.created_at)}</span>
      ),
    },
    {
      key: "last_used_at",
      header: "Last Used",
      render: (k) => (
        <span className="text-xs text-zinc-500 font-mono">
          {k.last_used_at ? formatDate(k.last_used_at) : "Never"}
        </span>
      ),
    },
    {
      key: "is_active",
      header: "Status",
      render: (k) =>
        k.is_active ? (
          <span className="text-[10px] font-medium text-emerald-400">Active</span>
        ) : (
          <span className="text-[10px] font-medium text-zinc-600">Revoked</span>
        ),
    },
    {
      key: "actions",
      header: "",
      render: (k) =>
        k.is_active ? (
          confirmId === k.id ? (
            <div className="flex items-center gap-2">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleRevoke(k.id);
                }}
                className="text-[10px] text-red-400 hover:text-red-300 font-medium transition-colors"
              >
                Confirm
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setConfirmId(null);
                }}
                className="text-[10px] text-zinc-600 hover:text-zinc-400 transition-colors"
              >
                Cancel
              </button>
            </div>
          ) : (
            <button
              onClick={(e) => {
                e.stopPropagation();
                setConfirmId(k.id);
              }}
              className="text-zinc-600 hover:text-red-400 transition-colors"
            >
              <Trash2 size={13} />
            </button>
          )
        ) : null,
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={data}
      isLoading={isLoading}
      emptyMessage="No keys found"
    />
  );
}
