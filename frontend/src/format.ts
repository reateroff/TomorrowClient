// The one formatting module. There used to be a second one at lib/format.ts
// exporting a rival formatBytes with English units, so the main screen said
// "0 B" while the profiles list said "0 Б" — same app, same value, two
// spellings. Everything lives here now.

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

// formatSpeed renders a byte/second rate (e.g. "2,3 МБ/с").
export function formatSpeed(bytesPerSec: number): string {
  return `${formatBytes(bytesPerSec)}/с`;
}

// formatUptime renders the time since a unix-ms start, growing a field at a
// time: "07" for the first minute, then "1:07", then "1:01:07". A fixed
// HH:MM:SS would spend the first minute showing two zeroed-out fields.
export function formatUptime(connectedAt: number, now: number): string {
  const secs = connectedAt
    ? Math.max(0, Math.floor((now - connectedAt) / 1000))
    : 0;
  const pad = (n: number) => n.toString().padStart(2, "0");

  const s = secs % 60;
  const m = Math.floor(secs / 60) % 60;
  const h = Math.floor(secs / 3600);

  if (h > 0) return `${h}:${pad(m)}:${pad(s)}`;
  if (m > 0) return `${m}:${pad(s)}`;
  return pad(s);
}

// plural picks the Russian form for n: 1 сервер / 2 сервера / 5 серверов.
export function plural(
  n: number,
  one: string,
  few: string,
  many: string
): string {
  const m10 = n % 10;
  const m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
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
