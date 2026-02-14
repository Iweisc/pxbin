import { useState, type FormEvent } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useAuth } from "../lib/auth.tsx";

export function LoginPage() {
  const [key, setKey] = useState("");
  const [error, setError] = useState("");
  const { login } = useAuth();
  const navigate = useNavigate();

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = key.trim();
    if (!trimmed) {
      setError("API key is required");
      return;
    }
    login(trimmed);
    navigate({ to: "/" });
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-zinc-950">
      <div
        className="w-full max-w-sm p-6"
        style={{ animation: "fadeInUp 0.4s ease-out forwards" }}
      >
        <h1 className="text-2xl font-bold font-mono text-zinc-100 mb-1 text-center tracking-tight">
          pxbin
        </h1>
        <p className="text-xs text-zinc-500 mb-8 text-center">
          Enter your management API key
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <input
              type="password"
              value={key}
              onChange={(e) => {
                setKey(e.target.value);
                setError("");
              }}
              placeholder="pxm_..."
              autoFocus
              className="w-full px-3 py-2.5 bg-zinc-900/80 border border-zinc-800/60 rounded-lg text-sm text-zinc-100 font-mono placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition-colors"
            />
            {error && (
              <p className="text-red-400 text-xs mt-1.5">{error}</p>
            )}
          </div>
          <button
            type="submit"
            className="w-full py-2.5 bg-emerald-600 hover:bg-emerald-500 rounded-lg text-xs font-medium text-white transition-all duration-150"
          >
            Sign in
          </button>
        </form>
      </div>
    </div>
  );
}
