import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown } from "lucide-react";
export interface SelectOption {
  id: string;
  label: string;
  hint?: string;
}
export default function ClientSelect({
  value,
  options,
  onChange,
  disabled = false,
  label,
  className = "",
  placeholder,
  icon,
  align = "right",
}: {
  value: string;
  options: SelectOption[];
  onChange: (v: string) => void;
  disabled?: boolean;
  label?: string;
  className?: string;
  placeholder?: string;
  icon?: ReactNode;
  align?: "left" | "right";
}) {
  const [open, setOpen] = useState(false),
    [active, setActive] = useState(0),
    [pos, setPos] = useState({ left: 0, top: 0, width: 224, maxHeight: 320 });
  const trigger = useRef<HTMLButtonElement>(null),
    popup = useRef<HTMLDivElement>(null);
  const id = useId();
  const current = options.find((o) => o.id === value);
  const shown =
    current?.label ?? (value || placeholder || options[0]?.label || "Выбрать");
  const choose = (i: number) => {
    const o = options[i];
    if (o) {
      onChange(o.id);
      setOpen(false);
      trigger.current?.focus();
    }
  };
  const toggle = () => {
    if (!open)
      setActive(
        Math.max(
          0,
          options.findIndex((o) => o.id === value),
        ),
      );
    setOpen((o) => !o);
  };
  useLayoutEffect(() => {
    if (!open || !trigger.current) return;
    const r = trigger.current.getBoundingClientRect();
    const width = Math.min(
      window.innerWidth - 16,
      Math.max(r.width, options.some((o) => o.hint) ? 264 : 190),
    );
    const list=popup.current;
    if(list){list.style.width=`${width}px`;list.style.maxHeight='none'}
    const natural=Math.ceil(list?.scrollHeight??options.length*34+10)+2;
    const below=Math.max(48,window.innerHeight-r.bottom-14),above=Math.max(48,r.top-14);
    const upward=natural>below && above>below;
    const height=Math.min(natural,upward?above:below,window.innerHeight-16);
    setPos({left:Math.max(8,Math.min(align==='left'?r.left:r.right-width,window.innerWidth-width-8)),top:upward?Math.max(8,r.top-height-6):r.bottom+6,width,maxHeight:height});
  }, [open, options.length, align]);
  useEffect(() => {
    if (!open) return;
    const list = popup.current;
    const item = list?.querySelector<HTMLElement>(`[data-index="${active}"]`);
    if (!list || !item) return;
    // Scroll only the menu, never the editor/page behind the portal.
    if (item.offsetTop < list.scrollTop) list.scrollTop = item.offsetTop;
    else if (item.offsetTop + item.offsetHeight > list.scrollTop + list.clientHeight)
      list.scrollTop = item.offsetTop + item.offsetHeight - list.clientHeight;
  }, [active, open]);
  useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    const down = (e: MouseEvent) => {
      if (
        !trigger.current?.contains(e.target as Node) &&
        !popup.current?.contains(e.target as Node)
      )
        close();
    };
    const scroll = (e: Event) => {
      if (!popup.current?.contains(e.target as Node)) close();
    };
    document.addEventListener("mousedown", down);
    window.addEventListener("resize", close);
    window.addEventListener("scroll", scroll, true);
    window.addEventListener("tc:close-popups", close);
    return () => {
      document.removeEventListener("mousedown", down);
      window.removeEventListener("resize", close);
      window.removeEventListener("scroll", scroll, true);
      window.removeEventListener("tc:close-popups", close);
    };
  }, [open]);
  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);
  return (
    <>
      <button
        ref={trigger}
        type="button"
        disabled={disabled}
        aria-label={label}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? id : undefined}
        aria-activedescendant={open ? `${id}-${active}` : undefined}
        onClick={toggle}
        onKeyDown={(e) => {
          if (e.key === "Escape" && open) {
            e.preventDefault();
            e.stopPropagation();
            setOpen(false);
          } else if (e.key === "ArrowDown" || e.key === "ArrowUp") {
            e.preventDefault();
            if (!open) {
              setOpen(true);
              setActive(
                Math.max(
                  0,
                  options.findIndex((o) => o.id === value),
                ),
              );
            } else
              setActive(
                (a) =>
                  (a + (e.key === "ArrowDown" ? 1 : -1) + options.length) %
                  options.length,
              );
          } else if (open && (e.key === "Enter" || e.key === " ")) {
            e.preventDefault();
            choose(active);
          } else if (open && e.key === "Tab") setOpen(false);
        }}
        className={`no-drag inline-flex min-w-0 items-center justify-between gap-2 rounded-lg border px-3 py-2 text-xs text-text-muted transition hover:bg-surface-2 hover:text-text disabled:opacity-50 ${open ? "border-accent/50 bg-surface-2" : "border-border bg-surface"} ${className}`}
      >
        {icon}
        <span className="min-w-0 flex-1 truncate text-left">{shown}</span>
        <ChevronDown
          size={13}
          className={`shrink-0 text-text-faint transition ${open ? "rotate-180" : ""}`}
        />
      </button>
      {open &&
        createPortal(
          <div
            id={id}
            ref={popup}
            role="listbox"
            aria-label={label || "Выбор значения"}
            data-tc-select
            className="tc-select-list animate-pop fixed z-[110] overflow-x-hidden overflow-y-auto rounded-lg border border-border bg-surface-2 p-1 shadow-xl"
            style={pos}
          >
            {options.map((o, i) => (
              <button
                key={o.id}
                id={`${id}-${i}`}
                data-index={i}
                type="button"
                role="option"
                aria-selected={o.id === value}
                tabIndex={-1}
                onMouseEnter={() => setActive(i)}
                onClick={() => choose(i)}
                className={`transition-colors duration-150 flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-xs ${i === active ? "bg-surface text-text" : "text-text-muted"}`}
              >
                <span className="min-w-0 flex-1">
                  <span className="block">{o.label}</span>
                  {o.hint && (
                    <span className="mt-0.5 block text-[11px] leading-snug text-text-faint">
                      {o.hint}
                    </span>
                  )}
                </span>
                {o.id === value && (
                  <Check size={13} className="shrink-0 text-accent" />
                )}
              </button>
            ))}
          </div>,
          document.body,
        )}
    </>
  );
}
