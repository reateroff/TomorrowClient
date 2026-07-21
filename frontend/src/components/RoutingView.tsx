import { useState } from "react";
import {
  Split,
  Globe2,
  Network,
  AppWindow,
  Plus,
  Trash2,
  ArrowRight,
  ShieldOff,
  MoveRight,
} from "lucide-react";
import type { AppSettings, RoutingRule } from "../types";

interface Props {
  settings: AppSettings;
  disabled: boolean; // locked while connected
  onChange: (s: AppSettings) => void;
}

type RuleType = RoutingRule["type"];
type RuleAction = RoutingRule["action"];

const TYPES: { key: RuleType; label: string; icon: React.ReactNode; ph: string }[] = [
  { key: "domain", label: "Домен", icon: <Globe2 size={15} />, ph: "example.com" },
  { key: "ip", label: "IP / CIDR", icon: <Network size={15} />, ph: "1.2.3.4 или 10.0.0.0/8" },
  { key: "process", label: "Приложение", icon: <AppWindow size={15} />, ph: "chrome.exe" },
];

const ACTIONS: { key: RuleAction; label: string; icon: React.ReactNode; cls: string }[] = [
  { key: "proxy", label: "Через VPN", icon: <ArrowRight size={13} />, cls: "text-accent" },
  { key: "direct", label: "Напрямую", icon: <MoveRight size={13} />, cls: "text-ok" },
  { key: "block", label: "Заблокировать", icon: <ShieldOff size={13} />, cls: "text-danger" },
];

// Routing overrides: match traffic by domain, IP or application and force it
// through the VPN, direct, or block it entirely.
export default function RoutingView({ settings, disabled, onChange }: Props) {
  const rules = settings.rules ?? [];
  const [type, setType] = useState<RuleType>("domain");
  const [action, setAction] = useState<RuleAction>("proxy");
  const [value, setValue] = useState("");

  const commit = (next: RoutingRule[]) =>
    onChange({ ...settings, rules: next });

  const add = () => {
    const v = value.trim();
    if (!v) return;
    commit([...rules, { type, value: v, action }]);
    setValue("");
  };

  const remove = (i: number) => commit(rules.filter((_, idx) => idx !== i));

  const typeMeta = TYPES.find((t) => t.key === type)!;

  return (
    <div className="animate-fade-up flex h-full flex-col p-6">
      <div className="mb-5 flex items-center gap-2.5">
        <Split size={20} className="text-accent" />
        <div>
          <h1 className="text-base font-medium text-text">Маршрутизация</h1>
          <p className="font-mono text-xs text-text-faint">
            правила для доменов, IP и приложений
          </p>
        </div>
      </div>

      {/* Rule builder */}
      <div className="mb-6 flex flex-col gap-3 rounded-lg border border-border bg-surface p-4">
        <div className="grid grid-cols-3 gap-2">
          {TYPES.map((t) => (
            <button
              key={t.key}
              disabled={disabled}
              onClick={() => setType(t.key)}
              className={`flex items-center justify-center gap-1.5 rounded-lg border py-2 text-xs transition disabled:cursor-not-allowed ${
                type === t.key
                  ? "border-accent/60 bg-surface-2 text-text"
                  : "border-border bg-bg text-text-muted hover:bg-surface-2/60"
              }`}
            >
              {t.icon}
              {t.label}
            </button>
          ))}
        </div>

        <div className="flex gap-2">
          <input
            value={value}
            disabled={disabled}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && add()}
            placeholder={typeMeta.ph}
            className="min-w-0 flex-1 rounded-lg border border-border bg-bg px-3 py-2.5 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 disabled:opacity-60"
          />
          <button
            onClick={add}
            disabled={disabled || !value.trim()}
            className="flex shrink-0 items-center gap-1.5 rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Plus size={16} />
            Добавить
          </button>
        </div>

        <div className="grid grid-cols-3 gap-2">
          {ACTIONS.map((a) => (
            <button
              key={a.key}
              disabled={disabled}
              onClick={() => setAction(a.key)}
              className={`flex items-center justify-center gap-1.5 rounded-lg border py-2 text-xs transition disabled:cursor-not-allowed ${
                action === a.key
                  ? "border-accent/60 bg-surface-2 text-text"
                  : "border-border bg-bg text-text-muted hover:bg-surface-2/60"
              }`}
            >
              <span className={a.cls}>{a.icon}</span>
              {a.label}
            </button>
          ))}
        </div>
      </div>

      {/* Rule list */}
      {rules.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
          <Split size={34} className="text-text-faint" />
          <p className="max-w-xs text-sm text-text-muted">
            Правил пока нет. Добавьте домен, IP или приложение и выберите, как
            обрабатывать его трафик.
          </p>
        </div>
      ) : (
        <div className="flex flex-1 flex-col gap-2 overflow-y-auto pr-1">
          {rules.map((r, i) => (
            <RuleRow key={i} rule={r} onDelete={() => remove(i)} />
          ))}
        </div>
      )}
    </div>
  );
}

function RuleRow({
  rule,
  onDelete,
}: {
  rule: RoutingRule;
  onDelete: () => void;
}) {
  const t = TYPES.find((x) => x.key === rule.type);
  const a = ACTIONS.find((x) => x.key === rule.action);
  return (
    <div className="group flex items-center gap-3 rounded-lg border border-border bg-surface px-4 py-3">
      <span className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-surface-2 text-text-muted">
        {t?.icon}
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate font-mono text-sm text-text">{rule.value}</div>
        <div className="text-xs text-text-faint">{t?.label}</div>
      </div>
      <span
        className={`flex items-center gap-1 rounded-md border border-border bg-bg px-2 py-1 text-[11px] ${
          a?.cls ?? "text-text-muted"
        }`}
      >
        {a?.icon}
        {a?.label}
      </span>
      <button
        onClick={onDelete}
        className="no-drag rounded-md p-1.5 text-text-faint opacity-0 transition hover:bg-danger/15 hover:text-danger group-hover:opacity-100"
        aria-label="Удалить правило"
      >
        <Trash2 size={15} />
      </button>
    </div>
  );
}
