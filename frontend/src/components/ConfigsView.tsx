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
import { countryCodeFor } from "../flags";
import { describeChain } from "../proto";
import { PingProfile } from "../../wailsjs/go/main/App";
import * as Flags from "country-flag-icons/react/3x2";

interface Props {
  profiles: Profile[];
  subscriptions: Subscription[];
  selectedGroup: string; // "manual" or a subscription id
  activeId: string;
  connected: boolean;
  onActivate: (id: string) => void;
}

// Latency in ms per profile id: >=0 reachable, -1 unreachable, undefined = not
// yet tested.
type PingMap = Record<string, number>;

// Shows the locations (servers) of the group picked on the Profiles tab, with a
// ping button that TCP-tests every server's reachability.
export default function ConfigsView({
  profiles,
  subscriptions,
  selectedGroup,
  activeId,
  connected,
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
    await Promise.all(
      servers.map(async (p) => {
        const ms = await PingProfile(p.id);
        setPings((prev) => ({ ...prev, [p.id]: ms }));
      })
    );
    setPinging(false);
  };

  return (
    <div className="animate-fade-up flex h-full flex-col p-6">
      <div className="mb-5 flex items-start justify-between">
        <div>
          <h1 className="text-base font-medium text-text">Конфигурации</h1>
          <p className="font-mono text-xs text-text-faint">
            {groupName
              ? `${groupName} · ${servers.length} локац(ий)`
              : "выбор локации"}
          </p>
        </div>
        {servers.length > 0 && (
          <button
            onClick={pingAll}
            disabled={pinging}
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
        <div className="flex flex-1 flex-col gap-2 overflow-y-auto pr-1">
          {servers.map((p) => (
            <LocationRow
              key={p.id}
              profile={p}
              active={p.id === activeId}
              connected={connected}
              ping={pings[p.id]}
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
  ping,
  onActivate,
}: {
  profile: Profile;
  active: boolean;
  connected: boolean;
  ping?: number;
  onActivate: (id: string) => void;
}) {
  const cc = countryCodeFor(p.name);
  const Flag = cc ? (Flags as Record<string, React.ComponentType<any>>)[cc] : null;
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
          {Flag ? (
            <Flag className="h-full w-full object-cover" />
          ) : (
            <Globe size={16} className="text-text-faint" />
          )}
        </span>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm text-text">{p.name}</span>
            {active && (
              <CheckCircle2 size={14} className="shrink-0 text-accent" />
            )}
          </div>
          <div className="truncate font-mono text-xs text-text-faint">
            {describeChain(p)} · {p.address}:{p.port}
          </div>
        </div>
      </button>

      {ping !== undefined && <PingBadge ms={ping} />}

      {active && connected && (
        <span className="flex items-center gap-1 rounded-md bg-ok/15 px-2 py-1 font-mono text-[10px] text-ok">
          <Zap size={11} />
          активен
        </span>
      )}
    </div>
  );
}

// PingBadge colours the latency: green fast, amber slow, red unreachable.
function PingBadge({ ms }: { ms: number }) {
  if (ms < 0) {
    return (
      <span className="shrink-0 rounded-md bg-danger/15 px-2 py-1 font-mono text-[10px] text-danger">
        —
      </span>
    );
  }
  const tone =
    ms < 150 ? "text-ok bg-ok/15" : ms < 400 ? "text-amber bg-amber/15" : "text-danger bg-danger/15";
  return (
    <span className={`shrink-0 rounded-md px-2 py-1 font-mono text-[10px] ${tone}`}>
      {ms} мс
    </span>
  );
}
