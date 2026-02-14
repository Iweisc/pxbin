interface DateRangePickerProps {
  from: string;
  to: string;
  onChange: (from: string, to: string) => void;
}

const PRESETS: { label: string; hours: number }[] = [
  { label: "24h", hours: 24 },
  { label: "7d", hours: 168 },
  { label: "30d", hours: 720 },
];

function toLocalDatetime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const offset = d.getTimezoneOffset();
  const local = new Date(d.getTime() - offset * 60_000);
  return local.toISOString().slice(0, 16);
}

export function DateRangePicker({ from, to, onChange }: DateRangePickerProps) {
  function applyPreset(hours: number) {
    const now = new Date();
    const start = new Date(now.getTime() - hours * 3600_000);
    onChange(start.toISOString(), now.toISOString());
  }

  return (
    <div className="flex items-center gap-2">
      <div className="flex gap-0.5 bg-zinc-900/80 rounded-lg p-0.5 border border-zinc-800/50">
        {PRESETS.map(({ label, hours }) => (
          <button
            key={label}
            onClick={() => applyPreset(hours)}
            className="px-2.5 py-1 text-[10px] font-medium text-zinc-500 hover:text-zinc-300 rounded-md transition-all duration-150"
          >
            {label}
          </button>
        ))}
      </div>
      <input
        type="datetime-local"
        value={toLocalDatetime(from)}
        onChange={(e) => {
          const val = e.target.value;
          if (val) onChange(new Date(val).toISOString(), to);
        }}
        className="bg-zinc-900/80 border border-zinc-800/50 rounded-lg text-[10px] text-zinc-400 px-2 py-1 focus:outline-none focus:border-zinc-700"
      />
      <span className="text-zinc-600 text-[10px]">to</span>
      <input
        type="datetime-local"
        value={toLocalDatetime(to)}
        onChange={(e) => {
          const val = e.target.value;
          if (val) onChange(from, new Date(val).toISOString());
        }}
        className="bg-zinc-900/80 border border-zinc-800/50 rounded-lg text-[10px] text-zinc-400 px-2 py-1 focus:outline-none focus:border-zinc-700"
      />
    </div>
  );
}
