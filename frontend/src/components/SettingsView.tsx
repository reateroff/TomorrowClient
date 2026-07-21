import { useEffect, useRef, useState } from "react";
import {
  Palette,
  Monitor,
  Network,
  ScrollText,
  Bug,
  Info,
  Cpu,
  Route,
  Globe2,
  Rocket,
  Zap,
  Check,
  Copy,
  Trash2,
  ChevronRight,
  ArrowLeft,
  Settings2,
  FileJson,
  Lock,
  PanelLeft,
  PanelTop,
  X,
} from "lucide-react";
import type { AppInfo, AppSettings, Core } from "../types";
import { THEME_PRESETS, ACCENTS, FONTS, RADII, isHex } from "../theme";
import {
  GetLogs,
  ClearLogs,
  PreviewConfig,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

type TabKey =
  | "appearance"
  | "application"
  | "connection"
  | "logs"
  | "developer"
  | "about";

interface MenuItem {
  key: TabKey;
  label: string;
  subtitle: string;
  icon: React.ReactNode;
  dev?: boolean;
}

const MENU: MenuItem[] = [
  {
    key: "appearance",
    label: "Внешний вид",
    subtitle: "Тема, шрифт, скругление",
    icon: <Palette size={18} />,
  },
  {
    key: "application",
    label: "Приложение",
    subtitle: "Автозапуск и поведение",
    icon: <Monitor size={18} />,
  },
  {
    key: "connection",
    label: "Соединение",
    subtitle: "Ядро, TUN, DNS, MTU",
    icon: <Network size={18} />,
  },
  {
    key: "logs",
    label: "Логи",
    subtitle: "Журнал работы ядра",
    icon: <ScrollText size={18} />,
  },
  {
    key: "developer",
    label: "Для разработчиков",
    subtitle: "Просмотр конфига и отладка",
    icon: <Bug size={18} />,
    dev: true,
  },
  {
    key: "about",
    label: "О приложении",
    subtitle: "Версия и информация",
    icon: <Info size={18} />,
  },
];

interface Props {
  settings: AppSettings;
  appInfo: AppInfo | null;
  disabled: boolean; // connection-affecting fields are locked while connected
  onChange: (s: AppSettings) => void;
}

// The settings screen is a drill-down menu: a list of section rows that open a
// dedicated page with a back button. This mirrors a mobile-style settings flow.
export default function SettingsView({
  settings,
  appInfo,
  disabled,
  onChange,
}: Props) {
  const [tab, setTab] = useState<TabKey | null>(null);
  const set = (patch: Partial<AppSettings>) =>
    onChange({ ...settings, ...patch });

  const active = MENU.find((m) => m.key === tab);

  // --- Menu (root) ---
  if (!tab || !active) {
    return (
      <div className="animate-fade-up flex h-full flex-col overflow-y-auto p-6">
        <div className="mb-5 flex items-center gap-2.5">
          <Settings2 size={20} className="text-accent" />
          <h1 className="text-lg font-semibold text-text">Настройки</h1>
        </div>
        <div className="flex flex-col gap-2.5">
          {MENU.map((m) => (
            <button
              key={m.key}
              onClick={() => setTab(m.key)}
              className="group flex items-center gap-4 rounded-lg border border-border bg-surface px-4 py-3.5 text-left transition hover:border-border hover:bg-surface-2"
            >
              <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-surface-2 text-text-muted transition group-hover:text-accent">
                {m.icon}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-text">
                    {m.label}
                  </span>
                  {m.dev && (
                    <span className="rounded bg-accent/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-accent">
                      DEV
                    </span>
                  )}
                </div>
                <div className="text-xs text-text-faint">{m.subtitle}</div>
              </div>
              <ChevronRight
                size={18}
                className="shrink-0 text-text-faint transition group-hover:text-text-muted"
              />
            </button>
          ))}
        </div>
      </div>
    );
  }

  // --- Section page ---
  return (
    <div className="animate-fade-up flex h-full min-h-0 flex-col p-6">
      <div className="mb-6 flex items-center gap-3">
        <button
          onClick={() => setTab(null)}
          className="grid h-8 w-8 place-items-center rounded-lg text-text-muted transition hover:bg-surface-2 hover:text-text"
        >
          <ArrowLeft size={18} />
        </button>
        <span className="text-accent">{active.icon}</span>
        <h1 className="text-lg font-semibold text-text">{active.label}</h1>
        {active.dev && (
          <span className="rounded bg-accent/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-accent">
            DEV
          </span>
        )}
      </div>

      <div
        className={`min-h-0 flex-1 ${
          tab === "logs" ? "flex flex-col" : "overflow-y-auto"
        }`}
      >
        {tab === "appearance" && <Appearance settings={settings} set={set} />}
        {tab === "application" && <Application settings={settings} set={set} />}
        {tab === "connection" && (
          <Connection settings={settings} set={set} disabled={disabled} />
        )}
        {tab === "logs" && <Logs />}
        {tab === "developer" && <Developer settings={settings} />}
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
        <CustomAccent
          value={settings.accent}
          onApply={(hex) => set({ accent: hex })}
        />
      </Section>

      <Section icon={<Monitor size={16} />} title="Шрифт">
        <div className="grid grid-cols-2 gap-3">
          {FONTS.map((f) => (
            <button
              key={f.id}
              onClick={() => set({ font: f.id })}
              className={`flex flex-col items-start gap-1 rounded-lg border px-4 py-3 text-left transition ${
                settings.font === f.id
                  ? "border-accent/60 bg-surface-2"
                  : "border-border bg-surface hover:bg-surface-2/60"
              }`}
            >
              <span
                style={{ fontFamily: f.stack }}
                className={`text-lg leading-tight ${
                  settings.font === f.id ? "text-text" : "text-text-muted"
                }`}
              >
                Соединение
              </span>
              <span className="font-mono text-[11px] text-text-faint">
                {f.name}
              </span>
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

      <Section icon={<PanelLeft size={16} />} title="Расположение вкладок">
        <div className="grid grid-cols-2 gap-3">
          <NavPosCard
            active={(settings.navPosition || "left") === "left"}
            icon={<PanelLeft size={18} />}
            title="Слева"
            onClick={() => set({ navPosition: "left" })}
          />
          <NavPosCard
            active={settings.navPosition === "top"}
            icon={<PanelTop size={18} />}
            title="Сверху"
            onClick={() => set({ navPosition: "top" })}
          />
        </div>
      </Section>
    </div>
  );
}

function NavPosCard({
  active,
  icon,
  title,
  onClick,
}: {
  active: boolean;
  icon: React.ReactNode;
  title: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-3 rounded-lg border px-4 py-3 text-left transition ${
        active
          ? "border-accent/60 bg-surface-2 text-text"
          : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
      }`}
    >
      <span className={active ? "text-accent" : "text-text-faint"}>{icon}</span>
      <span className="text-sm">{title}</span>
    </button>
  );
}

// CustomAccent opens a polished modal to pick a custom hex accent: a large live
// preview, a native colour wheel, a hex field and a strip of quick shades.
function CustomAccent({
  value,
  onApply,
}: {
  value: string;
  onApply: (hex: string) => void;
}) {
  const custom = isHex(value);
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className={`mt-3 flex items-center gap-2.5 rounded-lg border px-3 py-2 text-sm transition ${
          custom
            ? "border-accent/60 bg-surface-2 text-text"
            : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
        }`}
      >
        <span
          className="h-5 w-5 rounded-full border border-border"
          style={{ background: custom ? value : "conic-gradient(from 0deg,#f26d6d,#e0b155,#5bd6a0,#38bdf8,#b18cff,#f26d6d)" }}
        />
        {custom ? `Свой цвет · ${value}` : "Выбрать свой цвет"}
      </button>
      {open && (
        <ColorPickerModal
          initial={custom ? value : "#7c8cff"}
          onClose={() => setOpen(false)}
          onApply={(hex) => {
            onApply(hex);
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

const QUICK_SHADES = [
  "#7c8cff", "#4f8cff", "#38bdf8", "#4fd1c5", "#5bd6a0", "#a3e635",
  "#e0b155", "#fb923c", "#f26d6d", "#f08a9c", "#e879f9", "#b18cff",
  "#64748b", "#5a5f6b", "#3a7a55", "#8f4550",
];

function ColorPickerModal({
  initial,
  onClose,
  onApply,
}: {
  initial: string;
  onClose: () => void;
  onApply: (hex: string) => void;
}) {
  const [hex, setHex] = useState(initial);
  const valid = isHex(hex);
  const preview = valid ? hex : "#2a2a2f";

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-fade-up w-full max-w-sm rounded-xl border border-border bg-surface p-5 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-medium text-text">Свой цвет акцента</h2>
          <button
            onClick={onClose}
            className="no-drag text-text-faint transition hover:text-text"
          >
            <X size={16} />
          </button>
        </div>

        {/* Live preview */}
        <div
          className="mb-4 flex h-20 items-center justify-center rounded-lg border border-border"
          style={{ background: preview }}
        >
          <span
            className="rounded-md px-3 py-1 font-mono text-sm"
            style={{
              background: "rgba(0,0,0,0.35)",
              color: "#fff",
            }}
          >
            {valid ? hex.toUpperCase() : "—"}
          </span>
        </div>

        {/* Wheel + hex field */}
        <div className="mb-4 flex items-center gap-3">
          <label
            className="grid h-11 w-11 shrink-0 cursor-pointer place-items-center rounded-lg border border-border"
            style={{ background: preview }}
            title="Палитра"
          >
            <input
              type="color"
              value={valid ? hex : "#7c8cff"}
              onChange={(e) => setHex(e.target.value)}
              className="h-0 w-0 opacity-0"
            />
            <Palette size={16} className="text-white/80 mix-blend-difference" />
          </label>
          <input
            value={hex}
            onChange={(e) => setHex(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && valid && onApply(hex)}
            placeholder="#ffffff"
            spellCheck={false}
            className={`min-w-0 flex-1 rounded-lg border bg-bg px-3 py-2.5 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 ${
              hex.length > 1 && !valid ? "border-danger/50" : "border-border"
            }`}
          />
        </div>

        {/* Quick shades */}
        <div className="mb-5 grid grid-cols-8 gap-2">
          {QUICK_SHADES.map((c) => (
            <button
              key={c}
              onClick={() => setHex(c)}
              title={c}
              className={`h-7 w-full rounded-md border transition ${
                hex.toLowerCase() === c ? "border-text" : "border-transparent hover:border-border"
              }`}
              style={{ background: c }}
            />
          ))}
        </div>

        <div className="flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-2 text-sm text-text-muted transition hover:bg-surface-2"
          >
            Отмена
          </button>
          <button
            onClick={() => onApply(hex)}
            disabled={!valid}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft disabled:opacity-50"
          >
            Применить
          </button>
        </div>
      </div>
    </div>
  );
}

/* -------------------------------- Application -------------------------------- */

function Application({ settings, set }: { settings: AppSettings; set: SetFn }) {
  return (
    <div className="flex max-w-2xl flex-col gap-7">
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
      <Section icon={<Monitor size={16} />} title="Окно">
        <div className="flex flex-col gap-3">
          <Switch
            label="Сворачивать в трей"
            desc="Прятать окно в область уведомлений вместо панели задач"
            checked={settings.minimizeToTray}
            onChange={(v) => set({ minimizeToTray: v })}
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
      {disabled && (
        <div className="flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2.5 text-xs text-text-muted">
          <Lock size={14} className="text-text-faint" />
          Параметры соединения заблокированы, пока туннель активен.
        </div>
      )}

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
    <div className="flex h-full min-h-0 flex-col gap-3">
      <div className="flex items-center justify-end gap-2">
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
            <div
              key={i}
              className="whitespace-pre-wrap break-all text-text-muted"
            >
              {l}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

/* --------------------------------- Developer --------------------------------- */

function Developer({ settings }: { settings: AppSettings }) {
  const [config, setConfig] = useState<string>("");
  const [err, setErr] = useState<string>("");
  const [copied, setCopied] = useState(false);

  const load = async () => {
    setErr("");
    try {
      const c = await PreviewConfig();
      setConfig(c as string);
    } catch (e: any) {
      setConfig("");
      setErr(String(e?.message ?? e));
    }
  };

  const copy = async () => {
    await navigator.clipboard.writeText(config);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="flex max-w-2xl flex-col gap-7">
      <Section icon={<FileJson size={16} />} title="Конфигурация ядра">
        <p className="text-xs leading-relaxed text-text-muted">
          Сгенерированный JSON, который передаётся активному ядру (
          <span className="font-mono text-text">{settings.core}</span>) для
          выбранного профиля.
        </p>
        <div className="flex gap-2">
          <button
            onClick={load}
            className="rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft"
          >
            Показать конфиг
          </button>
          {config && (
            <button
              onClick={copy}
              className="flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text-muted transition hover:bg-surface-2 hover:text-text"
            >
              {copied ? <Check size={14} /> : <Copy size={14} />}
              {copied ? "Скопировано" : "Копировать"}
            </button>
          )}
        </div>
        {err && (
          <div className="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-xs text-danger">
            {err}
          </div>
        )}
        {config && (
          <pre className="max-h-96 overflow-auto rounded-lg border border-border bg-bg p-3 font-mono text-[11px] leading-relaxed text-text-muted">
            {config}
          </pre>
        )}
      </Section>
    </div>
  );
}

/* ----------------------------------- About ----------------------------------- */

function About({ appInfo }: { appInfo: AppInfo | null }) {
  return (
    <div className="flex max-w-2xl flex-col gap-7">
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
