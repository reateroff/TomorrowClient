import {createPortal} from "react-dom";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { MoreVertical } from "lucide-react";

export interface MenuItem {
  label: string;
  icon?: React.ReactNode;
  onClick: () => void;
  danger?: boolean;
}

interface Props {
  items: MenuItem[];
  label?: string;
}

// Menu is a kebab (⋮) button with a small popup.
//
// The popup is positioned `fixed` from the trigger's measured rect rather than
// `absolute` inside it: these menus sit in scrolling lists whose overflow-y
// would clip an absolutely positioned child, cutting the menu in half for any
// row near the bottom.
export default function Menu({ items, label = "Ещё" }: Props) {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, right: 0 });
  const btnRef = useRef<HTMLButtonElement>(null);
  const popRef = useRef<HTMLDivElement>(null);

  // Measure before paint so the menu never appears at the wrong spot first.
  useLayoutEffect(() => {
    if (!open) return;
    const b = btnRef.current?.getBoundingClientRect();
    if (!b) return;
    const height = Math.ceil(popRef.current?.offsetHeight ?? (items.length * 36 + 2));
    // Flip above the trigger when there is no room below it.
    const below = b.bottom + 6;
    const top =
      below + height > window.innerHeight ? Math.max(6, b.top - height - 6) : below;
    setPos({ top, right: Math.max(6, window.innerWidth - b.right) });
  }, [open, items.length]);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (popRef.current?.contains(t) || btnRef.current?.contains(t)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    // A scroll would leave the fixed popup floating away from its row.
    const onScroll = () => setOpen(false);
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    window.addEventListener("scroll", onScroll, true);
    window.addEventListener("resize", onScroll);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
      window.removeEventListener("scroll", onScroll, true);
      window.removeEventListener("resize", onScroll);
    };
  }, [open]);

  return (
    <>
      <button
        ref={btnRef}
        onClick={() => setOpen((o) => !o)}
        aria-label={label}
        className={`no-drag shrink-0 rounded-md p-1.5 transition hover:bg-surface-2 hover:text-text ${
          open ? "bg-surface-2 text-text" : "text-text-faint"
        }`}
      >
        <MoreVertical size={16} />
      </button>

      {open && createPortal(
        <div
          ref={popRef}
          style={{ top: pos.top, right: pos.right }}
          className="animate-pop fixed z-[100] min-w-48 overflow-hidden rounded-lg border border-border bg-surface-2 shadow-xl"
        >
          {items.map((it) => (
            <button
              key={it.label}
              onClick={() => {
                setOpen(false);
                it.onClick();
              }}
              className={`flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm transition hover:bg-surface ${
                it.danger ? "text-danger" : "text-text-muted hover:text-text"
              }`}
            >
              <span className="shrink-0">{it.icon}</span>
              {it.label}
            </button>
          ))}
        </div>, document.body
      )}
    </>
  );
}
