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
    <div className="flex items-center justify-between px-1 py-2.5">
      <div className="text-[10px] text-zinc-600 font-mono">
        {total > 0 ? (
          <>
            {from}–{to} of {total}
          </>
        ) : (
          "0 results"
        )}
      </div>
      <div className="flex items-center gap-2.5">
        <select
          value={perPage}
          onChange={(e) => {
            onPerPageChange(Number(e.target.value));
            onPageChange(1);
          }}
          className="bg-zinc-900/80 border border-zinc-800/50 rounded-lg text-[10px] text-zinc-400 px-2 py-1 focus:outline-none focus:border-zinc-700"
        >
          {PER_PAGE_OPTIONS.map((n) => (
            <option key={n} value={n}>
              {n} / page
            </option>
          ))}
        </select>
        <div className="flex gap-0.5">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            className="px-2.5 py-1 text-[10px] bg-zinc-900/80 hover:bg-zinc-800/60 disabled:opacity-30 disabled:cursor-not-allowed rounded-md text-zinc-400 transition-all duration-150 border border-zinc-800/40"
          >
            Prev
          </button>
          {totalPages <= 7 ? (
            Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
              <button
                key={p}
                onClick={() => onPageChange(p)}
                className={`px-2.5 py-1 text-[10px] rounded-md transition-all duration-150 border ${
                  p === page
                    ? "bg-zinc-700/80 text-zinc-100 border-zinc-700/60 shadow-sm"
                    : "bg-zinc-900/80 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800/60 border-zinc-800/40"
                }`}
              >
                {p}
              </button>
            ))
          ) : (
            <span className="px-2 py-1 text-[10px] text-zinc-600 font-mono">
              {page} / {totalPages}
            </span>
          )}
          <button
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="px-2.5 py-1 text-[10px] bg-zinc-900/80 hover:bg-zinc-800/60 disabled:opacity-30 disabled:cursor-not-allowed rounded-md text-zinc-400 transition-all duration-150 border border-zinc-800/40"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
}
