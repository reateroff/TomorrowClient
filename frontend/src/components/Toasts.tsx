import { useEffect, useState, useCallback } from "react";
import { CheckCircle2, AlertCircle, Info, X } from "lucide-react";

export type ToastKind = "info" | "ok" | "error";

export interface Toast {
  id: number;
  message: string;
  kind: ToastKind;
}

let _next = 1;
let _push: ((t: Toast) => void) | null = null;

// push() is the global imperative API — call it from anywhere in the app.
export function push(message: string, kind: ToastKind = "info") {
  _push?.({ id: _next++, message, kind });
}

const ICON = {
  ok: <CheckCircle2 size={15} className="shrink-0 text-ok" />,
  error: <AlertCircle size={15} className="shrink-0 text-danger" />,
  info: <Info size={15} className="shrink-0 text-accent" />,
};

const TTL = 4000;

export default function Toasts() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const remove = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // Register the global push handler.
  useEffect(() => {
    _push = (t) => setToasts((prev) => [...prev.slice(-4), t]);
    return () => { _push = null; };
  }, []);

  // Auto-dismiss after TTL.
  useEffect(() => {
    if (toasts.length === 0) return;
    const timer = setTimeout(() => remove(toasts[0].id), TTL);
    return () => clearTimeout(timer);
  }, [toasts, remove]);

  if (toasts.length === 0) return null;

  return (
    <div className="pointer-events-none absolute bottom-4 left-4 z-50 flex max-w-xs flex-col gap-2">
      {toasts.map((t) => (
        <div
          key={t.id}
          className="animate-fade-up flex w-fit max-w-full items-center gap-2.5 rounded-lg border border-border bg-surface py-2.5 pl-3.5 pr-2 shadow-lg"
        >
          {ICON[t.kind]}
          <span className="truncate text-sm text-text">{t.message}</span>
          <button
            onClick={() => remove(t.id)}
            className="no-drag pointer-events-auto ml-1 shrink-0 rounded p-1 text-text-faint transition hover:bg-surface-2 hover:text-text"
            aria-label="Закрыть"
          >
            <X size={13} />
          </button>
        </div>
      ))}
    </div>
  );
}
