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
        <p className="text-zinc-500 text-sm">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-lg overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-800">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-4 py-3 text-left text-xs font-medium text-zinc-400 uppercase tracking-wider ${col.className ?? ""}`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-800/50">
            {data.map((item, i) => (
              <tr
                key={i}
                onClick={() => onRowClick?.(item)}
                className={`text-zinc-100 transition-colors ${
                  onRowClick
                    ? "cursor-pointer hover:bg-zinc-800/50"
                    : "hover:bg-zinc-800/30"
                }`}
              >
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className={`px-4 py-3 whitespace-nowrap ${col.className ?? ""}`}
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
