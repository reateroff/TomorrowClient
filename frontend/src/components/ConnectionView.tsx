import { useEffect, useState } from "react";
import { Power, Cpu, ArrowDown, ArrowUp, Clock, Globe } from "lucide-react";
import type { Status, Profile, AppSettings } from "../types";
import { formatBytes, formatSpeed, formatUptime } from "../lib/format";

interface Props {
  status: Status;
  settings: AppSettings;
  activeProfile: Profile | null;
  onConnect: () => void;
  onDisconnect: () => void;
}

// The main screen: a big connect orb, the active server, live stats.
export default function ConnectionView({
  status,
  settings,
  activeProfile,
  onConnect,
  onDisconnect,
}: Props) {
  const [now, setNow] = useState(Date.now());

  // Tick every second so the uptime clock stays live.
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  const state = status.state;
  const isConnected = state === "connected";
  const isBusy = state === "connecting";

  const label =
    state === "connected"
      ? "Подключено"
      : state === "connecting"
      ? "Подключение…"
      : state === "error"
      ? "Ошибка"
      : "Отключено";

  const orbColor = isConnected
    ? "text-ok"
    : state === "error"
    ? "text-danger"
    : "text-text-muted";

  const handleClick = () => {
    if (isBusy) return;
    isConnected ? onDisconnect() : onConnect();
  };

  return (
    <div className="animate-fade-up flex h-full flex-col items-center justify-center gap-8 p-8">
      {/* Connect orb */}
      <div className="relative flex flex-col items-center gap-5">
        <button
          onClick={handleClick}
          disabled={isBusy || !activeProfile}
          className="no-drag group relative grid h-40 w-40 place-items-center rounded-full border border-border bg-surface transition disabled:cursor-not-allowed disabled:opacity-60"
        >
          {(isConnected || isBusy) && (
            <span
              className={`animate-orb absolute inset-0 rounded-full ${
                isConnected ? "bg-ok/10" : "bg-accent/10"
              }`}
            />
          )}
          <span
            className={`absolute inset-2 rounded-full border ${
              isConnected ? "border-ok/40" : "border-border"
            }`}
          />
          <Power
            size={44}
            className={`relative transition ${orbColor} ${
              isBusy ? "animate-pulse" : ""
            }`}
          />
        </button>

        <div className="text-center">
          <div className="font-mono text-sm tracking-widest text-text uppercase">
            {label}
          </div>
          {status.error && state === "error" && (
            <div className="mt-1 max-w-xs text-xs text-danger">
              {status.error}
            </div>
          )}
        </div>
      </div>

      {/* Active server */}
      <div className="flex w-full max-w-md items-center gap-3 rounded-lg border border-border bg-surface px-4 py-3">
        <Globe size={18} className="shrink-0 text-accent" />
        <div className="min-w-0 flex-1">
          {activeProfile ? (
            <>
              <div className="truncate text-sm text-text">
                {activeProfile.name}
              </div>
              <div className="truncate font-mono text-xs text-text-faint">
                {activeProfile.protocol} · {activeProfile.address}:
                {activeProfile.port}
              </div>
            </>
          ) : (
            <div className="text-sm text-text-muted">
              Сервер не выбран — добавьте профиль
            </div>
          )}
        </div>
        <span className="flex items-center gap-1.5 rounded-md border border-border bg-surface-2 px-2 py-1 font-mono text-[11px] text-text-muted">
          <Cpu size={12} className="text-accent" />
          {settings.core}
        </span>
      </div>

      {/* Live stats */}
      <div className="grid w-full max-w-md grid-cols-3 gap-3">
        <Stat
          icon={<ArrowDown size={14} className="text-ok" />}
          label="Загрузка"
          value={formatSpeed(status.stats.downloadSpeed)}
          sub={formatBytes(status.stats.download)}
        />
        <Stat
          icon={<ArrowUp size={14} className="text-accent" />}
          label="Отдача"
          value={formatSpeed(status.stats.uploadSpeed)}
          sub={formatBytes(status.stats.upload)}
        />
        <Stat
          icon={<Clock size={14} className="text-text-muted" />}
          label="Время"
          value={formatUptime(status.connectedAt, now)}
          sub={isConnected ? "в сети" : "—"}
        />
      </div>
    </div>
  );
}

function Stat({
  icon,
  label,
  value,
  sub,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub: string;
}) {
  return (
    <div className="rounded-lg border border-border bg-surface px-3 py-3">
      <div className="flex items-center gap-1.5 text-[11px] text-text-faint">
        {icon}
        {label}
      </div>
      <div className="mt-1.5 font-mono text-sm text-text">{value}</div>
      <div className="font-mono text-[11px] text-text-faint">{sub}</div>
    </div>
  );
}
