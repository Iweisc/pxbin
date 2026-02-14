import { useState } from "react";
import { Plus } from "lucide-react";
import { ProtectedRoute } from "../lib/auth.tsx";
import { KeysTable } from "../components/KeysTable.tsx";
import { CreateKeyDialog } from "../components/CreateKeyDialog.tsx";
import { Pagination } from "../components/Pagination.tsx";
import { useLLMKeys, useManagementKeys } from "../hooks/useKeys.ts";

export function KeysPage() {
  const [tab, setTab] = useState<"llm" | "management">("llm");
  const [showCreate, setShowCreate] = useState(false);
  const [llmPage, setLlmPage] = useState(1);
  const [llmPerPage, setLlmPerPage] = useState(20);
  const [mgmtPage, setMgmtPage] = useState(1);
  const [mgmtPerPage, setMgmtPerPage] = useState(20);

  const llmKeys = useLLMKeys(llmPage, llmPerPage);
  const mgmtKeys = useManagementKeys(mgmtPage, mgmtPerPage);

  return (
    <ProtectedRoute>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold text-zinc-100 tracking-tight">API Keys</h1>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-emerald-600 hover:bg-emerald-500 rounded-lg text-white font-medium transition-all duration-150"
          >
            <Plus size={13} />
            Create Key
          </button>
        </div>

        <div className="flex gap-1 bg-zinc-900/80 rounded-lg p-0.5 border border-zinc-800/50 w-fit">
          <button
            onClick={() => setTab("llm")}
            className={`px-3 py-1 text-[11px] font-medium rounded-md transition-colors ${
              tab === "llm"
                ? "bg-zinc-700/80 text-zinc-100"
                : "text-zinc-500 hover:text-zinc-300"
            }`}
          >
            LLM Keys
          </button>
          <button
            onClick={() => setTab("management")}
            className={`px-3 py-1 text-[11px] font-medium rounded-md transition-colors ${
              tab === "management"
                ? "bg-zinc-700/80 text-zinc-100"
                : "text-zinc-500 hover:text-zinc-300"
            }`}
          >
            Management Keys
          </button>
        </div>

        {tab === "llm" ? (
          <>
            <KeysTable
              data={llmKeys.data?.data ?? []}
              isLoading={llmKeys.isLoading}
            />
            {llmKeys.data && (
              <Pagination
                page={llmKeys.data.page}
                perPage={llmKeys.data.per_page}
                total={llmKeys.data.total}
                onPageChange={setLlmPage}
                onPerPageChange={setLlmPerPage}
              />
            )}
          </>
        ) : (
          <>
            <KeysTable
              data={mgmtKeys.data?.data ?? []}
              isLoading={mgmtKeys.isLoading}
            />
            {mgmtKeys.data && (
              <Pagination
                page={mgmtKeys.data.page}
                perPage={mgmtKeys.data.per_page}
                total={mgmtKeys.data.total}
                onPageChange={setMgmtPage}
                onPerPageChange={setMgmtPerPage}
              />
            )}
          </>
        )}

        {showCreate && (
          <CreateKeyDialog
            type={tab}
            onClose={() => setShowCreate(false)}
          />
        )}
      </div>
    </ProtectedRoute>
  );
}
