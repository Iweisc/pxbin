import { Link, useMatchRoute } from "@tanstack/react-router";
import {
  LayoutDashboard,
  ScrollText,
  DollarSign,
  Key,
  Box,
  Server,
  LogOut,
} from "lucide-react";
import { useAuth } from "../lib/auth.tsx";
import { cn } from "../lib/utils.ts";

const NAV_ITEMS = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard },
  { to: "/logs", label: "Logs", icon: ScrollText },
  { to: "/costs", label: "Costs", icon: DollarSign },
  { to: "/keys", label: "API Keys", icon: Key },
  { to: "/models", label: "Models", icon: Box },
  { to: "/upstreams", label: "Upstreams", icon: Server },
] as const;

export function Sidebar() {
  const { logout } = useAuth();
  const matchRoute = useMatchRoute();

  return (
    <aside className="flex flex-col w-56 min-h-screen bg-zinc-950 border-r border-zinc-800">
      <div className="px-5 py-5 border-b border-zinc-800">
        <span className="text-lg font-bold font-mono tracking-tight text-zinc-100">
          pxbin
        </span>
      </div>

      <nav className="flex-1 py-3 px-2 space-y-0.5">
        {NAV_ITEMS.map(({ to, label, icon: Icon }) => {
          const isActive = matchRoute({ to, fuzzy: to !== "/" });
          return (
            <Link
              key={to}
              to={to}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors",
                isActive
                  ? "bg-zinc-800 text-zinc-100"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900",
              )}
            >
              <Icon size={16} />
              {label}
            </Link>
          );
        })}
      </nav>

      <div className="px-2 py-3 border-t border-zinc-800">
        <button
          onClick={logout}
          className="flex items-center gap-3 px-3 py-2 w-full rounded-md text-sm text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900 transition-colors"
        >
          <LogOut size={16} />
          Logout
        </button>
      </div>
    </aside>
  );
}
