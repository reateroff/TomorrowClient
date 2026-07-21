import {
  Trash2,
  CheckCircle2,
  Zap,
  Globe,
  Server,
} from "lucide-react";
import type { Profile, Subscription } from "../types";
import { countryCodeFor } from "../flags";
import { describeChain } from "../proto";
import * as Flags from "country-flag-icons/react/3x2";

interface Props {
  profiles: Profile[];
  subscriptions: Subscription[];
  selectedGroup: string; // "manual" or a subscription id
  activeId: string;
  connected: boolean;
  onSelectGroup: (groupId: string) => void;
  onDelete: (id: string) => void;
  onActivate: (id: string) => void;
}

// Shows the locations (servers) of the group picked on the Profiles tab, with a
// small group switcher on top so you can jump between subscriptions here too.
export default function ConfigsView({
  profiles,
  subscriptions,
  selectedGroup,
  activeId,
  connected,
  onSelectGroup,
  onDelete,
  onActivate,
}: Props) {
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

  return (
    <div className="animate-fade-up flex h-full flex-col p-6">
      <div className="mb-5">
        <h1 className="text-base font-medium text-text">Конфигурации</h1>
        <p className="font-mono text-xs text-text-faint">
          {groupName ? `${groupName} · ${servers.length} локац(ий)` : "выбор локации"}
        </p>
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
      ) : (
        <>
          {/* Group switcher */}
          {groups.length > 1 && (
            <div className="mb-4 flex flex-wrap gap-2">
              {groups.map((g) => (
                <button
                  key={g.id}
                  onClick={() => onSelectGroup(g.id)}
                  className={`no-drag rounded-lg border px-3 py-1.5 text-xs transition ${
                    g.id === groupId
                      ? "border-accent/60 bg-surface-2 text-text"
                      : "border-border bg-surface text-text-muted hover:bg-surface-2/60"
                  }`}
                >
                  {g.name}
                </button>
              ))}
            </div>
          )}

          {servers.length === 0 ? (
            <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
              <Server size={32} className="text-text-faint" />
              <p className="text-sm text-text-muted">В этой группе нет серверов.</p>
            </div>
          ) : (
            <div className="flex flex-1 flex-col gap-2 overflow-y-auto pr-1">
              {servers.map((p) => (
                <LocationRow
                  key={p.id}
                  profile={p}
                  active={p.id === activeId}
                  connected={connected}
                  onActivate={onActivate}
                  onDelete={onDelete}
                />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

// A single server / location row with an SVG country flag on the left.
function LocationRow({
  profile: p,
  active,
  connected,
  onActivate,
  onDelete,
}: {
  profile: Profile;
  active: boolean;
  connected: boolean;
  onActivate: (id: string) => void;
  onDelete: (id: string) => void;
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

      {active && connected && (
        <span className="flex items-center gap-1 rounded-md bg-ok/15 px-2 py-1 font-mono text-[10px] text-ok">
          <Zap size={11} />
          активен
        </span>
      )}

      <button
        onClick={() => onDelete(p.id)}
        className="no-drag rounded-md p-1.5 text-text-faint opacity-0 transition hover:bg-danger/15 hover:text-danger group-hover:opacity-100"
        aria-label="Удалить"
      >
        <Trash2 size={15} />
      </button>
    </div>
  );
}
