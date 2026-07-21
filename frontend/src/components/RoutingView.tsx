import { useEffect, useMemo, useState } from "react";
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
  Search,
  X,
  Loader2,
  MousePointerClick,
} from "lucide-react";
import type { AppSettings, RoutingRule } from "../types";
import { ListProcesses } from "../../wailsjs/go/main/App";

interface Props {
  settings: AppSettings;
  disabled: boolean; // locked while connected
  onChange: (s: AppSettings) => void;
}

type RuleType = RoutingRule["type"];
type RuleAction = RoutingRule["action"];

const TYPES: {
  key: RuleType;
  label: string;
  icon: React.ReactNode;
  ph: string;
}[] = [
  { key: "domain", label: "Домены", icon: <Globe2 size={15} />, ph: "example.com" },
  { key: "ip", label: "IP / CIDR", icon: <Network size={15} />, ph: "1.2.3.4 или 10.0.0.0/8" },
  { key: "process", label: "Приложения", icon: <AppWindow size={15} />, ph: "chrome.exe" },
];

const ACTIONS: {
  key: RuleAction;
  label: string;
  icon: React.ReactNode;
  cls: string;
}[] = [
  { key: "proxy", label: "Через VPN", icon: <ArrowRight size={13} />, cls: "text-accent" },
  { key: "direct", label: "Напрямую", icon: <MoveRight size={13} />, cls: "text-ok" },
  { key: "block", label: "Заблокировать", icon: <ShieldOff size={13} />, cls: "text-danger" },
];

// Validate the value against the active rule type so an IP can't land in the
// domain list and vice-versa.
function validate(type: RuleType, raw: string): string | null {
  const v = raw.trim();
  if (!v) return "Введите значение";
  const looksIP = /^[0-9a-fA-F:.]+(\/\d{1,3})?$/.test(v) && /[.:]/.test(v);
  switch (type) {
    case "ip":
      if (!looksIP) return "Это не похоже на IP или CIDR";
      return null;
    case "domain":
      if (looksIP && !/[a-zA-Z]/.test(v.replace(/\/.*$/, "")))
        return "Похоже на IP — используйте вкладку «IP / CIDR»";
      if (!/^[a-zA-Z0-9.*_-]+\.[a-zA-Z]{2,}$/.test(v) && !v.startsWith("*."))
        return "Введите домен, напр. example.com";
      return null;
    case "process":
      if (!/\.exe$/i.test(v)) return "Имя процесса должно оканчиваться на .exe";
      return null;
  }
}

