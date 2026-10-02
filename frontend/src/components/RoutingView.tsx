import SegmentedControl from "./SegmentedControl";
import {tunStackInfo} from "../proto";
import ModalPortal from "./ModalPortal";
import RoutingTools from "./RoutingTools";
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
import RouteCanvas, { graphFromRules } from "./RouteCanvas";
import { Workflow, List, SlidersHorizontal, Monitor, RefreshCw } from "lucide-react";
import { ListProcesses } from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";

type ProcessInfo = main.ProcessInfo & {application?:boolean;path?:string};

interface Props {
  settings: AppSettings;
  disabled: boolean; // locked while connected
  onChange: (s: AppSettings) => void;
}

type RuleType = Extract<RoutingRule["type"], "domain" | "ip" | "process">;
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
// Routing has two modes. Easy is a flat list of overrides; Pro is a node
// graph where the user wires matchers to actions and decides priority and the
// catch-all. The core follows whichever mode is selected; the other mode's
// setup is kept, not applied.
export default function RoutingView({ settings, disabled, onChange }: Props) {
  const pro = settings.routingMode === "pro";
  const [tun, setTun] = useState(false);

  const setMode = (mode: "simple" | "pro") => {
    if (disabled || mode === (settings.routingMode || "simple")) return;
    let next: AppSettings = { ...settings, routingMode: mode };
    // The first switch to Pro starts from the simple rules, so nothing is lost.
    if (mode === "pro" && (settings.graph?.nodes ?? []).length === 0 && (settings.rules ?? []).length > 0) {
      next = { ...next, graph: graphFromRules(settings.rules, settings.simpleFinal ?? "proxy") };
    }
    onChange(next);
  };

  const modes: { id: "simple" | "pro"; label: string; icon: React.ReactNode }[] = [
    { id: "simple", label: "Easy", icon: <List size={13} /> },
    { id: "pro", label: "Pro", icon: <Workflow size={13} /> },
  ];

  return (
    <div className="animate-view flex h-full min-h-0 flex-col">
      <div className={`px-6 pt-6 ${pro ? "pb-4" : "pb-5"}`}>
        <div className={`mx-auto flex w-full items-center gap-2.5 ${pro ? "" : "max-w-2xl"}`}>
          <Split size={20} className="text-accent" />
          <div className="min-w-0 flex-1">
            <h1 className="text-base font-medium text-text">Маршрутизация</h1>
            <p className="font-mono text-xs text-text-faint">
              {pro ? "граф правил · локальный трафик всегда напрямую" : "локальный трафик всегда идёт напрямую"}
            </p>
          </div>
          <button
            onClick={() => setTun((o) => !o)}
            title="Параметры туннеля"
            className={`no-drag flex h-8 items-center gap-1.5 rounded-lg border px-2.5 text-xs transition ${
              tun ? "border-accent/50 bg-surface-2 text-text" : "border-border bg-surface text-text-muted hover:text-text"
            }`}
          >
            <SlidersHorizontal size={14} /> TUN
          </button>
          <SegmentedControl label="Режим маршрутизации" value={settings.routingMode||"simple"} options={modes} disabled={disabled} onChange={v=>setMode(v as "simple"|"pro")}/>

        </div>
      </div>

      <div className="px-6 pb-4"><RoutingTools settings={settings} disabled={disabled} onChange={onChange}/></div>
      <div className="relative flex min-h-0 flex-1">
        {pro ? (
          <RouteCanvas settings={settings} disabled={disabled} onChange={onChange} />
        ) : (
          <SimpleRouting settings={settings} disabled={disabled} onChange={onChange} />
        )}
        {tun && <TunPanel settings={settings} disabled={disabled} onChange={onChange} onClose={() => setTun(false)} />}
      </div>
    </div>
  );
}

