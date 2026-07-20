import { Minus, X, Square } from "lucide-react";
import {
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
} from "../../wailsjs/runtime/runtime";

// Custom title bar for the frameless window. The center area is draggable;
// the window controls on the right are not.
export default function TitleBar() {
  return (
    <div className="drag flex h-9 shrink-0 items-center justify-between border-b border-border bg-surface/60 px-3 backdrop-blur">
      <div className="flex items-center gap-2">
        <div className="h-2.5 w-2.5 rounded-full bg-accent" />
        <span className="font-mono text-xs tracking-wide text-text-muted">
          TomorrowClient
        </span>
      </div>

      <div className="no-drag flex items-center gap-1">
        <button
          onClick={WindowMinimise}
          className="grid h-6 w-8 place-items-center rounded text-text-faint transition hover:bg-surface-2 hover:text-text"
          aria-label="Minimize"
        >
          <Minus size={14} />
        </button>
        <button
          onClick={WindowToggleMaximise}
          className="grid h-6 w-8 place-items-center rounded text-text-faint transition hover:bg-surface-2 hover:text-text"
          aria-label="Maximize"
        >
          <Square size={11} />
        </button>
        <button
          onClick={Quit}
          className="grid h-6 w-8 place-items-center rounded text-text-faint transition hover:bg-danger/20 hover:text-danger"
          aria-label="Close"
        >
          <X size={14} />
        </button>
      </div>
    </div>
  );
}
