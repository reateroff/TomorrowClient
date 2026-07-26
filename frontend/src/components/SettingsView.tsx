import { useEffect, useRef, useState } from "react";
import {
  Palette,
  Monitor,
  Network,
  ScrollText,
  Bug,
  Info,
  Route,
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
  RefreshCw,
  RotateCcw,
  Download,
  Terminal,
  AlertTriangle,
  PlayCircle,
  Server,
  Trash,
  Square,
  Sparkles,
  BugOff,
  Pipette,
  ChevronDown,
  Plus,
  EyeOff,
} from "lucide-react";
import type { AppInfo, AppSettings } from "../types";
import {
  THEME_PRESETS,
  ACCENTS,
  FONTS,
  RADII,
  ANIMATIONS,
  THEMES_VISIBLE,
  isHex,
  hexToHsv,
  hsvToHex,
} from "../theme";
import type { HSV } from "../theme";
import {
  GetLogs,
  ClearLogs,
  PreviewConfig,
  GetSettingsJSON,
  RunNetDiag,
  ResetSettings,
  ExportDiagnostics,
  GetSettings,
  GetActiveProfileJSON,
  ResetAllData,
  SimulateStatus,
  StopSimulation,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { plural } from "../format";
import { push } from "./Toasts";
import logo from "../assets/logo.png";

// How many taps on the client name unlock the developer section.
const TAPS_TO_UNLOCK = 10;

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

  // Dev-only sections stay out of the menu until unlocked. Deriving the active
  // page from the visible list means switching dev mode off while standing on
  // the developer page drops back to the menu instead of stranding the user.
  const menu = MENU.filter((m) => !m.dev || settings.devMode);
  const active = menu.find((m) => m.key === tab);

  // --- Menu (root) ---
  if (!tab || !active) {
    return (
      <div
        key="menu"
        className="animate-view h-full overflow-y-auto p-6 [scrollbar-gutter:stable_both-edges]"
      >
        <div className="mx-auto flex w-full max-w-2xl flex-col">
        <div className="mb-5 flex items-center gap-2.5">
          <Settings2 size={20} className="text-accent" />
          <h1 className="text-lg font-semibold text-text">Настройки</h1>
        </div>
        <div className="flex flex-col gap-2.5">
          {menu.map((m) => (
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
      </div>
    );
  }

  // --- Section page ---
  // Both roots carry a key: a CSS entrance animation only replays on a fresh
  // mount, and without distinct keys React reuses the same node when moving
  // between the menu and a section, or between two sections.
  return (
    <div key={tab} className="animate-view flex h-full min-h-0 flex-col">
      {/* Header and body each centre a column of the same width, so the back
          arrow lines up with the left edge of the content below it. */}
      <div className="px-6 pt-6">
        <div className="mx-auto mb-6 flex w-full max-w-2xl items-center gap-3">
          <button
            onClick={() => setTab(null)}
            className="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-text-muted transition hover:bg-surface-2 hover:text-text"
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
      </div>

      {/* The scroller spans the full width so its bar sits at the window edge
          without a negative margin, which would have thrown the centred column
          off by half the padding. scrollbar-gutter reserves the same space on
          both sides, so content does not shift when the bar appears. */}
      <div
        className={`min-h-0 flex-1 px-6 pb-6 ${
          tab === "logs"
            ? "flex flex-col"
            : "overflow-y-auto [scrollbar-gutter:stable_both-edges]"
        }`}
      >
        {tab === "appearance" && <Appearance settings={settings} set={set} />}
        {tab === "application" && <Application settings={settings} set={set} />}
        {tab === "connection" && (
          <Connection settings={settings} set={set} disabled={disabled} />
        )}
        {tab === "logs" && <Logs />}
        {tab === "developer" && (
          <Developer settings={settings} set={set} onApply={onChange} />
        )}
        {tab === "about" && (
          <About
            appInfo={appInfo}
            devMode={settings.devMode}
            onUnlock={() => set({ devMode: true })}
          />
        )}
      </div>
    </div>
  );
}

type SetFn = (patch: Partial<AppSettings>) => void;

/* --------------------------------- Appearance -------------------------------- */

function Appearance({ settings, set }: { settings: AppSettings; set: SetFn }) {
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-7">
      <Section icon={<Palette size={16} />} title="Тема">
        <ThemePicker
          value={settings.theme}
          onChange={(id) => set({ theme: id })}
        />
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
          saved={settings.savedColors ?? []}
          onApply={(hex) => set({ accent: hex })}
          onSavedChange={(next) => set({ savedColors: next })}
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
              className={`flex flex-col items-center gap-2 border px-3 py-2.5 text-xs transition ${
                settings.radius === r.id
                  ? "border-accent/60 bg-surface-2 text-text"
                  : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
              }`}
              style={{ borderRadius: r.value }}
            >
              {/* 48px is the floor here: a radius stops reading as distinct once
                  it reaches half the height, so at the original 28px the 14px
                  and 22px presets drew an identical shape. 48px keeps all three
                  apart while staying compact. */}
              <span
                className="h-12 w-16 border border-text-faint"
                style={{ borderRadius: r.value }}
              />
              {r.name}
            </button>
          ))}
        </div>
      </Section>

      <Section icon={<Sparkles size={16} />} title="Анимации">
        <AnimationPicker
          value={settings.animation || "rise"}
          onChange={(id) => set({ animation: id })}
        />
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

// ThemePicker shows the plain presets first and keeps the rest behind a toggle,
// so the list does not open as a wall of twelve swatches.
function ThemePicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const hidden = THEME_PRESETS.length - THEMES_VISIBLE;

  // Only `expanded` decides what is shown. An earlier version force-expanded
  // the list when the current theme was one of the hidden ones, which also took
  // the toggle away and left no way to collapse it again — collapsed means
  // collapsed, even if that hides the active swatch.
  const shown = expanded
    ? THEME_PRESETS
    : THEME_PRESETS.slice(0, THEMES_VISIBLE);

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-3 gap-3">
        {shown.map((p) => (
          <button
            key={p.id}
            onClick={() => onChange(p.id)}
            className={`flex flex-col gap-2 rounded-lg border p-3 text-left transition ${
              value === p.id
                ? "border-accent/60 bg-surface-2"
                : "border-border bg-surface hover:bg-surface-2/60"
            }`}
          >
            <div className="flex gap-1">
              {[p.colors.bg, p.colors.surface2, p.colors.border].map((c, i) => (
                <span
                  key={i}
                  className="h-6 flex-1 rounded"
                  style={{ background: c }}
                />
              ))}
            </div>
            <span className="text-xs text-text">{p.name}</span>
          </button>
        ))}
      </div>

      {hidden > 0 && (
        <button
          onClick={() => setExpanded((e) => !e)}
          className="flex items-center justify-center gap-1.5 rounded-lg border border-border bg-surface py-2 text-xs text-text-muted transition hover:bg-surface-2/60 hover:text-text"
        >
          <ChevronDown
            size={14}
            className={`transition-transform duration-200 ${
              expanded ? "rotate-180" : ""
            }`}
          />
          {expanded ? "Свернуть" : `Показать ещё ${hidden}`}
        </button>
      )}
    </div>
  );
}

