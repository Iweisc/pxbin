import { useState } from "react";
import { X, Copy, AlertTriangle } from "lucide-react";
import { useCreateKey } from "../hooks/useKeys.ts";

interface CreateKeyDialogProps {
  type: "llm" | "management";
  onClose: () => void;
}

export function CreateKeyDialog({ type, onClose }: CreateKeyDialogProps) {
  const [name, setName] = useState("");
  const [rateLimit, setRateLimit] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const create = useCreateKey(type);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    create.mutate(
      {
        name,
        rate_limit: type === "llm" && rateLimit ? Number(rateLimit) : undefined,
      },
      {
        onSuccess: (data) => setCreatedKey(data.key),
      },
    );
  }

  function handleCopy() {
    if (createdKey) {
      navigator.clipboard.writeText(createdKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-[2px]" onClick={onClose} />
      <div
        className="relative bg-zinc-900/95 border border-zinc-800/40 rounded-xl shadow-2xl w-full max-w-md m-4"
        style={{ animation: "fadeInUp 0.25s ease-out forwards" }}
      >
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800/60">
          <h2 className="text-sm font-semibold text-zinc-100">
            Create {type === "llm" ? "LLM" : "Management"} Key
          </h2>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition-colors"
          >
            <X size={15} />
          </button>
        </div>

        {createdKey ? (
          <div className="p-5 space-y-4">
            <div className="flex items-start gap-2.5 p-3 bg-amber-950/20 border border-amber-800/30 rounded-lg">
              <AlertTriangle size={14} className="text-amber-400 mt-0.5 shrink-0" />
              <p className="text-xs text-amber-200/80">
                This key will only be shown once. Copy it now.
              </p>
            </div>
            <div className="flex items-center gap-2 bg-zinc-950/80 border border-zinc-800/40 rounded-lg p-3">
              <code className="text-xs text-zinc-200 break-all flex-1 font-mono">
                {createdKey}
              </code>
              <button
                onClick={handleCopy}
                className="shrink-0 text-zinc-500 hover:text-zinc-300 transition-colors"
              >
                <Copy size={13} />
              </button>
            </div>
            {copied && (
              <p className="text-xs text-emerald-400">Copied to clipboard</p>
            )}
            <button
              onClick={onClose}
              className="w-full py-2 text-xs bg-zinc-800/80 hover:bg-zinc-700/80 rounded-lg text-zinc-300 transition-all duration-150"
            >
              Done
            </button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="p-5 space-y-4">
            <div>
              <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">Name</label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                placeholder="My API key"
                className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-600 transition-colors"
              />
            </div>
            {type === "llm" && (
              <div>
                <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
                  Rate Limit (req/min, optional)
                </label>
                <input
                  value={rateLimit}
                  onChange={(e) => setRateLimit(e.target.value)}
                  type="number"
                  min="1"
                  placeholder="60"
                  className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-600 font-mono transition-colors"
                />
              </div>
            )}
            {create.isError && (
              <p className="text-xs text-red-400">
                {create.error?.message ?? "Failed to create key"}
              </p>
            )}
            <button
              type="submit"
              disabled={!name || create.isPending}
              className="w-full py-2 text-xs bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 disabled:cursor-not-allowed rounded-lg text-white font-medium transition-all duration-150"
            >
              {create.isPending ? "Creating..." : "Create Key"}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
