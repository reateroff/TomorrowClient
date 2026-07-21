import { useEffect, useRef, useState } from "react";
import {
  Palette,
  AppWindow,
  Network,
  ScrollText,
  Info,
  Cpu,
  Route,
  Globe2,
  Rocket,
  Zap,
  Check,
  Copy,
  Trash2,
} from "lucide-react";
import type { AppInfo, AppSettings, Core, RoutingMode } from "../types";
import { THEME_PRESETS, ACCENTS, FONTS, RADII } from "../theme";
import { GetLogs, ClearLogs } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

type TabKey = "appearance" | "application" | "connection" | "logs" | "about";

const TABS: { key: TabKey; label: string; icon: React.ReactNode }[] = [
  { key: "appearance", label: "Внешний вид", icon: <Palette size={16} /> },
  { key: "application", label: "Приложение", icon: <AppWindow size={16} /> },
  { key: "connection", label: "Соединение", icon: <Network size={16} /> },
  { key: "logs", label: "Логи", icon: <ScrollText size={16} /> },
  { key: "about", label: "О приложении", icon: <Info size={16} /> },
];

interface Props {
  settings: AppSettings;
  appInfo: AppInfo | null;
  disabled: boolean; // connection-affecting fields are locked while connected
  onChange: (s: AppSettings) => void;
}

// The settings screen is a vertical tab layout: a left rail of sections and a
// scrollable panel on the right.
export default function SettingsView({
  settings,
  appInfo,
  disabled,
  onChange,
}: Props) {
  const [tab, setTab] = useState<TabKey>("appearance");
  const set = (patch: Partial<AppSettings>) =>
    onChange({ ...settings, ...patch });

  return (
    <div className="animate-fade-up flex h-full min-h-0">
      {/* Vertical tab rail */}
      <nav className="flex w-52 shrink-0 flex-col gap-1 border-r border-border p-3">
        <div className="px-2 pb-2 pt-1">
          <h1 className="text-sm font-medium text-text">Настройки</h1>
        </div>
        {TABS.map((t) => {
          const active = tab === t.key;
          return (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`relative flex items-center gap-2.5 rounded-lg px-3 py-2 text-left text-sm transition ${
                active
                  ? "bg-surface-2 text-text"
                  : "text-text-muted hover:bg-surface/60 hover:text-text"
              }`}
            >
              {active && (
                <span className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-accent" />
              )}
              <span className={active ? "text-accent" : ""}>{t.icon}</span>
              {t.label}
            </button>
          );
        })}
      </nav>

      {/* Panel */}
      <div className="min-w-0 flex-1 overflow-y-auto p-6">
        {tab === "appearance" && (
          <Appearance settings={settings} set={set} />
        )}
        {tab === "application" && (
          <Application settings={settings} set={set} />
        )}
        {tab === "connection" && (
          <Connection settings={settings} set={set} disabled={disabled} />
        )}
        {tab === "logs" && <Logs />}
        {tab === "about" && <About appInfo={appInfo} />}
      </div>
    </div>
  );
}

type SetFn = (patch: Partial<AppSettings>) => void;

/* --------------------------------- Appearance -------------------------------- */

function Appearance({ settings, set }: { settings: AppSettings; set: SetFn }) {
  return (
    <div className="flex max-w-2xl flex-col gap-7">
      <PanelHead
        title="Внешний вид"
        subtitle="тема, акцент, шрифт и скругление"
      />

      <Section icon={<Palette size={16} />} title="Тема">
        <div className="grid grid-cols-3 gap-3">
          {THEME_PRESETS.map((p) => (
            <button
              key={p.id}
              onClick={() => set({ theme: p.id })}
              className={`flex flex-col gap-2 rounded-lg border p-3 text-left transition ${
                settings.theme === p.id
                  ? "border-accent/60 bg-surface-2"
                  : "border-border bg-surface hover:bg-surface-2/60"
              }`}
            >
              <div className="flex gap-1">
                {[p.colors.bg, p.colors.surface2, p.colors.border].map(
                  (c, i) => (
                    <span
                      key={i}
                      className="h-6 flex-1 rounded"
                      style={{ background: c }}
                    />
                  )
                )}
              </div>
              <span className="text-xs text-text">{p.name}</span>
            </button>
          ))}
        </div>
      </Section>

      <Section icon={<Zap size={16} />} title="Акцент">
        <div className="flex flex-wrap gap-2.5">
          {ACCENTS.map((a) => (
            <button
              key={a.id}
              onClick={() => set({ accent: a.id })}
              title={a.name}
              className={`grid h-9 w-9 place-items-center rounded-full border-2 transition ${
                settings.accent === a.id
                  ? "border-text"
                  : "border-transparent hover:border-border"
              }`}
            >
              <span
                className="grid h-6 w-6 place-items-center rounded-full"
                style={{ background: a.color }}
              >
                {settings.accent === a.id && (
                  <Check size={13} className="text-bg" strokeWidth={3} />
                )}
              </span>
            </button>
          ))}
        </div>
      </Section>

      <Section icon={<AppWindow size={16} />} title="Шрифт">
        <div className="grid grid-cols-3 gap-3">
          {FONTS.map((f) => (
            <button
              key={f.id}
              onClick={() => set({ font: f.id })}
              style={{ fontFamily: f.stack }}
              className={`rounded-lg border px-3 py-3 text-center text-sm transition ${
                settings.font === f.id
                  ? "border-accent/60 bg-surface-2 text-text"
                  : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
              }`}
            >
              {f.name}
            </button>
          ))}
        </div>
      </Section>

      <Section icon={<Route size={16} />} title="Скругление углов">
        <div className="grid grid-cols-3 gap-3">
          {RADII.map((r) => (
            <button
              key={r.id}
              onClick={() => set({ radius: r.id })}
              className={`flex flex-col items-center gap-2 border px-3 py-3 text-xs transition ${
                settings.radius === r.id
                  ? "border-accent/60 bg-surface-2 text-text"
                  : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
              }`}
              style={{ borderRadius: r.value }}
            >
              <span
                className="h-7 w-12 border border-text-faint"
                style={{ borderRadius: r.value }}
              />
              {r.name}
            </button>
          ))}
        </div>
      </Section>
    </div>
  );
}

