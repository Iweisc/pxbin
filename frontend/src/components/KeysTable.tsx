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
          <span className="font-mono text-xs">{k.key_prefix}...</span>
          <button
            onClick={(e) => {
              e.stopPropagation();
              navigator.clipboard.writeText(k.key_prefix);
            }}
            className="text-zinc-500 hover:text-zinc-300 transition-colors"
          >
            <Copy size={12} />
          </button>
        </div>
      ),
    },
    {
      key: "name",
      header: "Name",
      render: (k) => <span className="text-sm">{k.name}</span>,
    },
    {
      key: "created_at",
      header: "Created",
      render: (k) => (
        <span className="text-xs text-zinc-400">{formatDate(k.created_at)}</span>
      ),
    },
    {
      key: "last_used_at",
      header: "Last Used",
      render: (k) => (
        <span className="text-xs text-zinc-400">
          {k.last_used_at ? formatDate(k.last_used_at) : "Never"}
        </span>
      ),
    },
    {
      key: "is_active",
      header: "Status",
      render: (k) =>
        k.is_active ? (
          <span className="text-xs font-medium text-emerald-400">Active</span>
        ) : (
          <span className="text-xs font-medium text-zinc-500">Revoked</span>
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
                className="text-xs text-red-400 hover:text-red-300 font-medium transition-colors"
              >
                Confirm
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setConfirmId(null);
                }}
                className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
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
              className="text-zinc-500 hover:text-red-400 transition-colors"
            >
              <Trash2 size={14} />
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