// AnimationPicker lists the entrance presets. No preview tile: the choice is
// visible on the next screen change anyway, and the user asked for it gone.
function AnimationPicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (id: string) => void;
}) {
  return (
    <div className="grid grid-cols-3 gap-2">
      {ANIMATIONS.map((a) => (
        <button
          key={a.id}
          onClick={() => onChange(a.id)}
          className={`flex flex-col items-start gap-0.5 rounded-lg border px-3 py-2.5 text-left transition ${
            value === a.id
              ? "border-accent/60 bg-surface-2 text-text"
              : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
          }`}
        >
          <span className="text-sm">{a.name}</span>
          <span className="text-[11px] text-text-faint">{a.desc}</span>
        </button>
      ))}
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

// MAX_SAVED_COLORS caps the user's palette so the strip stays one tidy row.
const MAX_SAVED_COLORS = 16;

// CustomAccent opens the picker and owns the user's saved-colour list.
function CustomAccent({
  value,
  saved,
  onApply,
  onSavedChange,
}: {
  value: string;
  saved: string[];
  onApply: (hex: string) => void;
  onSavedChange: (next: string[]) => void;
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
          saved={saved}
          onClose={() => setOpen(false)}
          onApply={(hex) => {
            onApply(hex);
            setOpen(false);
          }}
          onSave={(hex) => {
            const h = hex.toLowerCase();
            // Re-saving an existing colour is a no-op rather than a duplicate.
            if (saved.includes(h)) return;
            onSavedChange([...saved, h].slice(-MAX_SAVED_COLORS));
          }}
          onRemove={(hex) =>
            onSavedChange(saved.filter((c) => c !== hex.toLowerCase()))
          }
        />
      )}
    </>
  );
}

// clamp01 keeps a normalised drag position inside the control.
const clamp01 = (n: number) => (n < 0 ? 0 : n > 1 ? 1 : n);