/* -------------------------------- Application -------------------------------- */

function Application({ settings, set }: { settings: AppSettings; set: SetFn }) {
  return (
    <div className="flex max-w-2xl flex-col gap-7">
      <PanelHead title="Приложение" subtitle="поведение при запуске" />
      <Section icon={<Rocket size={16} />} title="Запуск">
        <div className="flex flex-col gap-3">
          <Switch
            label="Запускать вместе с Windows"
            desc="Задача в планировщике с правами администратора"
            checked={settings.launchAtStartup}
            onChange={(v) => set({ launchAtStartup: v })}
          />
          <Switch
            label="Подключаться к последнему профилю"
            desc="Автоматически поднимать туннель при старте"
            checked={settings.autoConnect}
            onChange={(v) => set({ autoConnect: v })}
          />
        </div>
      </Section>
    </div>
  );
}

/* --------------------------------- Connection -------------------------------- */

function Connection({
  settings,
  set,
  disabled,
}: {
  settings: AppSettings;
  set: SetFn;
  disabled: boolean;
}) {
  return (
    <div className="flex max-w-2xl flex-col gap-7">
      <PanelHead
        title="Соединение"
        subtitle={
          disabled ? "заблокировано во время соединения" : "ядро и параметры туннеля"
        }
      />

      <Section icon={<Cpu size={16} />} title="Ядро">
        <div className="grid grid-cols-2 gap-3">
          <Card
            active={settings.core === "sing-box"}
            disabled={disabled}
            title="sing-box"
            desc="Нативный TUN + auto_route. Рекомендуется."
            onClick={() => set({ core: "sing-box" as Core })}
          />
          <Card
            active={settings.core === "xray"}
            disabled={disabled}
            title="xray"
            desc="Xray-core + tun2socks через WinTun."
            onClick={() => set({ core: "xray" as Core })}
          />
        </div>
      </Section>

      <Section icon={<Route size={16} />} title="Маршрутизация">
        <div className="grid grid-cols-2 gap-3">
          <Card
            active={settings.routingMode === "rules"}
            disabled={disabled}
            title="По правилам"
            desc="Локальная сеть напрямую"
            onClick={() => set({ routingMode: "rules" as RoutingMode })}
          />
          <Card
            active={settings.routingMode === "global"}
            disabled={disabled}
            title="Глобально"
            desc="Весь трафик через VPN"
            onClick={() => set({ routingMode: "global" as RoutingMode })}
          />
        </div>
      </Section>

      <Section icon={<Network size={16} />} title="TUN интерфейс">
        <Field label="Имя адаптера">
          <input
            value={settings.tunName}
            disabled={disabled}
            onChange={(e) => set({ tunName: e.target.value })}
            placeholder="TomorrowTun"
            className={inputCls}
          />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Сетевой стек (sing-box)">
            <div className="grid grid-cols-2 gap-2">
              {["gvisor", "system"].map((s) => (
                <button
                  key={s}
                  disabled={disabled}
                  onClick={() => set({ stack: s })}
                  className={`rounded-lg border py-2 font-mono text-xs transition disabled:cursor-not-allowed ${
                    (settings.stack || "gvisor") === s
                      ? "border-accent/60 bg-surface-2 text-text"
                      : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>
          </Field>
          <Field label="MTU (0 — по умолчанию)">
            <input
              type="number"
              value={settings.mtu || 0}
              disabled={disabled}
              onChange={(e) => set({ mtu: parseInt(e.target.value) || 0 })}
              placeholder="0"
              className={inputCls}
            />
          </Field>
        </div>
      </Section>

      <Section icon={<Globe2 size={16} />} title="DNS">
        <input
          value={settings.dns}
          disabled={disabled}
          onChange={(e) => set({ dns: e.target.value })}
          placeholder="1.1.1.1"
          className={inputCls}
        />
      </Section>
    </div>
  );
}

/* ----------------------------------- Logs ------------------------------------ */

function Logs() {
  const [lines, setLines] = useState<string[]>([]);
  const boxRef = useRef<HTMLDivElement>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    GetLogs().then((l) => setLines((l as string[]) ?? []));
    const off = EventsOn("vpn:log", (line: string) =>
      setLines((prev) => [...prev.slice(-499), line])
    );
    return () => off();
  }, []);

  // Auto-scroll to the newest line.
  useEffect(() => {
    const el = boxRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lines]);

  const copyAll = async () => {
    await navigator.clipboard.writeText(lines.join("\n"));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  const clear = async () => {
    await ClearLogs();
    setLines([]);
  };

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <div className="flex items-center justify-between">
        <PanelHead title="Логи" subtitle="вывод активного ядра" />
        <div className="flex gap-2">
          <button
            onClick={copyAll}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-1.5 text-xs text-text-muted transition hover:bg-surface-2 hover:text-text"
          >
            {copied ? <Check size={13} /> : <Copy size={13} />}
            {copied ? "Скопировано" : "Копировать"}
          </button>
          <button
            onClick={clear}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-1.5 text-xs text-text-muted transition hover:bg-surface-2 hover:text-danger"
          >
            <Trash2 size={13} />
            Очистить
          </button>
        </div>
      </div>
      <div
        ref={boxRef}
        className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-border bg-bg p-3 font-mono text-xs leading-relaxed"
      >
        {lines.length === 0 ? (
          <span className="text-text-faint">
            Логов пока нет. Подключитесь, чтобы увидеть вывод ядра.
          </span>
        ) : (
          lines.map((l, i) => (
            <div key={i} className="whitespace-pre-wrap break-all text-text-muted">
              {l}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

/* ----------------------------------- About ----------------------------------- */

function About({ appInfo }: { appInfo: AppInfo | null }) {
  return (
    <div className="flex max-w-2xl flex-col gap-7">
      <PanelHead title="О приложении" subtitle="версия и лицензия" />
      <div className="overflow-hidden rounded-lg border border-border bg-surface">
        <div className="flex items-center gap-4 border-b border-border p-5">
          <div className="grid h-14 w-14 place-items-center rounded-lg bg-accent/15 font-mono text-lg font-semibold text-accent">
            TC
          </div>
          <div>
            <div className="text-base font-medium text-text">TomorrowClient</div>
            <div className="font-mono text-xs text-text-faint">
              версия {appInfo?.version ?? "—"}
            </div>
          </div>
        </div>
        <div className="flex flex-col gap-3 p-5">
          <p className="text-sm leading-relaxed text-text-muted">
            Минималистичный VPN-клиент для Windows на WinTun с поддержкой двух
            ядер — sing-box и xray-core.
          </p>
          <div className="flex flex-wrap gap-2">
            {(appInfo?.builtWith ?? "Wails · Go · React · sing-box · xray-core")
              .split("·")
              .map((t) => (
                <span
                  key={t}
                  className="rounded-md border border-border bg-bg px-2 py-1 font-mono text-[11px] text-text-faint"
                >
                  {t.trim()}
                </span>
              ))}
          </div>
          <div className="border-t border-border pt-3 font-mono text-[11px] text-text-faint">
            {appInfo?.copyright ?? "© TomorrowClient"}
          </div>
        </div>
      </div>
    </div>
  );
}

/* --------------------------------- Primitives -------------------------------- */

const inputCls =
  "w-full rounded-lg border border-border bg-bg px-3 py-2.5 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 disabled:opacity-60";

function PanelHead({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div>
      <h2 className="text-base font-medium text-text">{title}</h2>
      <p className="font-mono text-xs text-text-faint">{subtitle}</p>
    </div>
  );
}

function Section({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-2.5">
      <div className="flex items-center gap-2 font-mono text-xs uppercase tracking-wide text-text-muted">
        <span className="text-accent">{icon}</span>
        {title}
      </div>
      {children}
    </section>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="text-xs text-text-faint">{label}</span>
      {children}
    </label>
  );
}

function Card({
  active,
  disabled,
  title,
  desc,
  onClick,
}: {
  active: boolean;
  disabled: boolean;
  title: string;
  desc: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={`flex flex-col gap-1 rounded-lg border px-4 py-3 text-left transition disabled:cursor-not-allowed ${
        active
          ? "border-accent/60 bg-surface-2"
          : "border-border bg-surface hover:bg-surface-2/60"
      }`}
    >
      <span className="text-sm text-text">{title}</span>
      <span className="text-xs text-text-faint">{desc}</span>
    </button>
  );
}

function Switch({
  label,
  desc,
  checked,
  onChange,
}: {
  label: string;
  desc: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-center justify-between rounded-lg border border-border bg-surface px-4 py-3">
      <div className="flex flex-col">
        <span className="text-sm text-text">{label}</span>
        <span className="text-xs text-text-faint">{desc}</span>
      </div>
      <button
        type="button"
        onClick={() => onChange(!checked)}
        className={`relative h-5 w-9 shrink-0 rounded-full transition ${
          checked ? "bg-accent" : "bg-border"
        }`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-bg transition-all ${
            checked ? "left-[18px]" : "left-0.5"
          }`}
        />
      </button>
    </label>
  );
}
