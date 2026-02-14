# pxbin Frontend Design System Rules

## Overview

pxbin is an LLM API proxy dashboard built with React 19, TypeScript, Tailwind CSS v4, TanStack Router, TanStack React Query, Recharts, and Lucide icons. The UI is a dark-theme, desktop-first admin dashboard.

## Tech Stack

- **Framework**: React 19 (`react@^19.2.0`)
- **Language**: TypeScript 5.9
- **Styling**: Tailwind CSS v4 via `@tailwindcss/vite` plugin (no `tailwind.config` file -- uses CSS `@theme` directive)
- **Routing**: TanStack Router v1 (`@tanstack/react-router`)
- **Data Fetching**: TanStack React Query v5 (`@tanstack/react-query`)
- **Charts**: Recharts v3 (`recharts@^3.7.0`)
- **Icons**: Lucide React (`lucide-react@^0.564.0`)
- **Bundler**: Vite 7 (`vite@^7.3.1`)

## Project Structure

```
frontend/
  src/
    lib/            # Core utilities, API client, auth context, types
      api.ts        # apiFetch<T>() client, ApiError, fetch helpers
      auth.tsx      # AuthProvider, useAuth(), ProtectedRoute
      types.ts      # All TypeScript interfaces (API responses, models, etc.)
      utils.ts      # formatCost, formatTokens, formatDuration, formatDate, cn()
    hooks/          # React Query hooks (one file per resource)
      useStats.ts   # useOverviewStats, useStatsByKey, useStatsByModel, useTimeSeries, useLatencyStats
      useLogs.ts    # useLogs(filters), useLogDetail(id)
      useKeys.ts    # useLLMKeys, useManagementKeys, useCreateKey, useRevokeKey
      useModels.ts  # useModels, useCreateModel, useUpdateModel, useDeleteModel
      useUpstreams.ts # useUpstreams, useCreateUpstream, useUpdateUpstream, useDeleteUpstream
    components/     # Reusable UI components
    routes/         # Page components (one per route)
    router.tsx      # Route definitions, createRouter
    main.tsx        # App entry point
    index.css       # Tailwind import + @theme overrides
```

## Token Definitions / Design Tokens

All tokens come from Tailwind CSS v4 defaults. Custom tokens are defined in `src/index.css`:

```css
@import "tailwindcss";

@theme {
  --font-mono: "JetBrains Mono", ui-monospace, SFMono-Regular, "SF Mono",
    Menlo, Consolas, "Liberation Mono", monospace;
}
```

### Color Palette (Tailwind zinc scale, dark theme)

| Role              | Class                | Hex       |
|-------------------|----------------------|-----------|
| Page background   | `bg-zinc-950`        | `#09090b` |
| Card/table bg     | `bg-zinc-900`        | `#18181b` |
| Borders           | `border-zinc-800`    | `#27272a` |
| Hover bg          | `bg-zinc-800`        | `#27272a` |
| Button default bg | `bg-zinc-800`        | `#27272a` |
| Button hover      | `hover:bg-zinc-700`  | `#3f3f46` |
| Primary text      | `text-zinc-100`      | `#f4f4f5` |
| Secondary text    | `text-zinc-400`      | `#a1a1aa` |
| Muted text        | `text-zinc-500`      | `#71717a` |
| Success/active    | `text-emerald-400`   | `#34d399` |
| Primary button    | `bg-emerald-600`     | `#059669` |
| Primary hover     | `hover:bg-emerald-500` | `#10b981` |
| Error             | `text-red-400`       | `#f87171` |
| Warning           | `text-amber-400`     | `#fbbf24` |

### Chart Colors

| Purpose          | Hex       | Usage                |
|------------------|-----------|----------------------|
| Cost / Success   | `#22c55e` | Area fills, bar fills |
| Latency p50      | `#3b82f6` | Blue line            |
| Latency p95      | `#f59e0b` | Amber line           |
| Latency p99      | `#ef4444` | Red line             |
| Grid lines       | `#27272a` | CartesianGrid stroke |
| Axis labels      | `#52525b` | XAxis/YAxis stroke   |
| Tooltip bg       | `#18181b` | Tooltip contentStyle |
| Tooltip border   | `#3f3f46` | Tooltip contentStyle |

## Typography

- **Body font**: `system-ui, -apple-system, sans-serif` (set on `body` in CSS)
- **Mono font**: `font-mono` class maps to JetBrains Mono stack (set via `@theme`)
- **Headings**: `text-xl font-semibold text-zinc-100` for page titles
- **Section titles**: `text-sm font-medium text-zinc-300`
- **Labels**: `text-xs text-zinc-400` or `text-xs text-zinc-500 uppercase tracking-wider`
- **Data values**: `font-mono text-xs` for numbers, keys, IDs, timestamps
- **Large numbers**: `text-2xl font-mono font-semibold` (stat cards)

## Spacing & Layout

- **Page layout**: `flex min-h-screen` with sidebar + `flex-1 p-6 overflow-auto` main
- **Sidebar width**: `w-56`
- **Page content**: `space-y-4` or `space-y-6` vertical rhythm
- **Cards**: `p-4` padding, `rounded-lg` corners, `border border-zinc-800`
- **Grid gaps**: `gap-3` standard
- **Responsive grid**: `grid-cols-2 lg:grid-cols-3 xl:grid-cols-6` for stat cards

## Component Patterns