// ColorPickerModal is a self-contained picker drawn in the app's own language:
// a saturation/value field, a hue rail, a hex field and quick shades. It
// deliberately avoids <input type="color">, whose popup is the browser's own
// widget and looks nothing like the rest of the client.
function ColorPickerModal({
  initial,
  saved,
  onClose,
  onApply,
  onSave,
  onRemove,
}: {
  initial: string;
  saved: string[];
  onClose: () => void;
  onApply: (hex: string) => void;
  onSave: (hex: string) => void;
  onRemove: (hex: string) => void;
}) {
  // HSV is authoritative while picking; hex is derived. Going through hex on
  // every drag would lose the hue as soon as the value reached black or white.
  const [hsv, setHsv] = useState(
    () => hexToHsv(initial) ?? { h: 230, s: 0.45, v: 1 }
  );
  const [text, setText] = useState(initial.toLowerCase());

  const hex = hsvToHex(hsv.h, hsv.s, hsv.v);
  const typedValid = isHex(text);
  const alreadySaved = saved.includes(hex.toLowerCase());

  const svRef = useRef<HTMLDivElement>(null);
  const hueRef = useRef<HTMLDivElement>(null);

  // Committing through here keeps the hex field in step with the handles.
  const commit = (next: HSV) => {
    setHsv(next);
    setText(hsvToHex(next.h, next.s, next.v));
  };

  const pickFromHex = (v: string) => {
    setText(v);
    const parsed = hexToHsv(v);
    if (parsed) setHsv(parsed);
  };

  // Pointer capture means a drag that leaves the control still tracks.
  const drag =
    (
      ref: React.RefObject<HTMLDivElement | null>,
      apply: (x: number, y: number) => void
    ) =>
    (e: React.PointerEvent) => {
      if (e.type === "pointermove" && e.buttons !== 1) return;
      const el = ref.current;
      if (!el) return;
      if (e.type === "pointerdown") {
        el.setPointerCapture(e.pointerId);
      }
      const r = el.getBoundingClientRect();
      apply(
        clamp01((e.clientX - r.left) / r.width),
        clamp01((e.clientY - r.top) / r.height)
      );
    };

  const onSV = drag(svRef, (x, y) => commit({ ...hsv, s: x, v: 1 - y }));
  const onHue = drag(hueRef, (x) => commit({ ...hsv, h: x * 360 }));

  // Chromium ships an eyedropper; offer it only where it actually exists.
  const eyeDropper = (window as any).EyeDropper;
  const pickFromScreen = async () => {
    try {
      const res = await new eyeDropper().open();
      if (res?.sRGBHex) pickFromHex(res.sRGBHex);
    } catch {
      /* the user dismissed the eyedropper */
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/60 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-view w-full max-w-xs rounded-xl border border-border bg-surface p-4 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-medium text-text">Цвет акцента</h2>
          <button
            onClick={onClose}
            className="no-drag text-text-faint transition hover:text-text"
          >
            <X size={16} />
          </button>
        </div>

        {/* Saturation (x) × value (y) over the current hue */}
        <div
          ref={svRef}
          onPointerDown={onSV}
          onPointerMove={onSV}
          className="relative mb-3 h-40 w-full cursor-crosshair touch-none overflow-hidden rounded-lg border border-border"
          style={{
            background: `linear-gradient(to top, #000, transparent), linear-gradient(to right, #fff, hsl(${hsv.h} 100% 50%))`,
          }}
        >
          <span
            className="pointer-events-none absolute h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow-[0_0_0_1px_rgba(0,0,0,0.45)]"
            style={{
              left: `${hsv.s * 100}%`,
              top: `${(1 - hsv.v) * 100}%`,
              background: hex,
            }}
          />
        </div>

        {/* Hue rail */}
        <div
          ref={hueRef}
          onPointerDown={onHue}
          onPointerMove={onHue}
          className="relative mb-3 h-3 w-full cursor-pointer touch-none rounded-full border border-border"
          style={{
            background:
              "linear-gradient(to right,#f00 0%,#ff0 17%,#0f0 33%,#0ff 50%,#00f 67%,#f0f 83%,#f00 100%)",
          }}
        >
          <span
            className="pointer-events-none absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow-[0_0_0_1px_rgba(0,0,0,0.45)]"
            style={{
              left: `${(hsv.h / 360) * 100}%`,
              background: `hsl(${hsv.h} 100% 50%)`,
            }}
          />
        </div>

        {/* Swatch + hex + eyedropper */}
        <div className="mb-3 flex items-center gap-2">
          <span
            className="h-9 w-9 shrink-0 rounded-lg border border-border"
            style={{ background: hex }}
          />
          <input
            value={text}
            onChange={(e) => pickFromHex(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && typedValid && onApply(hex)}
            placeholder="#ffffff"
            spellCheck={false}
            className={`min-w-0 flex-1 rounded-lg border bg-bg px-3 py-2 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 ${
              text.length > 1 && !typedValid ? "border-danger/50" : "border-border"
            }`}
          />
          {eyeDropper && (
            <button
              onClick={pickFromScreen}
              title="Взять цвет с экрана"
              className="shrink-0 rounded-lg border border-border p-2 text-text-faint transition hover:bg-surface-2 hover:text-text"
            >
              <Pipette size={15} />
            </button>
          )}
        </div>

        {/* The user's own palette. The ready-made shades that used to sit here
            were the accent palette again, one screen up. */}
        <div className="mb-4">
          <div className="mb-1.5 flex items-center justify-between">
            <span className="font-mono text-[11px] uppercase tracking-wide text-text-faint">
              Сохранённые
            </span>
            <button
              onClick={() => onSave(hex)}
              disabled={alreadySaved}
              title={alreadySaved ? "Уже сохранён" : "Сохранить текущий цвет"}
              className="flex items-center gap-1 rounded-md border border-border px-1.5 py-0.5 text-[11px] text-text-muted transition hover:bg-surface-2 hover:text-text disabled:cursor-not-allowed disabled:opacity-40"
            >
              {alreadySaved ? <Check size={11} /> : <Plus size={11} />}
              {alreadySaved ? "Сохранён" : "Сохранить"}
            </button>
          </div>

          {saved.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border px-3 py-2.5 text-center text-[11px] text-text-faint">
              Пока пусто — подберите цвет и нажмите «Сохранить»
            </div>
          ) : (
            <div className="grid grid-cols-8 gap-1.5">
              {saved.map((c) => (
                <div key={c} className="group relative">
                  <button
                    onClick={() => pickFromHex(c)}
                    title={c}
                    className={`h-6 w-full rounded-md border transition ${
                      hex.toLowerCase() === c
                        ? "border-text"
                        : "border-transparent hover:border-border"
                    }`}
                    style={{ background: c }}
                  />
                  <button
                    onClick={() => onRemove(c)}
                    title="Убрать"
                    className="absolute -right-1 -top-1 hidden h-3.5 w-3.5 place-items-center rounded-full border border-border bg-surface text-text-faint transition hover:text-danger group-hover:grid"
                  >
                    <X size={8} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="flex gap-2">
          <button
            onClick={onClose}
            className="flex-1 rounded-lg border border-border py-2 text-sm text-text-muted transition hover:bg-surface-2"
          >
            Отмена
          </button>
          <button
            onClick={() => onApply(hex)}
            className="flex-1 rounded-lg bg-accent py-2 text-sm font-medium text-bg transition hover:bg-accent-soft"
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
    <div className="mx-auto flex w-full max-w-xl flex-col gap-4">
      {/* Same single-card layout as the Connection screen so the two settings
          pages read as one system. */}
      <div className="overflow-hidden rounded-lg border border-border bg-surface">
        <Row
          label="Запускать с Windows"
          hint="Задача в планировщике с правами администратора"
        >
          <Toggle
            checked={settings.launchAtStartup}
            onChange={(v) => set({ launchAtStartup: v })}
          />
        </Row>
        <Row
          label="Подключаться при старте"
          hint="Поднимать туннель к последнему профилю"
        >
          <Toggle
            checked={settings.autoConnect}
            onChange={(v) => set({ autoConnect: v })}
          />
        </Row>
        <Row
          label="Сворачивать в трей"
          hint="Прятать окно в область уведомлений, а не на панель задач"
        >
          <Toggle
            checked={settings.minimizeToTray}
            onChange={(v) => set({ minimizeToTray: v })}
          />
        </Row>
        <Row
          label="Режим демонстрации"
          hint="Скрывать адреса и ссылки — для скриншотов и записи экрана"
        >
          <Toggle
            checked={settings.demoMode}
            onChange={(v) => set({ demoMode: v })}
          />
        </Row>
      </div>

      {settings.demoMode && (
        <div className="flex items-start gap-2 rounded-lg border border-accent/40 bg-accent/10 px-4 py-2.5 text-xs text-accent">
          <EyeOff size={14} className="mt-0.5 shrink-0" />
          <span>
            Скрыты адреса серверов, домены подписок, share-ссылки и дампы в
            разделе разработчика. Названия серверов, DNS и логи остаются
            видимыми.
          </span>
        </div>
      )}
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
    <div className="mx-auto flex w-full max-w-xl flex-col gap-4">
      {disabled && (
        <div className="flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2.5 text-xs text-text-muted">
          <Lock size={14} className="shrink-0 text-text-faint" />
          Параметры заблокированы, пока туннель активен
        </div>
      )}

      {/* No overflow-hidden on the card: the stack dropdown is absolutely
          positioned and would be clipped by it. Nothing inside paints over the
          rounded corners, so the clip was not buying anything. */}
      <div className="rounded-lg border border-border bg-surface">
        <Row label="Адаптер" hint="Имя WinTun-адаптера в системе">
          <input
            value={settings.tunName}
            disabled={disabled}
            onChange={(e) => set({ tunName: e.target.value })}
            placeholder="TomorrowTun"
            spellCheck={false}
            className={`${cellInput} w-44`}
          />
        </Row>

        <Row label="Сетевой стек" hint="Как ядро обрабатывает пакеты TUN">
          <Select
            value={settings.stack || "gvisor"}
            disabled={disabled}
            onChange={(v) => set({ stack: v })}
            options={STACKS}
          />
        </Row>

        <Row label="MTU" hint="0 — значение ядра по умолчанию">
          <input
            type="number"
            value={settings.mtu || 0}
            disabled={disabled}
            onChange={(e) => set({ mtu: parseInt(e.target.value) || 0 })}
            placeholder="0"
            className={`${cellInput} w-44`}
          />
        </Row>
      </div>

      <div className="rounded-lg border border-border bg-surface">
        <Row label="DNS основной" hint="Резолвер внутри туннеля">
          <input
            value={settings.dns}
            disabled={disabled}
            onChange={(e) => set({ dns: e.target.value })}
            placeholder="1.1.1.1"
            spellCheck={false}
            className={`${cellInput} w-44`}
          />
        </Row>

        <Row label="DNS резервный" hint="Для запросов в обход туннеля">
          <input
            value={settings.dnsFallback}
            disabled={disabled}
            onChange={(e) => set({ dnsFallback: e.target.value })}
            placeholder="8.8.8.8"
            spellCheck={false}
            className={`${cellInput} w-44`}
          />
        </Row>
      </div>
    </div>
  );
}

// STACKS are the TUN network stacks sing-box accepts.
const STACKS: SelectOption[] = [
  { id: "gvisor", label: "gvisor", hint: "Пользовательский стек, надёжнее" },
  { id: "mixed", label: "mixed", hint: "gvisor для TCP, system для UDP" },
  { id: "system", label: "system", hint: "Системный стек, быстрее" },
];

// Row is one label/control line inside a settings card.
function Row({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-border/60 px-4 py-3 last:border-0">
      <div className="min-w-0">
        <div className="text-sm text-text">{label}</div>
        {hint && <div className="text-xs text-text-faint">{hint}</div>}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

export interface SelectOption {
  id: string;
  label: string;
  hint?: string;
}

// Select is a small dropdown: the trigger shows the current value and its
// chevron rotates while a compact menu is open. Built by hand rather than with
// <select>, whose popup is drawn by the OS and ignores the app's theme.
function Select({
  value,
  options,
  disabled,
  onChange,
}: {
  value: string;
  options: SelectOption[];
  disabled?: boolean;
  onChange: (v: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const current = options.find((o) => o.id === value) ?? options[0];

  // Dismiss on an outside click or Escape, the way a native menu behaves.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  // A disabled control must not be left hanging open.
  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
        className={`flex w-44 items-center justify-between gap-2 rounded-lg border bg-bg px-3 py-2 font-mono text-sm text-text transition disabled:cursor-not-allowed disabled:opacity-60 ${
          open ? "border-accent/60" : "border-border hover:border-text-faint"
        }`}
      >
        {current.label}
        <ChevronDown
          size={14}
          className={`shrink-0 text-text-faint transition-transform duration-200 ${
            open ? "rotate-180" : ""
          }`}
        />
      </button>

      {open && (
        <div className="animate-pop absolute right-0 z-30 mt-1.5 w-56 overflow-hidden rounded-lg border border-border bg-surface-2 shadow-xl">
          {options.map((o) => (
            <button
              key={o.id}
              type="button"
              onClick={() => {
                onChange(o.id);
                setOpen(false);
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-left transition hover:bg-surface"
            >
              <span className="min-w-0 flex-1">
                <span
                  className={`block font-mono text-sm ${
                    o.id === value ? "text-text" : "text-text-muted"
                  }`}
                >
                  {o.label}
                </span>
                {o.hint && (
                  <span className="block text-[11px] text-text-faint">
                    {o.hint}
                  </span>
                )}
              </span>
              {o.id === value && (
                <Check size={14} className="shrink-0 text-accent" />
              )}
            </button>
          ))}
        </div>
      )}
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
    <div className="mx-auto flex h-full min-h-0 w-full max-w-2xl flex-col gap-3">
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

function Developer({
  settings,
  set,
  onApply,
}: {
  settings: AppSettings;
  set: SetFn;
  onApply: (s: AppSettings) => void;
}) {
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-7 pb-2">
      {/* Every one of these dumps carries server addresses, uuids or passwords
          in plain text, so demo mode replaces them outright rather than trying
          to redact JSON field by field. */}
      <ActiveProfileDump hide={settings.demoMode} />
      <CoreConfig core={settings.core} hide={settings.demoMode} />
      <SettingsDump hide={settings.demoMode} />
      <NetDiag hide={settings.demoMode} />
      <Simulation />
      <Maintenance />

      <Section icon={<Bug size={16} />} title="Режим разработчика">
        <ActionRow
          icon={<BugOff size={15} />}
          title="Выключить режим разработчика"
          desc="Раздел исчезнет из настроек. Вернуть — снова 10 раз нажать на название в «О приложении»"
          action={
            <ToolButton onClick={() => set({ devMode: false })}>
              Выключить
            </ToolButton>
          }
        />
      </Section>

      <DangerZone onApply={onApply} />
    </div>
  );
}

// ActiveProfileDump shows the currently selected server as it is stored — every
// field parsed out of the share link, including the ones no screen displays.
function ActiveProfileDump({ hide }: { hide: boolean }) {
  const [json, setJson] = useState("");
  const [err, setErr] = useState("");

  const load = async () => {
    setErr("");
    try {
      setJson((await GetActiveProfileJSON()) as string);
    } catch (e: any) {
      setJson("");
      setErr(String(e?.message ?? e));
    }
  };

  return (
    <Section icon={<Server size={16} />} title="Активный конфиг">
      <p className="text-xs leading-relaxed text-text-muted">
        Выбранный сейчас сервер целиком: протокол, транспорт, TLS и ключи —
        всё, что удалось разобрать из ссылки.
      </p>
      {hide ? (
        <HiddenNotice />
      ) : (
        <>
          <div className="flex gap-2">
            <ToolButton onClick={load} primary>
              Показать конфиг
            </ToolButton>
            {json && <CopyButton text={json} labelled />}
          </div>
          {err && <ErrorBox text={err} />}
          {json && <Output text={json} />}
        </>
      )}
    </Section>
  );
}

// SIM_SCENARIOS mirrors the ids accepted by the SimulateStatus binding.
const SIM_SCENARIOS: { id: string; label: string; tone?: "ok" | "danger" }[] = [
  { id: "connecting", label: "Подключение…" },
  { id: "connected", label: "Подключено", tone: "ok" },
  { id: "disconnected", label: "Отключено" },
  { id: "error", label: "Ошибка подключения", tone: "danger" },
  { id: "no-internet", label: "Нет интернета", tone: "danger" },
  { id: "no-wintun", label: "Нет wintun.dll", tone: "danger" },
  { id: "no-admin", label: "Нет прав администратора", tone: "danger" },
  { id: "timeout", label: "Таймаут", tone: "danger" },
  { id: "handshake", label: "Обрыв рукопожатия", tone: "danger" },
];

// Simulation pushes fake connection states to the UI so every screen can be
// checked without a working server.
function Simulation() {
  const [active, setActive] = useState<string | null>(null);

  const play = async (id: string) => {
    try {
      await SimulateStatus(id);
      setActive(id);
    } catch (e: any) {
      push(String(e?.message ?? e), "error");
    }
  };

  const stop = async () => {
    await StopSimulation();
    setActive(null);
    push("Эмуляция остановлена", "info");
  };

  return (
    <Section icon={<PlayCircle size={16} />} title="Эмуляция состояний">
      <p className="text-xs leading-relaxed text-text-muted">
        Подменяет то, что показывает интерфейс — туннель не поднимается и ядро
        не запускается. «Подключено» ещё и крутит счётчики трафика. Реальное
        событие от движка перебивает эмуляцию.
      </p>
      <div className="grid grid-cols-3 gap-2">
        {SIM_SCENARIOS.map((s) => (
          <button
            key={s.id}
            onClick={() => play(s.id)}
            className={`rounded-lg border px-2 py-2 text-xs transition ${
              active === s.id
                ? "border-accent/60 bg-surface-2 text-text"
                : s.tone === "danger"
                ? "border-border bg-surface text-danger/80 hover:bg-surface-2/60"
                : s.tone === "ok"
                ? "border-border bg-surface text-ok/90 hover:bg-surface-2/60"
                : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
            }`}
          >
            {s.label}
          </button>
        ))}
      </div>
      {active && (
        <div className="flex items-center gap-2 rounded-lg border border-accent/40 bg-accent/10 px-3 py-2 text-xs text-accent">
          <PlayCircle size={14} />
          <span className="flex-1">Эмуляция активна — состояние подменено</span>
          <button
            onClick={stop}
            className="flex items-center gap-1 rounded-md border border-accent/40 px-2 py-1 transition hover:bg-accent/15"
          >
            <Square size={11} />
            Остановить
          </button>
        </div>
      )}
    </Section>
  );
}

// CoreConfig previews the JSON handed to the active core.
function CoreConfig({ core, hide }: { core: string; hide: boolean }) {
  const [config, setConfig] = useState("");
  const [err, setErr] = useState("");

  const load = async () => {
    setErr("");
    try {
      setConfig((await PreviewConfig()) as string);
    } catch (e: any) {
      setConfig("");
      setErr(String(e?.message ?? e));
    }
  };

  return (
    <Section icon={<FileJson size={16} />} title="Конфигурация ядра">
      <p className="text-xs leading-relaxed text-text-muted">
        Сгенерированный JSON, который передаётся активному ядру (
        <span className="font-mono text-text">{core}</span>) для выбранного
        профиля.
      </p>
      {hide ? (
        <HiddenNotice />
      ) : (
        <>
          <div className="flex gap-2">
            <ToolButton onClick={load} primary>
              Показать конфиг
            </ToolButton>
            {config && <CopyButton text={config} labelled />}
          </div>
          {err && <ErrorBox text={err} />}
          {config && <Output text={config} />}
        </>
      )}
    </Section>
  );
}

// SettingsDump shows the settings exactly as persisted on disk.
function SettingsDump({ hide }: { hide: boolean }) {
  const [json, setJson] = useState("");

  return (
    <Section icon={<Terminal size={16} />} title="Настройки (JSON)">
      <p className="text-xs leading-relaxed text-text-muted">
        Содержимое <span className="font-mono text-text">settings.json</span> —
        включая поля, которых нет в интерфейсе.
      </p>
      {hide ? (
        <HiddenNotice />
      ) : (
        <>
          <div className="flex gap-2">
            <ToolButton
              onClick={async () => setJson((await GetSettingsJSON()) as string)}
            >
              Показать
            </ToolButton>
            {json && <CopyButton text={json} labelled />}
          </div>
          {json && <Output text={json} />}
        </>
      )}
    </Section>
  );
}

// NetDiag dumps the interface and IPv4 route tables.
function NetDiag({ hide }: { hide: boolean }) {
  const [out, setOut] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const load = async () => {
    setBusy(true);
    setErr("");
    try {
      setOut((await RunNetDiag()) as string);
    } catch (e: any) {
      setOut("");
      setErr(String(e?.message ?? e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Section icon={<Network size={16} />} title="Сетевая диагностика">
      <p className="text-xs leading-relaxed text-text-muted">
        Таблица интерфейсов и маршрутов IPv4 — здесь видно, поднялся ли TUN и
        куда уходит трафик по умолчанию.
      </p>
      {hide ? (
        <HiddenNotice />
      ) : (
        <>
          <div className="flex gap-2">
            <ToolButton onClick={load} disabled={busy}>
              <RefreshCw size={14} className={busy ? "animate-spin" : ""} />
              {busy ? "Собираю…" : "Собрать"}
            </ToolButton>
            {out && <CopyButton text={out} labelled />}
          </div>
          {err && <ErrorBox text={err} />}
          {out && <Output text={out} />}
        </>
      )}
    </Section>
  );
}

// HiddenNotice stands in for content demo mode is withholding.
function HiddenNotice() {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-border bg-bg px-3 py-2.5 text-xs text-text-muted">
      <EyeOff size={14} className="shrink-0 text-text-faint" />
      Скрыто в режиме демонстрации
    </div>
  );
}

// Maintenance groups the non-destructive recovery tools.
function Maintenance() {
  const exportDiag = async () => {
    try {
      const path = (await ExportDiagnostics()) as string;
      push(`Отчёт сохранён: ${path}`, "ok");
    } catch (e: any) {
      push(String(e?.message ?? e), "error");
    }
  };

  return (
    <Section icon={<Download size={16} />} title="Обслуживание">
      <div className="flex flex-col gap-2">
        <ActionRow
          icon={<Download size={15} />}
          title="Выгрузить отчёт"
          desc="Среда, настройки и лог ядра в один текстовый файл"
          action={<ToolButton onClick={exportDiag}>Выгрузить</ToolButton>}
        />
      </div>
    </Section>
  );
}

// DangerZone sits at the very bottom and holds the irreversible actions, each
// behind a two-step confirmation.
function DangerZone({ onApply }: { onApply: (s: AppSettings) => void }) {
  const doResetSettings = async () => {
    await ResetSettings();
    onApply((await GetSettings()) as AppSettings);
    push("Настройки сброшены", "ok");
  };

  // ResetAllData clears DevMode too, so this section disappears with it — the
  // frontend reloads through the "app:datareset" event fired by the backend.
  const doResetAll = async () => {
    await ResetAllData();
  };

  return (
    <section className="flex flex-col gap-2.5 rounded-lg border border-danger/30 bg-danger/[0.04] p-4">
      <div className="flex items-center gap-2 font-mono text-xs uppercase tracking-wide text-danger">
        <AlertTriangle size={16} />
        Опасная зона
      </div>
      <div className="flex flex-col gap-2">
        <ConfirmRow
          icon={<RotateCcw size={15} />}
          title="Сбросить настройки"
          desc="Вернуть параметры по умолчанию. Профили и подписки останутся"
          cta="Сбросить"
          confirmCta="Точно сбросить"
          onConfirm={doResetSettings}
        />
        <ConfirmRow
          icon={<Trash size={15} />}
          title="Удалить все данные"
          desc="Профили, подписки и настройки — всё как после установки. Туннель будет разорван, режим разработчика выключится"
          cta="Удалить всё"
          confirmCta="Да, удалить всё"
          onConfirm={doResetAll}
        />
      </div>
    </section>
  );
}

// ConfirmRow is an action row whose button turns into a confirm/cancel pair on
// the first click, so an irreversible action always takes two deliberate taps.
function ConfirmRow({
  icon,
  title,
  desc,
  cta,
  confirmCta,
  onConfirm,
}: {
  icon: React.ReactNode;
  title: string;
  desc: string;
  cta: string;
  confirmCta: string;
  onConfirm: () => Promise<void>;
}) {
  const [armed, setArmed] = useState(false);
  const [busy, setBusy] = useState(false);

  const go = async () => {
    setBusy(true);
    try {
      await onConfirm();
    } catch (e: any) {
      push(String(e?.message ?? e), "error");
    } finally {
      setBusy(false);
      setArmed(false);
    }
  };

  return (
    <ActionRow
      icon={icon}
      title={title}
      desc={desc}
      action={
        armed ? (
          <div className="flex gap-2">
            <ToolButton onClick={() => setArmed(false)} disabled={busy}>
              Отмена
            </ToolButton>
            <ToolButton onClick={go} danger disabled={busy}>
              {confirmCta}
            </ToolButton>
          </div>
        ) : (
          <ToolButton onClick={() => setArmed(true)} danger>
            {cta}
          </ToolButton>
        )
      }
    />
  );
}

function ActionRow({
  icon,
  title,
  desc,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  desc: string;
  action: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border bg-surface px-4 py-3">
      <span className="shrink-0 text-text-faint">{icon}</span>
      <div className="min-w-0 flex-1">
        <div className="text-sm text-text">{title}</div>
        <div className="text-xs text-text-faint">{desc}</div>
      </div>
      <div className="shrink-0">{action}</div>
    </div>
  );
}

/* ----------------------------- Developer helpers ----------------------------- */

function Output({ text }: { text: string }) {
  return (
    <pre className="max-h-96 overflow-auto rounded-lg border border-border bg-bg p-3 font-mono text-[11px] leading-relaxed text-text-muted">
      {text}
    </pre>
  );
}

function ErrorBox({ text }: { text: string }) {
  return (
    <div className="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-xs text-danger">
      {text}
    </div>
  );
}

function ToolButton({
  onClick,
  children,
  primary,
  danger,
  disabled,
}: {
  onClick: () => void;
  children: React.ReactNode;
  primary?: boolean;
  danger?: boolean;
  disabled?: boolean;
}) {
  const tone = primary
    ? "bg-accent text-bg hover:bg-accent-soft"
    : danger
    ? "border border-danger/40 bg-danger/10 text-danger hover:bg-danger/20"
    : "border border-border bg-surface text-text-muted hover:bg-surface-2 hover:text-text";
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={`flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50 ${tone}`}
    >
      {children}
    </button>
  );
}

function CopyButton({ text, labelled }: { text: string; labelled?: boolean }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  if (!labelled) {
    return (
      <button
        onClick={copy}
        title="Копировать"
        className="shrink-0 rounded-md p-1.5 text-text-faint transition hover:bg-surface-2 hover:text-text"
      >
        {copied ? <Check size={14} /> : <Copy size={14} />}
      </button>
    );
  }
  return (
    <ToolButton onClick={copy}>
      {copied ? <Check size={14} /> : <Copy size={14} />}
      {copied ? "Скопировано" : "Копировать"}
    </ToolButton>
  );
}

/* ----------------------------------- About ----------------------------------- */

function About({
  appInfo,
  devMode,
  onUnlock,
}: {
  appInfo: AppInfo | null;
  devMode: boolean;
  onUnlock: () => void;
}) {
  // Tapping the client name repeatedly unlocks the developer section. The
  // counter only lives while the About page is mounted, so leaving resets it.
  const [taps, setTaps] = useState(0);
  const left = TAPS_TO_UNLOCK - taps;

  const tap = () => {
    if (devMode) return;
    const n = taps + 1;
    if (n >= TAPS_TO_UNLOCK) {
      setTaps(0);
      onUnlock();
      push("Режим разработчика включён", "ok");
      return;
    }
    setTaps(n);
  };

  // Full column width, like every other settings page — a narrow card left the
  // section looking half-empty.
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <div className="overflow-hidden rounded-lg border border-border bg-surface">
        {/* Hero: the logo carries the name, so no wordmark repeats it in text.
            The soft accent wash behind it ties a fixed brand colour to whatever
            accent the user picked. */}
        <div className="relative flex flex-col items-center gap-5 px-8 pb-10 pt-14">
          <div
            aria-hidden
            className="pointer-events-none absolute inset-x-0 top-0 h-56 bg-[radial-gradient(ellipse_at_top,var(--color-accent)_0%,transparent_70%)] opacity-[0.08]"
          />

          {/* The logo is the unlock target — the usual place to tap. */}
          <button
            onClick={tap}
            title="TomorrowClient"
            className="no-drag relative select-none outline-none transition active:scale-[0.98]"
          >
            <img
              src={logo}
              alt="Tomorrow"
              draggable={false}
              className="h-24 w-auto"
            />
          </button>

          <div className="relative flex items-center gap-2">
            <span className="rounded-md border border-border bg-bg px-2.5 py-1 font-mono text-xs text-text-muted">
              v{appInfo?.version ?? "—"}
            </span>
            {devMode && (
              <span className="rounded-md bg-accent/15 px-2.5 py-1 font-mono text-xs text-accent">
                DEV
              </span>
            )}
          </div>

          {!devMode && taps >= 3 && (
            <div className="relative font-mono text-[11px] text-text-faint">
              ещё {left} {plural(left, "шаг", "шага", "шагов")}…
            </div>
          )}
        </div>

        <div className="flex flex-col gap-4 border-t border-border p-6">
          {/* The card is wide now, but a line of prose that wide is hard to
              read, so the paragraph keeps its own measure. */}
          <p className="mx-auto max-w-md text-center text-sm leading-relaxed text-text-muted">
            Минималистичный VPN-клиент для Windows на WinTun. Ядро sing-box
            встроено в приложение и работает в его процессе.
          </p>
          <div className="flex flex-wrap justify-center gap-1.5">
            {(appInfo?.builtWith ?? "Wails · Go · React · sing-box")
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
        </div>
      </div>

      <p className="text-center font-mono text-[11px] text-text-faint">
        {appInfo?.copyright ?? "© TomorrowClient"}
      </p>
    </div>
  );
}

/* --------------------------------- Primitives -------------------------------- */

// cellInput is the compact field used inside settings rows. Text reads from the
// left edge: right-aligning a name like "TomorrowTun" pushed it away from the
// caret and made the field look like a number box.
const cellInput =
  "rounded-lg border border-border bg-bg px-3 py-2 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 disabled:opacity-60";

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

// Toggle is the bare on/off pill, used on its own inside settings rows.
function Toggle({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
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
  );
}

