interface PaginationProps {
  page: number;
  perPage: number;
  total: number;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
}

const PER_PAGE_OPTIONS = [10, 20, 50, 100];

export function Pagination({
  page,
  perPage,
  total,
  onPageChange,
  onPerPageChange,
}: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / perPage));
  const from = (page - 1) * perPage + 1;
  const to = Math.min(page * perPage, total);

  return (
    <div className="flex items-center justify-between px-1 py-3">
      <div className="text-xs text-zinc-500">
        {total > 0 ? (
          <>
            {from}-{to} of {total}
          </>
        ) : (
          "0 results"
        )}
      </div>
      <div className="flex items-center gap-3">
        <select
          value={perPage}
          onChange={(e) => {
            onPerPageChange(Number(e.target.value));
            onPageChange(1);
          }}
          className="bg-zinc-800 border border-zinc-700 rounded-md text-xs text-zinc-300 px-2 py-1"
        >
          {PER_PAGE_OPTIONS.map((n) => (
            <option key={n} value={n}>
              {n} / page
            </option>
          ))}
        </select>
        <div className="flex gap-1">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            className="px-2.5 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed rounded-md text-zinc-300 transition-colors"
          >
            Prev
          </button>
          {totalPages <= 7 ? (
            Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
              <button
                key={p}
                onClick={() => onPageChange(p)}
                className={`px-2.5 py-1 text-xs rounded-md transition-colors ${
                  p === page
                    ? "bg-zinc-700 text-zinc-100"
                    : "bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-200"
                }`}
              >
                {p}
              </button>
            ))
          ) : (
            <span className="px-2 py-1 text-xs text-zinc-500">
              {page} / {totalPages}
            </span>
          )}
          <button
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="px-2.5 py-1 text-xs bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed rounded-md text-zinc-300 transition-colors"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
}
