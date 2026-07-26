import { useEffect, useState } from "react";
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  Timer,
  Gauge,
  ChevronRight,
} from "lucide-react";
import type { Status, Profile } from "../types";
import {
  formatBytes,
  formatSpeed,
  formatUptime,
  latencyTone,
} from "../format";
import { stripCountryPrefix } from "../flags";
import { PingLatency } from "../../wailsjs/go/main/App";

// How often the live ping refreshes while connected.
const PING_INTERVAL_MS = 5000;
import FlagChip from "./FlagChip";
import logoMark from "../assets/logo-mark.png";

interface Props {
  status: Status;
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

  // Tick every second so the uptime clock stays live — only while connected,
  // since that clock is the only thing reading it and an idle screen has no
  // reason to re-render once a second.
  useEffect(() => {
    if (!isConnected) return;
    setNow(Date.now());
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [isConnected]);

  // Live ping: only while connected, reset on drop or when the server changes.
  // This is the ICMP-only call, a few 32-byte packets, so refreshing it every
  // few seconds costs nothing measurable. The next run is scheduled after the
  // previous one returns rather than on a fixed interval, so a server that
  // makes us wait out the echo timeout cannot pile requests up.
  const pid = activeProfile?.id;
  useEffect(() => {
    if (!isConnected || !pid) {
      setPing(null);
      return;
    }
    let alive = true;
    let timer: number | undefined;

    const run = async () => {
      try {
        const ms = await PingLatency(pid);
        if (alive) setPing(ms);
      } catch {
        if (alive) setPing(-1);
      }
      if (alive) timer = window.setTimeout(run, PING_INTERVAL_MS);
    };
    run();

    return () => {
      alive = false;
      if (timer) clearTimeout(timer);
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

  // Filled disc styling: accent when connected, surface otherwise. The mark on
  // top carries no colour of its own from these, so only fill and border here.
  const disc = isConnected
    ? "bg-accent shadow-xl shadow-accent/25"
    : state === "error"
    ? "bg-surface border border-danger/40"
    : "bg-surface border border-border group-hover:border-text-faint";

  return (
    <div className="animate-view flex h-full flex-col p-6">
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
            {/* brightness-0 invert repaints the mark pure white while keeping
                its alpha, because once the disc fills with the accent the logo's
                own periwinkle sits right on top of it — with the default indigo
                accent the two are nearly the same colour. Off, it keeps its own
                colour and just dims until hovered. */}
            <img
              src={logoMark}
              alt=""
              draggable={false}
              className={`h-24 w-24 transition ${
                isConnected
                  ? "brightness-0 invert"
                  : "opacity-70 group-hover:opacity-100"
              }`}
            />
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

      {/* Stats only exist once there is a tunnel. While disconnected the screen
          is just the button, and the figures fade in on connect rather than
          sitting there as a row of dashes and zeroes. */}
      {isConnected && (
        <div className="animate-view mx-auto grid w-full max-w-sm grid-cols-4 gap-2">
          <Stat
            icon={<Gauge size={12} className="text-text-faint" />}
            label="Пинг"
            value={pingLabel(ping)}
            valueCls={pingTone(ping)}
          />
          <Stat
            icon={<Timer size={12} className="text-text-faint" />}
            label="Время"
            value={formatUptime(status.connectedAt, now)}
          />
          <Stat
            icon={<ArrowDownToLine size={12} className="text-text-faint" />}
            label="Скачано"
            value={formatBytes(status.stats.download)}
            valueCls="text-ok"
            sub={formatSpeed(status.stats.downloadSpeed)}
          />
          {/* Upload deliberately uses the fixed --color-up rather than the
              accent: the accent is user-chosen, so it clashed with the green
              download figure whenever someone picked an unrelated colour. */}
          <Stat
            icon={<ArrowUpFromLine size={12} className="text-text-faint" />}
            label="Отдано"
            value={formatBytes(status.stats.upload)}
            valueCls="text-up"
            sub={formatSpeed(status.stats.uploadSpeed)}
          />
        </div>
      )}
    </div>
  );
}

// A dash means the server keeps quiet about ICMP, which plenty of them do while
// working fine — not an error, so it stays muted rather than red.
function pingLabel(ms: number | null): string {
  return ms == null || ms < 0 ? "—" : `${ms} мс`;
}

function pingTone(ms: number | null): string {
  if (ms == null) return "text-text";
  if (ms < 0) return "text-text-faint";
  return {
    ok: "text-ok",
    amber: "text-amber",
    danger: "text-danger",
  }[latencyTone(ms)];
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
