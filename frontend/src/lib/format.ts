// Small formatting helpers shared across views.

/** Formats a byte count into a human readable string (e.g. "1.4 MB"). */
export function formatBytes(bytes: number): string {
  if (!bytes || bytes < 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const val = bytes / Math.pow(1024, i);
  return `${val.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

/** Formats a byte/second rate (e.g. "2.3 MB/s"). */
export function formatSpeed(bytesPerSec: number): string {
  return `${formatBytes(bytesPerSec)}/s`;
}

/** Formats an elapsed duration from a unix-ms start time to "HH:MM:SS". */
export function formatUptime(connectedAt: number, now: number): string {
  if (!connectedAt) return "00:00:00";
  const secs = Math.max(0, Math.floor((now - connectedAt) / 1000));
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  const s = secs % 60;
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${pad(h)}:${pad(m)}:${pad(s)}`;
}
