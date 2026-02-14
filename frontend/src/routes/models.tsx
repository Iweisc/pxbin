import { useState } from "react";
import { Plus, RefreshCw } from "lucide-react";
import { ProtectedRoute } from "../lib/auth.tsx";
import { ModelsTable } from "../components/ModelsTable.tsx";
import { AddModelsDialog } from "../components/AddModelsDialog.tsx";
import { useModels, useSyncPricing } from "../hooks/useModels.ts";
import { useUpstreams } from "../hooks/useUpstreams.ts";

export function ModelsPage() {
  const [showCreate, setShowCreate] = useState(false);
  const models = useModels();
  const upstreams = useUpstreams();
  const syncPricing = useSyncPricing();

  return (
    <ProtectedRoute>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-zinc-100 tracking-tight">Models</h1>
          <div className="flex gap-2">
            <button
              onClick={() => syncPricing.mutate()}
              disabled={syncPricing.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-zinc-800/80 hover:bg-zinc-700/80 border border-zinc-700/50 disabled:opacity-40 disabled:cursor-not-allowed rounded-lg text-zinc-300 font-medium transition-all duration-150"
            >
              <RefreshCw size={13} className={syncPricing.isPending ? "animate-spin" : ""} />
              Sync Pricing
            </button>
            <button
              onClick={() => setShowCreate(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-emerald-600 hover:bg-emerald-500 rounded-lg text-white font-medium transition-all duration-150"
            >
              <Plus size={13} />
              Add Models
            </button>
          </div>
        </div>

        <ModelsTable
          data={models.data ?? []}
          isLoading={models.isLoading}
          upstreams={upstreams.data ?? []}
        />

        {showCreate && (
          <AddModelsDialog onClose={() => setShowCreate(false)} />
        )}
      </div>
    </ProtectedRoute>
  );
}
