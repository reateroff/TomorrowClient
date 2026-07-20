import { Cpu, Route, Globe2, Zap, Info } from "lucide-react";
import type { AppInfo, AppSettings, Core, RoutingMode } from "../types";

interface Props {
  settings: AppSettings;
  appInfo: AppInfo | null;
  disabled: boolean; // locked while connected
  onChange: (s: AppSettings) => void;
}

// App configuration: core engine, routing mode, DNS, auto-connect.
export default function SettingsView({
  settings,
  appInfo,
  disabled,
  onChange,
}: Props) {
  const set = (patch: Partial<AppSettings>) =>
    onChange({ ...settings, ...patch });

  return (
    <div className="animate-fade-up flex h-full flex-col gap-6 overflow-y-auto p-6">
      <div>
        <h1 className="text-base font-medium text-text">Настройки</h1>
        <p className="font-mono text-xs text-text-faint">
          {disabled ? "заблокировано во время соединения" : "конфигурация клиента"}
        </p>
      </div>

      {/* Core selector */}
      <Section icon={<Cpu size={16} className="text-accent" />} title="Ядро">
        <div className="grid grid-cols-2 gap-3">
          <CoreCard
            active={settings.core === "sing-box"}
            disabled={disabled}
            name="sing-box"
            desc="Нативный TUN + auto_route. Рекомендуется."
            onClick={() => set({ core: "sing-box" as Core })}
          />
          <CoreCard
            active={settings.core === "xray"}
            disabled={disabled}
            name="xray"
            desc="Xray-core + tun2socks через WinTun."
            onClick={() => set({ core: "xray" as Core })}
          />
        </div>
      </Section>

      {/* Routing mode */}
      <Section icon={<Route size={16} className="text-accent" />} title="Маршрутизация">
        <div className="grid grid-cols-2 gap-3">
          <Toggle
            active={settings.routingMode === "rules"}
            disabled={disabled}
            label="По правилам"
            desc="Локальная сеть напрямую"
            onClick={() => set({ routingMode: "rules" as RoutingMode })}
          />
          <Toggle
            active={settings.routingMode === "global"}
            disabled={disabled}
            label="Глобально"
            desc="Весь трафик через VPN"
            onClick={() => set({ routingMode: "global" as RoutingMode })}
          />
        </div>
      </Section>

      {/* DNS */}
      <Section icon={<Globe2 size={16} className="text-accent" />} title="DNS">
        <input
          value={settings.dns}
          disabled={disabled}
          onChange={(e) => set({ dns: e.target.value })}
          placeholder="1.1.1.1"
          className="w-full rounded-lg border border-border bg-bg px-3 py-2.5 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 disabled:opacity-60"
        />
      </Section>

      {/* Auto connect */}
      <Section icon={<Zap size={16} className="text-accent" />} title="Автозапуск">
        <label className="flex cursor-pointer items-center justify-between rounded-lg border border-border bg-surface px-4 py-3">
          <span className="text-sm text-text-muted">
            Подключаться при запуске
          </span>
          <input
            type="checkbox"
            checked={settings.autoConnect}
            onChange={(e) => set({ autoConnect: e.target.checked })}
            className="h-4 w-4 accent-[var(--color-accent)]"
          />
        </label>
      </Section>

      {/* About */}
      <Section icon={<Info size={16} className="text-accent" />} title="О приложении">
        <div className="flex flex-col gap-3 rounded-lg border border-border bg-surface px-4 py-4">
          <div className="flex items-center gap-3">
            <div className="grid h-10 w-10 place-items-center rounded-lg bg-accent/15 font-mono text-sm font-semibold text-accent">
              TC
            </div>
            <div>
              <div className="text-sm text-text">TomorrowClient</div>
              <div className="font-mono text-xs text-text-faint">
                v{appInfo?.version ?? "—"}
              </div>
            </div>
          </div>
          <p className="text-xs leading-relaxed text-text-muted">
            {appInfo?.builtWith ??
              "Wails · Go · React · sing-box · xray-core"}
          </p>
          <div className="border-t border-border pt-3 font-mono text-[11px] text-text-faint">
            {appInfo?.copyright ?? "© TomorrowClient"}
          </div>
        </div>
      </Section>
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
        {icon}
        {title}
      </div>
      {children}
    </section>
  );
}

function CoreCard({
  active,
  disabled,
  name,
  desc,
  onClick,
}: {
  active: boolean;
  disabled: boolean;
  name: string;
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
      <span className="font-mono text-sm text-text">{name}</span>
      <span className="text-xs text-text-faint">{desc}</span>
    </button>
  );
}

function Toggle({
  active,
  disabled,
  label,
  desc,
  onClick,
}: {
  active: boolean;
  disabled: boolean;
  label: string;
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
      <span className="text-sm text-text">{label}</span>
      <span className="text-xs text-text-faint">{desc}</span>
    </button>
  );
}
