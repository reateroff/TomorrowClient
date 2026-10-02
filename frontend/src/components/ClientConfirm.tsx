import { useEffect, useRef, useState } from "react";
import { Info, X } from "lucide-react";
import ModalPortal from "./ModalPortal";
type Request = {
  title: string;
  description: string;
  confirmLabel?: string;
  danger?: boolean;
  resolve: (result: boolean) => void;
};
export function confirmAction(
  options: Omit<Request, "resolve">,
): Promise<boolean> {
  return new Promise((resolve) =>
    window.dispatchEvent(
      new CustomEvent("tc:confirm", { detail: { ...options, resolve } }),
    ),
  );
}
export function ConfirmationHost() {
  const [request, setRequest] = useState<Request | null>(null);
  const current = useRef<Request | null>(null);
  current.current = request;
  const close = (result = false) => {
    current.current?.resolve(result);
    current.current = null;
    setRequest(null);
  };
  useEffect(() => {
    const receive = (e: Event) => {
      current.current?.resolve(false);
      const next = (e as CustomEvent<Request>).detail;
      current.current = next;
      setRequest(next);
    };
    window.addEventListener("tc:confirm", receive);
    return () => {
      window.removeEventListener("tc:confirm", receive);
      current.current?.resolve(false);
    };
  }, []);
  if (!request) return null;
  return (
    <ModalPortal label={request.title} onClose={() => close()}>
      <div
        className="fixed inset-0 z-[120] grid place-items-center bg-black/55 p-6 backdrop-blur-sm"
        onClick={() => close()}
      >
        <div
          className="animate-view w-full max-w-sm rounded-xl border border-border bg-surface p-5 shadow-2xl"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center gap-3">
            <span className="grid h-8 w-8 shrink-0 place-items-center rounded-md bg-accent/10 text-accent">
              <Info size={16} />
            </span>
            <h2 className="min-w-0 flex-1 text-sm font-medium text-text">
              {request.title}
            </h2>
            <button
              type="button"
              aria-label="Закрыть подтверждение"
              onClick={() => close()}
              className="text-text-faint hover:text-text"
            >
              <X size={16} />
            </button>
          </div>
          <p className="my-4 text-xs leading-relaxed text-text-muted">
            {request.description}
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => close()}
              className="rounded-lg border border-border px-4 py-2 text-xs text-text-muted hover:bg-surface-2"
            >
              Отмена
            </button>
            <button
              onClick={() => close(true)}
              className={`rounded-lg px-4 py-2 text-xs font-medium ${request.danger ? "bg-danger text-white" : "bg-accent text-on-accent"} hover:opacity-90`}
            >
              {request.confirmLabel || "Применить"}
            </button>
          </div>
        </div>
      </div>
    </ModalPortal>
  );
}