### Card

```tsx
<div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
  {/* content */}
</div>
```

### Loading Skeleton

```tsx
<div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 animate-pulse">
  <div className="h-4 w-24 bg-zinc-800 rounded mb-3" />
  <div className="h-[240px] bg-zinc-800/50 rounded" />
</div>
```

### Empty State

```tsx
<div className="bg-zinc-900 border border-zinc-800 rounded-lg flex items-center justify-center py-16">
  <p className="text-zinc-500 text-sm">No data found</p>
</div>
```

### Period Selector (tab toggle)

```tsx
<div className="flex gap-1 bg-zinc-900 rounded-lg p-0.5 border border-zinc-800">
  <button className={active ? "bg-zinc-700 text-zinc-100" : "text-zinc-400 hover:text-zinc-200"}>
    {label}
  </button>
</div>
```

### Primary Button

```tsx
<button className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-emerald-600 hover:bg-emerald-500 rounded-md text-white font-medium transition-colors">
  <Plus size={14} />
  Create
</button>
```

### Default Button

```tsx
<button className="px-4 py-1.5 text-sm bg-zinc-800 hover:bg-zinc-700 rounded-md text-zinc-200 transition-colors">
  Action
</button>
```

### Form Input

```tsx
<input className="w-full bg-zinc-800 border border-zinc-700 rounded-md text-sm text-zinc-200 px-3 py-2 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-600" />
```

### Select

```tsx
<select className="bg-zinc-800 border border-zinc-700 rounded-md text-xs text-zinc-300 px-2 py-1.5">
```

### Modal Overlay

```tsx
<div className="fixed inset-0 z-50 flex items-center justify-center">
  <div className="absolute inset-0 bg-black/60" onClick={onClose} />
  <div className="relative bg-zinc-900 border border-zinc-800 rounded-lg shadow-xl w-full max-w-md m-4">
    {/* header with border-b border-zinc-800, px-5 py-4 */}
    {/* content with p-5 space-y-4 */}
  </div>
</div>
```

### Status Badge

```tsx
// Active
<span className="text-xs font-medium text-emerald-400">Active</span>
// Inactive
<span className="text-xs font-medium text-zinc-500">Inactive</span>
// Error
<span className="text-xs font-medium text-red-400">Error</span>
```

## Icon System

Icons are imported individually from `lucide-react`. Standard size is `size={16}` in nav, `size={14}` in buttons/actions.

```tsx
import { Plus, X, Copy, Trash2, Pencil, Check, AlertTriangle } from "lucide-react";

// In navigation
<Icon size={16} />

// In buttons / action areas
<Plus size={14} />
```

Common icons used:
- `LayoutDashboard` - Dashboard
- `ScrollText` - Logs
- `DollarSign` - Costs
- `Key` - API Keys
- `Box` - Models
- `Server` - Upstreams
- `LogOut` - Logout
- `Plus` - Create actions
- `X` - Close/cancel
- `Copy` - Copy to clipboard
- `Trash2` - Delete/revoke
- `Pencil` - Edit
- `Check` - Confirm/save
- `AlertTriangle` - Warning

## Data Fetching Patterns

### Query Hook

```tsx
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../lib/api.ts";

export function useResource() {
  return useQuery({
    queryKey: ["resource-name"],
    queryFn: () => apiFetch<ResourceType>("/endpoint"),
    staleTime: 30_000,
  });
}
```

### Mutation Hook

```tsx
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useCreateResource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateRequest) =>
      apiFetch<Resource>("/endpoint", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["resource-name"] });
    },
  });
}
```

## Route Page Pattern

Every route page follows this structure:

```tsx
import { ProtectedRoute } from "../lib/auth.tsx";

export function ResourcePage() {
  return (
    <ProtectedRoute>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-zinc-100">Page Title</h1>
          {/* action buttons on right */}
        </div>
        {/* filters, tabs */}
        {/* data table or chart */}
        {/* pagination */}
        {/* modals (conditionally rendered) */}
      </div>
    </ProtectedRoute>
  );
}
```

## Chart Pattern (Recharts)

```tsx
<div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
  <h3 className="text-sm font-medium text-zinc-300 mb-3">Chart Title</h3>
  <ResponsiveContainer width="100%" height={240}>
    <AreaChart data={data}>
      <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
      <XAxis dataKey="timestamp" tickFormatter={formatDate} stroke="#52525b" fontSize={11} />
      <YAxis tickFormatter={formatter} stroke="#52525b" fontSize={11} />
      <Tooltip contentStyle={{ background: "#18181b", border: "1px solid #3f3f46", borderRadius: 8, fontSize: 12 }} />
      <Area type="monotone" dataKey="value" stroke="#22c55e" fill="url(#gradient)" strokeWidth={1.5} />
    </AreaChart>
  </ResponsiveContainer>
</div>
```

## Import Conventions

- Always use `.tsx` or `.ts` extensions in imports
- Types use `import type { ... }` syntax
- Named exports only (no default exports)
- Group imports: react, third-party, local lib, local components, local hooks

## Utility Function: cn()

```tsx
export function cn(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(" ");
}
```

## Responsive Strategy

- Desktop-first layout
- Sidebar always visible (no mobile collapse)
- Responsive grids with `grid-cols-*` breakpoints (`lg:`, `xl:`)
- Tables use `overflow-x-auto` for horizontal scroll on narrow screens
