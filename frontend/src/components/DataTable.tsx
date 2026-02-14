import type { ReactNode } from "react";

export interface Column<T> {
  key: string;
  header: string;
  render?: (item: T) => ReactNode;
  className?: string;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  onRowClick?: (item: T) => void;
  emptyMessage?: string;
  isLoading?: boolean;
}

export function DataTable<T>({
  columns,
  data,
  onRowClick,
  emptyMessage = "No data",
  isLoading,
}: DataTableProps<T>) {
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
        <p className="text-zinc-600 text-sm">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div
      className="bg-zinc-900/60 border border-zinc-800/40 rounded-xl overflow-hidden"
      style={{ animation: "fadeIn 0.4s ease-out forwards" }}
    >
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-800/60">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-4 py-2.5 text-left text-[10px] font-medium text-zinc-500 uppercase tracking-wider ${col.className ?? ""}`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-800/30">
            {data.map((item, i) => (
              <tr
                key={i}
                onClick={() => onRowClick?.(item)}
                className={`text-zinc-200 transition-colors ${
                  onRowClick
                    ? "cursor-pointer hover:bg-zinc-800/30"
                    : "hover:bg-zinc-800/20"
                }`}
              >
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className={`px-4 py-2.5 whitespace-nowrap ${col.className ?? ""}`}
                  >
                    {col.render
                      ? col.render(item)
                      : String((item as Record<string, unknown>)[col.key] ?? "")}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
