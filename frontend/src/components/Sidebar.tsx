import { Shield, Server, Globe, Split, Settings as Cog } from "lucide-react";
import type { ViewKey } from "../types";

const NAV: { key: ViewKey; label: string; icon: typeof Shield }[] = [
  { key: "connection", label: "Соединение", icon: Shield },
  { key: "profiles", label: "Профили", icon: Server },
  { key: "configs", label: "Конфигурации", icon: Globe },
  { key: "routing", label: "Маршрутизация", icon: Split },
  { key: "settings", label: "Настройки", icon: Cog },
];

interface Props {
  active: ViewKey;
  position: "left" | "top";
  onSelect: (v: ViewKey) => void;
}

// Navigation tabs: a vertical rail on the left or a horizontal bar on top,
// chosen in Settings → Внешний вид.
export default function Sidebar({ active, position, onSelect }: Props) {
  // Top mode is a floating pill bar rather than a full-width strip: one rounded
  // container holding the tabs, with the active one outlined in the accent.
  // Both the container and the tabs use rounded-lg so they follow the corner
  // rounding chosen in Settings → Внешний вид.
  if (position === "top") {
    return (
      <nav className="flex shrink-0 justify-center px-3 py-3">
        <div className="flex items-center gap-1 rounded-lg border border-border bg-surface p-1.5 shadow-lg shadow-black/20">
          {NAV.map(({ key, label, icon: Icon }) => {
            const isActive = active === key;
            return (
              <button
                key={key}
                onClick={() => onSelect(key)}
                className={`flex items-center gap-2 rounded-lg border px-3.5 py-1.5 text-sm transition ${
                  isActive
                    ? "border-accent/50 bg-surface-2 text-text shadow-md shadow-accent/15"
                    : "border-transparent text-text-muted hover:bg-surface-2/60 hover:text-text"
                }`}
              >
                <Icon
                  size={15}
                  className={isActive ? "text-accent" : "text-text-faint"}
                />
                {label}
              </button>
            );
          })}
        </div>
      </nav>
    );
  }

  return (
    <aside className="flex w-52 shrink-0 flex-col border-r border-border bg-surface/40">
      <nav className="flex flex-1 flex-col gap-1 p-3">
        {NAV.map(({ key, label, icon: Icon }) => {
          const isActive = active === key;
          return (
            <button
              key={key}
              onClick={() => onSelect(key)}
              className={`group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm transition ${
                isActive
                  ? "bg-surface-2 text-text"
                  : "text-text-muted hover:bg-surface-2/60 hover:text-text"
              }`}
            >
              {isActive && (
                <span className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-accent" />
              )}
              <Icon size={17} className={isActive ? "text-accent" : ""} />
              {label}
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
