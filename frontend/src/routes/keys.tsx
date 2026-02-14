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
          <h1 className="text-xl font-semibold text-zinc-100">API Keys</h1>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-emerald-600 hover:bg-emerald-500 rounded-md text-white font-medium transition-colors"
          >
            <Plus size={14} />
            Create Key
          </button>
        </div>

        <div className="flex gap-1 bg-zinc-900 rounded-lg p-0.5 border border-zinc-800 w-fit">
          <button
            onClick={() => setTab("llm")}
            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
              tab === "llm"
                ? "bg-zinc-700 text-zinc-100"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            LLM Keys
          </button>
          <button
            onClick={() => setTab("management")}
            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
              tab === "management"
                ? "bg-zinc-700 text-zinc-100"
                : "text-zinc-400 hover:text-zinc-200"
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
