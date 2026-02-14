import { useState, useMemo } from "react";
import { X, Loader2, Check } from "lucide-react";
import { useModels, useDiscoverModels, useImportModels } from "../hooks/useModels.ts";
import { useUpstreams } from "../hooks/useUpstreams.ts";
import type { DiscoveredModel } from "../lib/types.ts";

export function AddModelsDialog({ onClose }: { onClose: () => void }) {
  const [step, setStep] = useState<1 | 2>(1);
  const [selectedUpstreamId, setSelectedUpstreamId] = useState("");
  const [discovered, setDiscovered] = useState<DiscoveredModel[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const upstreams = useUpstreams();
  const discover = useDiscoverModels();
  const importModels = useImportModels();
  const existingModels = useModels();

  const existingNames = useMemo(
    () => new Set((existingModels.data ?? []).map((m) => m.name)),
    [existingModels.data],
  );

  const selectableModels = useMemo(
    () => discovered.filter((m) => !existingNames.has(m.id)),
    [discovered, existingNames],
  );

  const selectedUpstream = useMemo(
    () => (upstreams.data ?? []).find((u) => u.id === selectedUpstreamId),
    [upstreams.data, selectedUpstreamId],
  );

  function handleFetch(e: React.FormEvent) {
    e.preventDefault();
    discover.mutate(
      { upstream_id: selectedUpstreamId },
      {
        onSuccess: (models) => {
          setDiscovered(models);
          const newIds = new Set(
            models.filter((m) => !existingNames.has(m.id)).map((m) => m.id),
          );
          setSelected(newIds);
          setStep(2);
        },
      },
    );
  }

  function handleToggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function handleToggleAll() {
    if (selected.size === selectableModels.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(selectableModels.map((m) => m.id)));
    }
  }

  function handleImport() {
    const models = discovered
      .filter((m) => selected.has(m.id))
      .map((m) => ({ name: m.id, provider: m.owned_by }));

    importModels.mutate(
      {
        upstream_id: selectedUpstreamId,
        models,
      },
      { onSuccess: () => onClose() },
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-[2px]" onClick={onClose} />
      <div
        className="relative bg-zinc-900/95 border border-zinc-800/40 rounded-xl shadow-2xl w-full max-w-lg m-4"
        style={{ animation: "fadeInUp 0.25s ease-out forwards" }}
      >
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800/60">
          <h2 className="text-sm font-semibold text-zinc-100">
            {step === 1
              ? "Add Models"
              : `${selectedUpstream?.name ?? "Upstream"} — Select Models`}
          </h2>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition-colors"
          >
            <X size={15} />
          </button>
        </div>

        {step === 1 && (
          <form onSubmit={handleFetch} className="p-5 space-y-4">
            <div>
              <label className="block text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
                Upstream
              </label>
              <select
                value={selectedUpstreamId}
                onChange={(e) => setSelectedUpstreamId(e.target.value)}
                required
                className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg text-sm text-zinc-200 px-3 py-2 focus:outline-none focus:border-zinc-600 transition-colors"
              >
                <option value="">Select an upstream...</option>
                {(upstreams.data ?? [])
                  .filter((u) => u.is_active)
                  .map((u) => (
                    <option key={u.id} value={u.id}>
                      {u.name} ({u.format})
                    </option>
                  ))}
              </select>
            </div>
            {discover.isError && (
              <p className="text-xs text-red-400">
                {discover.error?.message ?? "Failed to fetch models"}
              </p>
            )}
            <button
              type="submit"
              disabled={!selectedUpstreamId || discover.isPending}
              className="w-full py-2 text-xs bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 disabled:cursor-not-allowed rounded-lg text-white font-medium transition-all duration-150 flex items-center justify-center gap-2"
            >
              {discover.isPending && (
                <Loader2 size={13} className="animate-spin" />
              )}
              {discover.isPending ? "Fetching Models..." : "Fetch Models"}
            </button>
          </form>
        )}

        {step === 2 && (
          <div className="p-5 space-y-4">
            <div className="flex items-center justify-between">
              <p className="text-[10px] text-zinc-500">
                {discovered.length} models found
                {selectableModels.length < discovered.length && (
                  <span>
                    {" "}
                    · {discovered.length - selectableModels.length} already added
                  </span>
                )}
              </p>
              <button
                onClick={handleToggleAll}
                className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium transition-colors"
              >
                {selected.size === selectableModels.length
                  ? "Deselect All"
                  : "Select All"}
              </button>
            </div>

            <div className="max-h-80 overflow-y-auto space-y-0.5 -mx-1 px-1">
              {discovered.map((model) => {
                const isExisting = existingNames.has(model.id);
                const isSelected = selected.has(model.id);

                return (
                  <label
                    key={model.id}
                    className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                      isExisting
                        ? "opacity-40 cursor-not-allowed"
                        : "hover:bg-zinc-800/30 cursor-pointer"
                    }`}
                  >
                    <div
                      className={`w-3.5 h-3.5 rounded border flex items-center justify-center flex-shrink-0 transition-colors ${
                        isExisting
                          ? "border-zinc-700 bg-zinc-700"
                          : isSelected
                            ? "border-emerald-500 bg-emerald-600"
                            : "border-zinc-600 bg-zinc-800"
                      }`}
                    >
                      {(isSelected || isExisting) && (
                        <Check size={9} className="text-white" />
                      )}
                    </div>
                    <input
                      type="checkbox"
                      checked={isSelected || isExisting}
                      disabled={isExisting}
                      onChange={() => !isExisting && handleToggle(model.id)}
                      className="sr-only"
                    />
                    <div className="min-w-0 flex-1">
                      <span className="text-xs text-zinc-200 font-mono block truncate">
                        {model.id}
                      </span>
                      <span className="text-[10px] text-zinc-600">
                        {model.owned_by}
                        {isExisting && " · already added"}
                      </span>
                    </div>
                  </label>
                );
              })}
            </div>

            {importModels.isError && (
              <p className="text-xs text-red-400">
                {importModels.error?.message ?? "Failed to import models"}
              </p>
            )}

            <button
              onClick={handleImport}
              disabled={selected.size === 0 || importModels.isPending}
              className="w-full py-2 text-xs bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 disabled:cursor-not-allowed rounded-lg text-white font-medium transition-all duration-150 flex items-center justify-center gap-2"
            >
              {importModels.isPending && (
                <Loader2 size={13} className="animate-spin" />
              )}
              {importModels.isPending
                ? "Importing..."
                : `Import ${selected.size} Model${selected.size !== 1 ? "s" : ""}`}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
