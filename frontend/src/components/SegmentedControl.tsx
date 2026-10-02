import type { ReactNode } from "react";
export default function SegmentedControl({
  value,
  options,
  onChange,
  disabled = false,
  label,
  className = "",
}: {
  value: string;
  options: { id: string; label: string; icon?: ReactNode }[];
  onChange: (id: string) => void;
  disabled?: boolean;
  label?: string;
  className?: string;
}) {
  return (
    <div
      role="tablist"
      aria-label={label}
      className={`client-segmented no-drag inline-flex shrink-0 items-center gap-0.5 border border-border bg-surface p-0.5 ${className}`}
    >
      {options.map((o) => (
        <button
          key={o.id}
          type="button"
          role="tab"
          aria-selected={value === o.id}
          disabled={disabled}
          onClick={() => onChange(o.id)}
          className={`flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs transition disabled:opacity-50 ${value === o.id ? "bg-surface-2 text-text" : "text-text-muted hover:text-text"}`}
        >
          {o.icon}
          {o.label}
        </button>
      ))}
    </div>
  );
}
