import {confirmAction} from "./ClientConfirm";
import type {SpeedResult} from "../backend";
import {TestProfileSpeed} from "../backend";
import {push} from "./Toasts";
import { useEffect, useState } from "react";
import {
  CheckCircle2,
  Zap,
  Globe,
  Server,
  Activity,
  Loader2,
  AlertTriangle,
  FileCode2,
  Trash2,
  Gauge,
} from "lucide-react";
import ProfileEditor from "./ProfileEditor";
import type { Profile, Subscription } from "../types";
import { describeChain } from "../proto";
import { plural, HIDDEN, latencyTone } from "../format";
import { stripCountryPrefix } from "../flags";
import { GetCoreIssues, PingProfile } from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import FlagChip from "./FlagChip";

interface Props {
  profiles: Profile[];
  subscriptions: Subscription[];
  selectedGroup: string; // "manual" or a subscription id
  activeId: string;
  connected: boolean;
  hideData: boolean; // demo mode: mask server addresses
  core: string; // the core setting; servers it cannot run are flagged
  onActivate: (id: string) => void;
  onDelete: (id:string)=>Promise<void>;
  onChanged: () => void; // a server was edited
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
  core,
  onActivate,
  onChanged,
  onDelete,
}: Props) {
  // The server whose configuration is open.
  const [editing, setEditing] = useState<Profile | null>(null);
  const [pings, setPings] = useState<PingMap>({});
  const [pinging, setPinging] = useState(false);
  const [speedBusy,setSpeedBusy]=useState(false);
  const speed=async()=>{if(!activeId)return;if(!await confirmAction({title:"Тест скорости профиля",description:"Тест скачает до 25 МБ через выбранный профиль. Продолжить?",confirmLabel:"Запустить тест"}))return;setSpeedBusy(true);try{const id=activeId;const r=await TestProfileSpeed(id);await onChanged();push(`Скорость: ${r.downloadMbps.toFixed(2)} Мбит/с`,"ok",{description:`${r.core} · ${(r.bytes/1e6).toFixed(1)} МБ · ${(r.durationMs/1000).toFixed(1)} с`})}catch(e){push(String(e),"error")}finally{setSpeedBusy(false)}};
  // Servers the chosen core cannot run, with the reason.
  const [issues, setIssues] = useState<Record<string, string>>({});

  useEffect(() => {
    GetCoreIssues().then((m) => setIssues(m ?? {}));
  }, [profiles, core]);

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

  const remove=async(p:Profile)=>{if(!await confirmAction({title:"Удалить сервер?",description:`«${hideData?"Выбранный сервер":p.name}» будет удалён из добавленных вручную.`,confirmLabel:"Удалить",danger:true}))return;try{await onDelete(p.id);push("Сервер удалён","ok")}catch(e){push(String(e),"error")}};
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
    <div className="animate-view flex h-full min-h-0 flex-col">
      <div className="px-6 pt-6">
      <div className="mx-auto mb-5 flex w-full max-w-2xl items-start justify-between">
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
        <button disabled={!activeId||speedBusy} onClick={speed} className="no-drag ml-auto mr-2 rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text-muted disabled:opacity-50" title="Скорость загрузки выбранного профиля, до 25 МБ">{speedBusy?"Тест скорости…":"Скорость"}</button>
        {servers.length > 0 && (
          <button
            onClick={pingAll}
            disabled={pinging}
            title="Проверяет каждый сервер методом из «Настройки → Проверка серверов»"
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
        <div className="min-h-0 flex-1 overflow-y-auto px-6 pb-6 [scrollbar-gutter:stable_both-edges]">
          <div className="mx-auto flex w-full max-w-2xl flex-col gap-2">
          {servers.map((p) => (
            <LocationRow
              key={p.id}
              profile={p}
              active={p.id === activeId}
              connected={connected}
              hideData={hideData}
              ping={pings[p.id]}
              pending={pinging && pings[p.id] === undefined}
              issue={issues[p.id]}
              onActivate={onActivate}
              onEdit={() => setEditing(p)}
              speed={p.speed}
              onDelete={!p.subId?()=>void remove(p):undefined}
            />
          ))}
          </div>
        </div>
      )}
      {editing && (
        <ProfileEditor
          profile={editing}
          hideData={hideData}
          onClose={() => setEditing(null)}
          onSaved={onChanged}
        />
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
  issue,
  onActivate,
  onEdit,
  speed,
  onDelete,
}: {
  profile: Profile;
  active: boolean;
  connected: boolean;
  hideData: boolean;
  ping?: main.PingResult;
  pending: boolean;
  issue?: string;
  onActivate: (id: string) => void;
  onEdit: () => void;
  speed?:SpeedResult;
  onDelete?:()=>void;
}) {
  return (
    <div
      className={`group overflow-hidden rounded-lg border transition ${
        active
          ? "border-accent/50 bg-surface-2"
          : "border-border bg-surface hover:bg-surface-2/60"
      }`}
    >
      <div className="flex min-w-0 items-center gap-3 px-4 py-3">
      <button
        onClick={() => onActivate(p.id)}
        onDoubleClick={onEdit}
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

      <button
        onClick={onEdit}
        title="Конфигурация сервера"
        className="no-drag shrink-0 rounded-md p-1.5 text-text-faint opacity-0 transition hover:bg-surface hover:text-text group-hover:opacity-100"
      >
        <FileCode2 size={15} />
      </button>

      {onDelete&&<button aria-label="Удалить сервер" onClick={onDelete} className="no-drag shrink-0 rounded-md p-1.5 text-text-faint transition hover:bg-danger/10 hover:text-danger disabled:opacity-30" disabled={active&&connected}><Trash2 size={15}/></button>}
      {/* A pinned core that cannot run this server says so up front, rather
          than letting the user find out on Connect. */}
      {issue && (
        <span title={issue} className="shrink-0 text-amber">
          <AlertTriangle size={14} />
        </span>
      )}

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
      {speed&&<div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-border-soft px-4 py-2.5 text-xs text-text-muted"><span className="inline-flex items-center gap-1.5"><Gauge size={13} className="text-accent"/>Загрузка</span><span className="font-mono font-medium text-accent">{speed.downloadMbps.toFixed(2)} Мбит/с</span><span className="ml-auto font-mono text-[10px] text-text-faint">{(speed.bytes/1e6).toFixed(1)} МБ · {(speed.durationMs/1000).toFixed(1)} с · {speed.core}</span></div>}
    </div>
  );
}

// PingBadge shows the measured delay, or n/a when the check failed.
function PingBadge({ result }: { result: main.PingResult }) {
  if (!result.ok) {
    return (
      <span
        title="Проверка не прошла: сервер не ответил за отведённое время"
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
