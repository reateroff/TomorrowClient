import { useEffect, useState } from "react";
import {
  Power,
  ArrowDownToLine,
  ArrowUpFromLine,
  Timer,
  Gauge,
  ChevronRight,
} from "lucide-react";
import type { Status, Profile, AppSettings } from "../types";
import { formatBytes, formatSpeed, formatUptime } from "../lib/format";
import { stripCountryPrefix } from "../flags";
import { PingProfile } from "../../wailsjs/go/main/App";
import FlagChip from "./FlagChip";

interface Props {
  status: Status;
  settings: AppSettings;
  activeProfile: Profile | null;
  onConnect: () => void;
  onDisconnect: () => void;
  onOpenConfigs: () => void;
}

// The main screen: the active-server chip sits top-left (opens Configs, uses the
// themed corner radius), a large filled connect button is centered, and a slim
// stats row is pinned to the bottom.
export default function ConnectionView({
  status,
  settings,
  activeProfile,
  onConnect,
  onDisconnect,
  onOpenConfigs,
}: Props) {
  const [now, setNow] = useState(Date.now());
  const [ping, setPing] = useState<number | null>(null);

  const state = status.state;
  const isConnected = state === "connected";
  const isBusy = state === "connecting";

  // Tick every second so the uptime clock stays live.
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  // Active ping: only while connected, refreshed every 3s. Reset on drop or
  // when the active server changes.
  const pid = activeProfile?.id;
  useEffect(() => {
    if (!isConnected || !pid) {
      setPing(null);
      return;
    }
    let alive = true;
    const run = async () => {
      const ms = await PingProfile(pid);
      if (alive) setPing(ms);
    };
    run();
    const id = setInterval(run, 3000);
    return () => {
      alive = false;
      clearInterval(id);
    };
  }, [isConnected, pid]);

  const label =
    state === "connected"
      ? "Вы подключены"
      : state === "connecting"
      ? "Подключение…"
      : state === "error"
      ? "Ошибка"
      : "Отключено";

  const handleClick = () => {
    if (isBusy) return;
    isConnected ? onDisconnect() : onConnect();
  };

  // Filled disc styling: accent when connected, surface otherwise.
  const disc = isConnected
    ? "bg-accent text-white shadow-xl shadow-accent/25"
    : state === "error"
    ? "bg-surface border border-danger/40 text-danger"
    : "bg-surface border border-border text-text-muted group-hover:border-text-faint group-hover:text-text";

  return (
    <div className="animate-fade-up flex h-full flex-col p-6">
      {/* Active-server chip, top-left → opens Configs. Radius follows theme. */}
      <button
        onClick={onOpenConfigs}
        className="no-drag group flex max-w-xs items-center gap-2.5 self-start rounded-lg border border-border bg-surface py-2 pl-2 pr-3 transition hover:bg-surface-2/60"
      >
        <span className="grid h-7 w-9 shrink-0 place-items-center overflow-hidden rounded-md border border-border bg-bg">
          <FlagChip name={activeProfile?.name ?? ""} />
        </span>
        <span className="min-w-0 truncate text-sm text-text">
          {activeProfile
            ? stripCountryPrefix(activeProfile.name)
            : "Сервер не выбран"}
        </span>
        <span className="shrink-0 rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[9px] tracking-wide text-text-muted uppercase">
          {settings.core}
        </span>
        <ChevronRight
          size={15}
          className="shrink-0 text-text-faint transition group-hover:text-text-muted"
        />
      </button>

      {/* Centered connect button + status */}
      <div className="flex flex-1 flex-col items-center justify-center gap-5">
        <button
          onClick={handleClick}
          disabled={isBusy || !activeProfile}
          className="no-drag group relative grid place-items-center disabled:cursor-not-allowed disabled:opacity-50"
        >
          {(isConnected || isBusy) && (
            <span
              className={`absolute inset-0 rounded-full blur-3xl ${
                isConnected ? "bg-accent/25" : "bg-accent/15"
              }`}
            />
          )}
          <span
            className={`relative grid h-52 w-52 place-items-center rounded-full transition ${disc} ${
              isBusy ? "animate-pulse" : ""
            }`}
          >
            <Power size={72} strokeWidth={2} />
          </span>
        </button>

        <div className="text-center">
          <div className="text-base font-medium text-text">{label}</div>
          {status.error && state === "error" && (
            <div className="mt-1 max-w-xs text-xs text-danger">
              {status.error}
            </div>
          )}
        </div>
      </div>

      {/* Slim stats row, pinned to the bottom */}
      <div className="mx-auto grid w-full max-w-sm grid-cols-4 gap-2">
        <Stat
          icon={<Gauge size={12} className="text-text-faint" />}
          label="Пинг"
          value={ping == null || ping < 0 ? "—" : `${ping} мс`}
          valueCls={pingTone(ping)}
        />
        <Stat
          icon={<Timer size={12} className="text-text-faint" />}
          label="Время"
          value={isConnected ? formatUptime(status.connectedAt, now) : "—"}
        />
        <Stat
          icon={<ArrowDownToLine size={12} className="text-text-faint" />}
          label="Скачано"
          value={formatBytes(status.stats.download)}
          valueCls="text-ok"
          sub={formatSpeed(status.stats.downloadSpeed)}
        />
        <Stat
          icon={<ArrowUpFromLine size={12} className="text-text-faint" />}
          label="Отдано"
          value={formatBytes(status.stats.upload)}
          valueCls="text-accent"
          sub={formatSpeed(status.stats.uploadSpeed)}
        />
      </div>
    </div>
  );
}

function pingTone(ping: number | null): string {
  if (ping == null || ping < 0) return "text-text";
  if (ping < 150) return "text-ok";
  if (ping < 400) return "text-amber";
  return "text-danger";
}

function Stat({
  icon,
  label,
  value,
  valueCls,
  sub,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  valueCls?: string;
  sub?: string;
}) {
  return (
    <div className="flex flex-col items-center gap-0.5 text-center">
      <div className="flex items-center gap-1 text-[9px] tracking-wide text-text-faint uppercase">
        {icon}
        {label}
      </div>
      <div className={`font-mono text-xs ${valueCls ?? "text-text"}`}>
        {value}
      </div>
      {sub && <div className="font-mono text-[9px] text-text-faint">{sub}</div>}
    </div>
  );
}
