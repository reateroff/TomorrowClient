// Formatting helpers for subscription traffic and expiry, with an unlimited (∞)
// fallback used when the provider reports nothing.

export const INFINITY = "∞";

// formatBytes renders a byte count as a compact human string (e.g. "1.5 ГБ").
export function formatBytes(bytes: number): string {
  if (!bytes || bytes < 0) return "0 Б";
  const units = ["Б", "КБ", "МБ", "ГБ", "ТБ", "ПБ"];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  const val = n >= 100 || i === 0 ? Math.round(n) : Math.round(n * 10) / 10;
  return `${val} ${units[i]}`;
}

// formatTraffic returns "used / total" where used = upload + download, and the
// total is ∞ when the provider reports no quota.
export function formatTraffic(
  upload: number,
  download: number,
  total: number
): string {
  const used = (upload || 0) + (download || 0);
  const cap = total > 0 ? formatBytes(total) : INFINITY;
  return `${formatBytes(used)} / ${cap}`;
}

// formatExpiry renders the expiry unix-seconds timestamp as a date, or ∞ when
// the subscription never expires.
export function formatExpiry(expire: number): string {
  if (!expire || expire <= 0) return INFINITY;
  const d = new Date(expire * 1000);
  return d.toLocaleDateString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

// subDomain extracts the host from a subscription URL for display.
export function subDomain(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    const s = url.replace(/^[a-z0-9+.-]+:\/\//i, "");
    const cut = s.search(/[/?#]/);
    return cut >= 0 ? s.slice(0, cut) : s;
  }
}