// Routing overrides. Local/LAN traffic (127.0.0.1, 192.168.*, etc.) is always
// direct — handled in the core config — so there is no global mode here. Rules
// are grouped by type; each type has its own list.
export default function RoutingView({ settings, disabled, onChange }: Props) {
  const rules = settings.rules ?? [];
  const [type, setType] = useState<RuleType>("domain");
  const [action, setAction] = useState<RuleAction>("proxy");
  const [value, setValue] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [picking, setPicking] = useState(false);

  const commitRules = (next: RoutingRule[]) => onChange({ ...settings, rules: next });

  const addRule = (t: RuleType, v: string, a: RuleAction) => {
    const val = v.trim();
    if (!val) return;
    // Avoid exact duplicates within the same type.
    if (rules.some((r) => r.type === t && r.value.toLowerCase() === val.toLowerCase()))
      return;
    commitRules([...rules, { type: t, value: val, action: a }]);
  };

  const add = () => {
    const problem = validate(type, value);
    if (problem) {
      setErr(problem);
      return;
    }
    addRule(type, value, action);
    setValue("");
    setErr(null);
  };

  const remove = (rule: RoutingRule) =>
    commitRules(rules.filter((r) => r !== rule));

  const typeMeta = TYPES.find((t) => t.key === type)!;
  const shown = rules.filter((r) => r.type === type);

  return (
    <div className="animate-fade-up flex h-full flex-col p-6">
      <div className="mb-5 flex items-center gap-2.5">
        <Split size={20} className="text-accent" />
        <div>
          <h1 className="text-base font-medium text-text">Маршрутизация</h1>
          <p className="font-mono text-xs text-text-faint">
            локальный трафик всегда идёт напрямую
          </p>
        </div>
      </div>

      {/* Type tabs */}
      <div className="mb-4 grid grid-cols-3 gap-2">
        {TYPES.map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setType(t.key);
              setValue("");
              setErr(null);
            }}
            className={`flex items-center justify-center gap-1.5 rounded-lg border py-2.5 text-sm transition ${
              type === t.key
                ? "border-accent/60 bg-surface-2 text-text"
                : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
            }`}
          >
            {t.icon}
            {t.label}
          </button>
        ))}
      </div>

      {/* Rule builder */}
      <div className="mb-5 flex flex-col gap-3 rounded-lg border border-border bg-surface p-4">
        {type === "process" ? (
          <button
            onClick={() => setPicking(true)}
            disabled={disabled}
            className="flex items-center justify-center gap-2 rounded-lg border border-dashed border-border bg-bg py-3 text-sm text-text-muted transition hover:border-accent/50 hover:bg-surface-2/60 hover:text-text disabled:cursor-not-allowed disabled:opacity-50"
          >
            <MousePointerClick size={16} />
            Выбрать запущенное приложение
          </button>
        ) : null}

        <div className="flex gap-2">
          <input
            value={value}
            disabled={disabled}
            onChange={(e) => {
              setValue(e.target.value);
              setErr(null);
            }}
            onKeyDown={(e) => e.key === "Enter" && add()}
            placeholder={typeMeta.ph}
            className={`min-w-0 flex-1 rounded-lg border bg-bg px-3 py-2.5 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 disabled:opacity-60 ${
              err ? "border-danger/50" : "border-border"
            }`}
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

        {err && <div className="text-xs text-danger">{err}</div>}

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

      {/* Rule list for the active type */}
      {shown.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
          <span className="text-text-faint">{typeMeta.icon}</span>
          <p className="max-w-xs text-sm text-text-muted">
            {type === "domain" && "Правил по доменам пока нет."}
            {type === "ip" && "Правил по IP пока нет."}
            {type === "process" && "Правил по приложениям пока нет."}
          </p>
        </div>
      ) : (
        <div className="-mr-6 flex flex-1 flex-col gap-2 overflow-y-auto pr-1.5">
          {shown.map((r, i) => (
            <RuleRow key={`${r.value}-${i}`} rule={r} onDelete={() => remove(r)} />
          ))}
        </div>
      )}

      {picking && (
        <ProcessPicker
          onClose={() => setPicking(false)}
          onPick={(name) => {
            addRule("process", name, action);
          }}
        />
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

// ProcessPicker lists running processes with a live filter. Clicking a process
// adds it as a rule immediately and closes the picker.
function ProcessPicker({
  onClose,
  onPick,
}: {
  onClose: () => void;
  onPick: (name: string) => void;
}) {
  const [all, setAll] = useState<string[] | null>(null);
  const [q, setQ] = useState("");

  useEffect(() => {
    ListProcesses().then((p) => setAll((p as string[]) ?? []));
  }, []);

  const filtered = useMemo(() => {
    if (!all) return [];
    const s = q.trim().toLowerCase();
    return s ? all.filter((n) => n.toLowerCase().includes(s)) : all;
  }, [all, q]);

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-fade-up flex max-h-[70vh] w-full max-w-md flex-col rounded-xl border border-border bg-surface p-4 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-medium text-text">
            Выберите приложение
          </h2>
          <button
            onClick={onClose}
            className="no-drag text-text-faint transition hover:text-text"
          >
            <X size={16} />
          </button>
        </div>

        <div className="mb-3 flex items-center gap-2 rounded-lg border border-border bg-bg px-3">
          <Search size={14} className="shrink-0 text-text-faint" />
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Поиск процесса…"
            autoFocus
            className="min-w-0 flex-1 bg-transparent py-2.5 font-mono text-sm text-text outline-none placeholder:text-text-faint"
          />
        </div>

        <div className="-mr-2 flex flex-1 flex-col gap-1 overflow-y-auto pr-1.5">
          {all === null ? (
            <div className="flex items-center justify-center gap-2 py-8 text-sm text-text-muted">
              <Loader2 size={15} className="animate-spin" />
              Загрузка…
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-8 text-center text-sm text-text-muted">
              Ничего не найдено.
            </div>
          ) : (
            filtered.map((name) => (
              <button
                key={name}
                onClick={() => onPick(name)}
                className="no-drag flex items-center gap-2.5 rounded-lg px-3 py-2 text-left font-mono text-sm text-text-muted transition hover:bg-surface-2 hover:text-text"
              >
                <AppWindow size={15} className="shrink-0 text-text-faint" />
                <span className="truncate">{name}</span>
                <Plus size={14} className="ml-auto shrink-0 text-text-faint" />
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
