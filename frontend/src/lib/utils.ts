export function formatCost(cents: number): string {
  if (cents == null || isNaN(cents)) {
    return "$0.00";
  }
  if (cents >= 100) {
    return `$${(cents / 100).toFixed(2)}`;
  }
  return `${cents.toFixed(2)}c`;
}

export function formatTokens(n: number): string {
  if (n == null || isNaN(n)) {
    return "0";
  }
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1)}K`;
  }
  return n.toString();
}

export function formatDuration(ms: number): string {
  if (ms == null || isNaN(ms)) {
    return "0ms";
  }
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  return `${Math.round(ms)}ms`;
}

export function formatMicroseconds(us: number): string {
  if (us == null || isNaN(us)) {
    return "0us";
  }
  if (us >= 1_000_000) {
    return `${(us / 1_000_000).toFixed(1)}s`;
  }
  if (us >= 1000) {
    return `${(us / 1000).toFixed(1)}ms`;
  }
  return `${Math.round(us)}us`;
}

export function formatDate(iso: string): string {
  if (!iso) {
    return "";
  }
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function cn(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(" ");
}
