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
    <aside className="flex flex-col w-52 min-h-screen bg-zinc-950 border-r border-zinc-800/60">
      <div className="px-5 py-4 border-b border-zinc-800/60">
        <span className="text-base font-bold font-mono tracking-tight text-zinc-100">
          pxbin
        </span>
      </div>

      <nav className="flex-1 py-2.5 px-2 space-y-0.5">
        {NAV_ITEMS.map(({ to, label, icon: Icon }) => {
          const isActive = matchRoute({ to, fuzzy: to !== "/" });
          return (
            <Link
              key={to}
              to={to}
              className={cn(
                "flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-[13px] transition-all duration-150",
                isActive
                  ? "bg-zinc-800/70 text-zinc-100"
                  : "text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900/60",
              )}
            >
              <Icon size={15} strokeWidth={isActive ? 2 : 1.5} />
              {label}
            </Link>
          );
        })}
      </nav>

      <div className="px-2 py-2.5 border-t border-zinc-800/60">
        <button
          onClick={logout}
          className="flex items-center gap-2.5 px-3 py-1.5 w-full rounded-lg text-[13px] text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900/60 transition-all duration-150"
        >
          <LogOut size={15} strokeWidth={1.5} />
          Logout
        </button>
      </div>
    </aside>
  );
}
