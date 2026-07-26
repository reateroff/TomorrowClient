import { useState } from "react";
import {
  CheckCircle2,
  Zap,
  Globe,
  Server,
  Activity,
  Loader2,
} from "lucide-react";
import type { Profile, Subscription } from "../types";
import { describeChain } from "../proto";
import { plural, HIDDEN, latencyTone } from "../format";
import { stripCountryPrefix } from "../flags";
import { PingProfile } from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import FlagChip from "./FlagChip";

interface Props {
  profiles: Profile[];
  subscriptions: Subscription[];
  selectedGroup: string; // "manual" or a subscription id
  activeId: string;
  connected: boolean;
  hideData: boolean; // demo mode: mask server addresses
  onActivate: (id: string) => void;
}

// Ping result per profile id; undefined means "not tested yet".
type PingMap = Record<string, main.PingResult>;

// Shows the locations (servers) of the group picked on the Profiles tab, with a
// ping button that TCP-tests every server's reachability.
export default function ConfigsView({
  profiles,
  subscriptions,
  selectedGroup,
  activeId,
  connected,
  hideData,
  onActivate,
}: Props) {
  const [pings, setPings] = useState<PingMap>({});
  const [pinging, setPinging] = useState(false);

  const manual = profiles.filter((p) => !p.subId);
  const bySub = (id: string) => profiles.filter((p) => p.subId === id);

  const groups: { id: string; name: string }[] = [
    ...(manual.length > 0 ? [{ id: "manual", name: "Вручную" }] : []),
    ...subscriptions.map((s) => ({ id: s.id, name: s.name })),
  ];

  // Resolve the effective group: the selected one, else the group of the active
  // server, else the first group.
  const groupId =
    (groups.some((g) => g.id === selectedGroup) && selectedGroup) ||
    profiles.find((p) => p.id === activeId)?.subId ||
    (manual.some((p) => p.id === activeId) ? "manual" : "") ||
    groups[0]?.id ||
    "";

  const servers = groupId === "manual" ? manual : bySub(groupId);
  const groupName = groups.find((g) => g.id === groupId)?.name ?? "";

  const empty = profiles.length === 0;

  // pingAll tests every server in the current group concurrently and stores the
  // latency (or -1) per profile as results arrive.
  const pingAll = async () => {
    if (pinging || servers.length === 0) return;
    setPinging(true);
    // Clear previous results for this group so stale values don't linger.
    setPings((prev) => {
      const next = { ...prev };
      servers.forEach((p) => delete next[p.id]);
      return next;
    });
    // Each probe swallows its own failure and try/finally releases the button:
    // a single rejected call used to reject the whole Promise.all, skip
    // setPinging(false) and leave the spinner turning forever.
    try {
      await Promise.all(
        servers.map(async (p) => {
          let res: main.PingResult = { latencyMs: -1, ok: false };
          try {
            res = (await PingProfile(p.id)) as main.PingResult;
          } catch {
            /* a failed call reads the same as a dead server */
          }
          setPings((prev) => ({ ...prev, [p.id]: res }));
        })
      );
    } finally {
      setPinging(false);
    }
  };

  return (
    <div className="animate-view flex h-full flex-col p-6">
      <div className="mb-5 flex items-start justify-between">
        <div>
          <h1 className="text-base font-medium text-text">Конфигурации</h1>
          <p className="font-mono text-xs text-text-faint">
            {groupName
              ? `${groupName} · ${servers.length} ${plural(
                  servers.length,
                  "локация",
                  "локации",
                  "локаций"
                )}`
              : "выбор локации"}
          </p>
        </div>
        {servers.length > 0 && (
          <button
            onClick={pingAll}
            disabled={pinging}
            title="Отправляет реальный запрос через каждый сервер — измеряет то, что действительно работает, а не просто открытый порт"
            className="no-drag flex items-center gap-2 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text-muted transition hover:bg-surface-2 hover:text-text disabled:opacity-50"
          >
            {pinging ? (
              <Loader2 size={15} className="animate-spin" />
            ) : (
              <Activity size={15} />
            )}
            Пинг
          </button>
        )}
      </div>

      {empty ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
          <Globe size={36} className="text-text-faint" />
          <p className="text-sm text-text-muted">
            Нет конфигураций.
            <br />
            Сначала добавьте профиль во вкладке «Профили».
          </p>
        </div>
      ) : servers.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
          <Server size={32} className="text-text-faint" />
          <p className="text-sm text-text-muted">
            В этой группе нет серверов.
            <br />
            Выберите профиль во вкладке «Профили».
          </p>
        </div>
      ) : (
        <div className="-mr-6 flex flex-1 flex-col gap-2 overflow-y-auto pr-1.5">
          {servers.map((p) => (
            <LocationRow
              key={p.id}
              profile={p}
              active={p.id === activeId}
              connected={connected}
              hideData={hideData}
              ping={pings[p.id]}
              pending={pinging && pings[p.id] === undefined}
              onActivate={onActivate}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// A single server / location row with an SVG country flag on the left and an
// optional latency badge on the right.
function LocationRow({
  profile: p,
  active,
  connected,
  hideData,
  ping,
  pending,
  onActivate,
}: {
  profile: Profile;
  active: boolean;
  connected: boolean;
  hideData: boolean;
  ping?: main.PingResult;
  pending: boolean;
  onActivate: (id: string) => void;
}) {
  return (
    <div
      className={`group flex items-center gap-3 rounded-lg border px-4 py-3 transition ${
        active
          ? "border-accent/50 bg-surface-2"
          : "border-border bg-surface hover:bg-surface-2/60"
      }`}
    >
      <button
        onClick={() => onActivate(p.id)}
        className="no-drag flex min-w-0 flex-1 items-center gap-3 text-left"
      >
        {/* Flag chip */}
        <span className="grid h-9 w-12 shrink-0 place-items-center overflow-hidden rounded-md border border-border bg-bg">
          <FlagChip name={p.name} />
        </span>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm text-text">
              {stripCountryPrefix(p.name)}
            </span>
            {active && (
              <CheckCircle2 size={14} className="shrink-0 text-accent" />
            )}
          </div>
          <div className="truncate font-mono text-xs text-text-faint">
            {describeChain(p)} ·{" "}
            {hideData ? HIDDEN : `${p.address}:${p.port}`}
          </div>
        </div>
      </button>

      {/* A probe takes real time now, and the backend runs only a few at once,
          so rows waiting their turn say so instead of looking untested. */}
      {pending ? (
        <Loader2 size={13} className="shrink-0 animate-spin text-text-faint" />
      ) : (
        ping !== undefined && <PingBadge result={ping} />
      )}

      {active && connected && (
        <span className="flex items-center gap-1 rounded-md bg-ok/15 px-2 py-1 font-mono text-[10px] text-ok">
          <Zap size={11} />
          активен
        </span>
      )}
    </div>
  );
}

// PingBadge shows the ICMP round trip, or n/a when the profile itself does not
// work. The two states are distinct on purpose: a server can answer ICMP and
// still be unusable, and it can work perfectly while filtering ICMP.
function PingBadge({ result }: { result: main.PingResult }) {
  if (!result.ok) {
    return (
      <span
        title="Через этот сервер не проходит запрос — профиль не работает"
        className="shrink-0 rounded-md bg-danger/15 px-2 py-1 font-mono text-[10px] text-danger"
      >
        n/a
      </span>
    );
  }
  if (result.latencyMs < 0) {
    return (
      <span
        title="Сервер работает, но не отвечает на ICMP — задержку измерить нечем"
        className="shrink-0 rounded-md bg-surface-2 px-2 py-1 font-mono text-[10px] text-text-faint"
      >
        —
      </span>
    );
  }
  const tone = {
    ok: "text-ok bg-ok/15",
    amber: "text-amber bg-amber/15",
    danger: "text-danger bg-danger/15",
  }[latencyTone(result.latencyMs)];
  return (
    <span className={`shrink-0 rounded-md px-2 py-1 font-mono text-[10px] ${tone}`}>
      {result.latencyMs} мс
    </span>
  );
}