// TunPanel keeps every tunnel parameter in one place, next to the rules they
// shape. The same values are also on the Settings screen.
function TunPanel({
  settings: s,
  disabled,
  onChange,
  onClose,
}: Props & { onClose: () => void }) {
  const set = (patch: Partial<AppSettings>) => onChange({ ...s, ...patch });
  const input =
    "w-full rounded-lg border border-border bg-bg px-3 py-2 font-mono text-xs text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 disabled:opacity-60";
  return (
    <div className="animate-pop absolute bottom-20 right-3 top-3 z-20 flex w-[300px] flex-col overflow-hidden rounded-[var(--radius-lg)] border border-border bg-surface/95 shadow-2xl backdrop-blur-md">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div>
          <div className="text-sm font-medium text-text">Параметры TUN</div>
          <div className="text-[11px] text-text-faint">Применяются при подключении</div>
        </div>
        <button onClick={onClose} className="no-drag rounded p-1 text-text-faint transition hover:bg-surface-2 hover:text-text">
          <X size={15} />
        </button>
      </div>
      <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
        <TunToggle label="IPv6" hint="Туннелировать IPv6, а не пускать его мимо" on={s.ipv6} disabled={disabled} onChange={(v) => set({ ipv6: v })} />
        <TunToggle label="Строгий маршрут" hint="Не выпускать трафик и DNS в обход туннеля" on={s.strictRoute} disabled={disabled} onChange={(v) => set({ strictRoute: v })} />
        <TunToggle label="Сниффинг" hint="Узнавать домен из TLS, HTTP и QUIC. Нужен доменным правилам и протоколам" on={s.sniff} disabled={disabled} onChange={(v) => set({ sniff: v })} />

        <TunField label="Сетевой стек">
          <div className="rounded-lg border border-border bg-bg px-3 py-2 text-xs text-text-muted" aria-label="Сетевой стек закреплён за ядром">{tunStackInfo(s.core).label}<p className="mt-1 text-[10px] leading-relaxed text-text-faint">{tunStackInfo(s.core).hint}</p></div>
        </TunField>
        <TunField label="Имя адаптера">
          <input className={input} value={s.tunName} disabled={disabled} placeholder="TomorrowTun" spellCheck={false} onChange={(e) => set({ tunName: e.target.value })} />
        </TunField>
        <TunField label="MTU" hint="0 — по умолчанию ядра">
          <input className={input} type="number" value={s.mtu || 0} disabled={disabled} onChange={(e) => set({ mtu: parseInt(e.target.value) || 0 })} />
        </TunField>
        <TunField label="DNS через туннель" hint="IP, tls://, https://, quic://">
          <input className={input} value={s.dns} disabled={disabled} placeholder="1.1.1.1" spellCheck={false} onChange={(e) => set({ dns: e.target.value })} />
        </TunField>
        <TunField label="DNS напрямую" hint="Для прямых доменов и самого сервера">
          <input className={input} value={s.dnsFallback} disabled={disabled} placeholder="8.8.8.8" spellCheck={false} onChange={(e) => set({ dnsFallback: e.target.value })} />
        </TunField>
      </div>
    </div>
  );
}

