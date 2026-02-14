import { Outlet, useMatch } from "@tanstack/react-router";
import { Sidebar } from "../components/Sidebar.tsx";
import { ErrorBoundary } from "../components/ErrorBoundary.tsx";
import { useAuth } from "../lib/auth.tsx";

export function RootLayout() {
  const { isAuthenticated } = useAuth();
  const loginMatch = useMatch({ from: "/login", shouldThrow: false });
  const isLoginPage = !!loginMatch;

  if (isLoginPage || !isAuthenticated) {
    return (
      <ErrorBoundary>
        <Outlet />
      </ErrorBoundary>
    );
  }

  return (
    <div className="flex min-h-screen bg-zinc-950 text-zinc-100">
      <Sidebar />
      <main className="flex-1 p-6 overflow-auto">
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </main>
    </div>
  );
}
