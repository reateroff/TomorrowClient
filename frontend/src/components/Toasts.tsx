import { Toaster, toast } from "sonner";
import {
  CheckCircle2,
  AlertCircle,
  Info,
  TriangleAlert,
  Loader2,
  X,
} from "lucide-react";

// Notifications are Sonner (sonner.emilkowal.ski): stacked cards that fan out
// on hover, swipe away, and can carry a description and an action. Sonner does
// the layout and motion; the look below is the app's own, drawn from the same
// theme tokens as every card, so toasts follow the user's theme and accent.

export type ToastKind = "info" | "ok" | "error" | "warn";

export interface ToastOptions {
  description?: string;
  action?: { label: string; onClick: () => void };
  duration?: number;
}

// push() is the global imperative API — call it from anywhere in the app.
export function push(message: string, kind: ToastKind = "info", opts: ToastOptions = {}) {
  const o = {
    description: opts.description,
    duration: opts.duration,
    action: opts.action ? { label: opts.action.label, onClick: opts.action.onClick } : undefined,
  };
  switch (kind) {
    case "ok":
      return toast.success(message, o);
    case "error":
      return toast.error(message, { ...o, duration: opts.duration ?? 6000 });
    case "warn":
      return toast.warning(message, o);
    default:
      return toast.info(message, o);
  }
}

// track shows a spinner toast while a task runs and turns it into the result.
export function track<T>(
  task: Promise<T>,
  msgs: { loading: string; success: string | ((v: T) => string); error: string | ((e: unknown) => string) }
): Promise<T> {
  toast.promise(task, msgs);
  return task;
}

export { toast };

const ICONS = {
  success: <CheckCircle2 size={16} className="text-ok" />,
  error: <AlertCircle size={16} className="text-danger" />,
  info: <Info size={16} className="text-accent" />,
  warning: <TriangleAlert size={16} className="text-amber" />,
  loading: <Loader2 size={16} className="animate-spin text-text-muted" />,
  close: <X size={12} />,
};

// lifted raises the stack clear of a screen's own bottom-right control (the
// Pro routing canvas keeps its + button there).
export default function Toasts({ lifted = false }: { lifted?: boolean }) {
  return (
    <Toaster
      position="bottom-right"
      offset={{ right: 16, bottom: lifted ? 80 : 16 }}
      gap={8}
      visibleToasts={4}
      closeButton
      icons={ICONS}
      toastOptions={{
        unstyled: true,
        duration: 4000,
        classNames: {
          toast:
            "group/toast relative flex w-[var(--width)] items-start gap-3 rounded-[var(--radius-lg)] border border-border bg-surface/95 px-4 py-3 shadow-[0_8px_30px_rgb(0_0_0/0.35)] backdrop-blur-md font-sans",
          icon: "mt-px flex shrink-0 items-center",
          content: "flex min-w-0 flex-1 flex-col gap-0.5",
          title: "text-sm leading-snug text-text break-words",
          description: "text-xs leading-relaxed text-text-muted break-words",
          actionButton:
            "no-drag shrink-0 self-center rounded-md bg-accent px-2.5 py-1 text-xs font-medium text-bg transition hover:bg-accent-soft",
          cancelButton:
            "no-drag shrink-0 self-center rounded-md border border-border px-2.5 py-1 text-xs text-text-muted transition hover:bg-surface-2",
          closeButton:
            "no-drag !absolute !left-auto !right-[-6px] !top-[-6px] grid !h-5 !w-5 place-items-center rounded-full border border-border bg-surface-2 text-text-faint opacity-0 transition group-hover/toast:opacity-100 hover:text-text",
          error: "border-danger/40",
          warning: "border-amber/40",
        },
      }}
    />
  );
}