function TunToggle({
  label,
  hint,
  on,
  disabled,
  onChange,
}: {
  label: string;
  hint: string;
  on: boolean;
  disabled: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onChange(!on)}
      className="no-drag flex items-start gap-3 text-left disabled:cursor-not-allowed disabled:opacity-60"
    >
      <div className="min-w-0 flex-1">
        <div className="text-sm text-text">{label}</div>
        <div className="text-[11px] leading-snug text-text-faint">{hint}</div>
      </div>
      <span className={`relative mt-0.5 h-5 w-9 shrink-0 rounded-full transition ${on ? "bg-accent" : "bg-surface-2"}`}>
        <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-text transition-all ${on ? "left-[18px]" : "left-0.5"}`} />
      </span>
    </button>
  );
}

function TunField({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="flex items-baseline justify-between gap-2">
        <span className="text-xs text-text-muted">{label}</span>
        {hint && <span className="truncate text-[10px] text-text-faint">{hint}</span>}
      </span>
      {children}
    </div>
  );
}

function SimpleRouting({ settings, disabled, onChange }: Props) {
  const rules = settings.rules ?? [];
  const [type, setType] = useState<RuleType>("domain");
  const [action, setAction] = useState<RuleAction>("proxy");
  const [value, setValue] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [picking, setPicking] = useState(false);

  const commitRules = (next: RoutingRule[]) => onChange({ ...settings, rules: next });

  const addRule = (t: RuleType, v: string, a: RuleAction, icon?: string) => {
    const val = v.trim();
    if (!val) return;
    // Avoid exact duplicates within the same type.
    if (rules.some((r) => r.type === t && r.value.toLowerCase() === val.toLowerCase()))
      return;
    commitRules([...rules, { type: t, value: val, action: a, icon }]);
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
  const shown = rules.filter((r) => r.type === type || (type === "domain" && r.type === "geosite") || (type === "ip" && r.type === "geoip"));

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Everything sits in a centred column of one width; only the scroller
          below spans the window, so its bar stays at the edge. */}
      <div className="px-6">
      <div className="mx-auto w-full max-w-2xl">

      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs text-text-faint">Остальной трафик</span>
        <SegmentedControl disabled={disabled} label="Остальной трафик Easy" value={settings.simpleFinal || "proxy"} onChange={(simpleFinal) => !disabled && onChange({...settings,simpleFinal: simpleFinal as RuleAction})} options={[{id:"proxy",label:"Через VPN"},{id:"direct",label:"Напрямую"},{id:"block",label:"Блокировать"}]} />
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
            className="flex shrink-0 items-center gap-1.5 rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-on-accent transition hover:bg-accent-soft disabled:cursor-not-allowed disabled:opacity-50"
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
        <div className="min-h-0 flex-1 overflow-y-auto px-6 pb-6 [scrollbar-gutter:stable_both-edges]">
          <div className="mx-auto flex w-full max-w-2xl flex-col gap-2">
            {shown.map((r, i) => (
              <RuleRow
                key={`${r.value}-${i}`}
                rule={r}
                onDelete={() => remove(r)}
              />
            ))}
          </div>
        </div>
      )}

      {picking && (
        <ProcessPicker
          onClose={() => setPicking(false)}
          onPick={(name, icon) => {
            addRule("process", name, action, icon);
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
  const t = TYPES.find((x) => x.key === (rule.type === "geosite" ? "domain" : rule.type === "geoip" ? "ip" : rule.type));
  const a = ACTIONS.find((x) => x.key === rule.action);
  return (
    <div className="group flex items-center gap-3 rounded-lg border border-border bg-surface px-4 py-3">
      <span className="grid h-8 w-8 shrink-0 place-items-center overflow-hidden rounded-lg bg-surface-2 text-text-muted">
        {rule.type === "process" && rule.icon ? (
          <img
            src={rule.icon}
            alt=""
            className="h-5 w-5 object-contain"
            draggable={false}
          />
        ) : (
          t?.icon
        )}
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate font-mono text-sm text-text">{rule.value}</div>
        <div className="text-xs text-text-faint">{rule.type === "geosite" ? "GeoSite · готовый список сайтов" : rule.type === "geoip" ? "GeoIP · страна по IP" : t?.label}</div>
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

// ProcessPicker lists running processes with a live filter and their real app
// icons. Clicking a process adds it as a rule immediately and closes the picker.
export function ProcessPicker({
  onClose,
  onPick,
}: {
  onClose: () => void;
  onPick: (name: string, icon?: string) => void;
}) {
  const [all, setAll] = useState<ProcessInfo[] | null>(null);
  const [q, setQ] = useState("");
  const [appsOnly,setAppsOnly] = useState(true);

  const load = () => {
    setAll(null);
    ListProcesses().then((p) => setAll((p as ProcessInfo[]) ?? []));
  };
  useEffect(load, []);

  const filtered = useMemo(() => {
    if (!all) return [];
    const s = q.trim().toLowerCase();
    return all.filter(p=>(!appsOnly || p.application) && (!s || p.name.toLowerCase().includes(s) || p.path?.toLowerCase().includes(s)));
  }, [all, q, appsOnly]);

  return (
    <ModalPortal onClose={onClose}>
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-view flex max-h-[70vh] w-full max-w-md flex-col rounded-2xl border border-border bg-surface p-4 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-3 flex items-center gap-2">
          <Monitor size={16} className="text-text-muted" />
          <h2 className="flex-1 text-sm font-medium text-text">Запущенные процессы</h2>
          <button
            onClick={load}
            title="Обновить"
            className="no-drag rounded-md p-1.5 text-text-faint transition hover:bg-surface-2 hover:text-text"
          >
            <RefreshCw size={15} />
          </button>
          <button
            onClick={onClose}
            className="no-drag rounded-md p-1.5 text-text-faint transition hover:bg-surface-2 hover:text-text"
          >
            <X size={16} />
          </button>
        </div>

        <div className="mb-3 flex gap-2 text-xs"><button className="rounded-md border border-border px-3 py-2" onClick={()=>setAppsOnly(true)}>Приложения {appsOnly?"✓":""}</button><button className="rounded-md border border-border px-3 py-2" onClick={()=>setAppsOnly(false)}>Все процессы {!appsOnly?"✓":""}</button></div>
        <div className="mb-3 flex items-center gap-2 rounded-xl border border-border bg-bg px-3 transition focus-within:border-accent/70 focus-within:ring-2 focus-within:ring-accent/20">
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
            filtered.map((p) => (
              <button
                key={p.name}
                onClick={() => onPick(p.name, p.icon || undefined)}
                className="no-drag flex items-center gap-2.5 rounded-lg px-3 py-2 text-left font-mono text-sm text-text-muted transition hover:bg-surface-2 hover:text-text"
              >
                {p.icon ? (
                  <img
                    src={p.icon}
                    alt=""
                    className="h-[18px] w-[18px] shrink-0 object-contain"
                    draggable={false}
                  />
                ) : (
                  <AppWindow size={15} className="shrink-0 text-text-faint" />
                )}
                <span className="truncate">{p.name}</span>
                <Plus size={14} className="ml-auto shrink-0 text-text-faint" />
              </button>
            ))
          )}
        </div>
      </div>
    </div>
    </ModalPortal>
  );
}
