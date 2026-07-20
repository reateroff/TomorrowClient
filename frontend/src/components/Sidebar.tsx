import { Shield, Server, Settings as Cog } from "lucide-react";
import type { ViewKey } from "../types";

const NAV: { key: ViewKey; label: string; icon: typeof Shield }[] = [
  { key: "connection", label: "Соединение", icon: Shield },
  { key: "profiles", label: "Профили", icon: Server },
  { key: "settings", label: "Настройки", icon: Cog },
];

interface Props {
  active: ViewKey;
  onSelect: (v: ViewKey) => void;
}

// Left navigation rail. The active item gets an accent left-bar.
export default function Sidebar({ active, onSelect }: Props) {
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
