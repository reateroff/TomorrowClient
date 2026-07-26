import { useState } from "react";
import {
  Plus,
  Trash2,
  Server,
  Rss,
  RefreshCw,
  Loader2,
  Link2,
  Check,
  Gauge,
  CalendarClock,
} from "lucide-react";
import type { Profile, Subscription } from "../types";
import { formatTraffic, formatExpiry, subDomain, plural } from "../format";
import AddModal from "./AddModal";

interface Props {
  profiles: Profile[];
  subscriptions: Subscription[];
  selectedGroup: string;
  onImportLink: (raw: string) => Promise<void>;
  onAddSub: (name: string, url: string) => Promise<void>;
  onUpdateSub: (id: string) => Promise<void>;
  onDeleteSub: (id: string) => void;
  // Pick a group as the active one; the Configs tab shows its servers.
  onSelectGroup: (groupId: string) => void;
}

// Lists the added profiles as groups: manually-added servers and each VPN
// subscription. Selecting a group marks it active; its locations are chosen
// separately in the Configs tab.
export default function ProfilesView({
  profiles,
  subscriptions,
  selectedGroup,
  onImportLink,
  onAddSub,
  onUpdateSub,
  onDeleteSub,
  onSelectGroup,
}: Props) {
  const [adding, setAdding] = useState(false);
  const [updating, setUpdating] = useState<string | null>(null);

  const manual = profiles.filter((p) => !p.subId);
  const bySub = (id: string) => profiles.filter((p) => p.subId === id);

  const runUpdate = async (id: string) => {
    setUpdating(id);
    try {
      await onUpdateSub(id);
    } finally {
      setUpdating(null);
    }
  };

  const empty = profiles.length === 0 && subscriptions.length === 0;

  return (
    <div className="animate-view flex h-full flex-col p-6">
      <div className="mb-5 flex items-center justify-between">
        <div>
          <h1 className="text-base font-medium text-text">Профили</h1>
          <p className="font-mono text-xs text-text-faint">
            {profiles.length}{" "}
            {plural(profiles.length, "сервер", "сервера", "серверов")} ·{" "}
            {subscriptions.length}{" "}
            {plural(subscriptions.length, "подписка", "подписки", "подписок")}
          </p>
        </div>
        <button
          onClick={() => setAdding(true)}
          className="no-drag flex items-center gap-2 rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft"
        >
          <Plus size={16} />
          Добавить
        </button>
      </div>

      {empty ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
          <Server size={36} className="text-text-faint" />
          <p className="text-sm text-text-muted">
            Пока нет серверов.
            <br />
            Импортируйте по ссылке или добавьте подписку.
          </p>
        </div>
      ) : (
        <>
          <p className="mb-3 text-xs text-text-faint">
            Выберите профиль — его локации появятся во вкладке «Конфигурации».
          </p>
          <div className="-mr-6 flex flex-1 flex-col gap-2 overflow-y-auto pr-1.5">
            {manual.length > 0 && (
              <GroupCard
                icon={<Link2 size={16} />}
                name="Добавленные вручную"
                count={manual.length}
                selected={selectedGroup === "manual"}
                onSelect={() => onSelectGroup("manual")}
              />
            )}
            {subscriptions.map((sub) => (
              <GroupCard
                key={sub.id}
                icon={<Rss size={16} />}
                name={sub.name}
                domain={subDomain(sub.url)}
                count={bySub(sub.id).length}
                sub={sub}
                selected={selectedGroup === sub.id}
                onSelect={() => onSelectGroup(sub.id)}
                updating={updating === sub.id}
                onUpdate={() => runUpdate(sub.id)}
                onDelete={() => onDeleteSub(sub.id)}
              />
            ))}
          </div>
        </>
      )}

      {adding && (
        <AddModal
          onClose={() => setAdding(false)}
          onImportLink={onImportLink}
          onAddSub={onAddSub}
        />
      )}
    </div>
  );
}

// A subscription (or manual group) card. Clicking it selects the group; the
// selected one shows an "активен" badge. Subscriptions also show their domain,
// used/total traffic and expiry (∞ when the provider reports no limit).
function GroupCard({
  icon,
  name,
  domain,
  count,
  sub,
  selected,
  onSelect,
  updating,
  onUpdate,
  onDelete,
}: {
  icon: React.ReactNode;
  name: string;
  domain?: string;
  count: number;
  sub?: Subscription;
  selected: boolean;
  onSelect: () => void;
  updating?: boolean;
  onUpdate?: () => void;
  onDelete?: () => void;
}) {
  return (
    <div
      className={`group flex items-center gap-3 rounded-lg border px-4 py-3 transition ${
        selected
          ? "border-accent/50 bg-surface-2"
          : "border-border bg-surface hover:bg-surface-2/60"
      }`}
    >
      <button
        onClick={onSelect}
        className="no-drag flex min-w-0 flex-1 items-center gap-3 text-left"
      >
        <span
          className={`grid h-9 w-9 shrink-0 place-items-center rounded-lg ${
            selected ? "bg-accent/15 text-accent" : "bg-surface-2 text-text-muted"
          }`}
        >
          {icon}
        </span>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm text-text">{name}</span>
            {domain && (
              <span className="truncate font-mono text-xs text-text-faint">
                {domain}
              </span>
            )}
          </div>
          <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 font-mono text-xs text-text-faint">
            <span>
              {count} {plural(count, "сервер", "сервера", "серверов")}
            </span>
            {sub && (
              <>
                <span className="inline-flex items-center gap-1">
                  <Gauge size={11} />
                  {formatTraffic(sub.upload, sub.download, sub.total)}
                </span>
                <span className="inline-flex items-center gap-1">
                  <CalendarClock size={11} />
                  {formatExpiry(sub.expire)}
                </span>
              </>
            )}
          </div>
        </div>
      </button>

      {selected && (
        <span className="flex items-center gap-1 rounded-md bg-accent/15 px-2 py-1 font-mono text-[10px] text-accent">
          <Check size={11} />
          активен
        </span>
      )}

      {onUpdate && (
        <button
          onClick={onUpdate}
          disabled={updating}
          className="no-drag rounded-md p-1.5 text-text-faint transition hover:bg-surface-2 hover:text-text disabled:opacity-50"
          aria-label="Обновить"
        >
          {updating ? (
            <Loader2 size={15} className="animate-spin" />
          ) : (
            <RefreshCw size={15} />
          )}
        </button>
      )}
      {onDelete && (
        <button
          onClick={onDelete}
          className="no-drag rounded-md p-1.5 text-text-faint transition hover:bg-danger/15 hover:text-danger"
          aria-label="Удалить подписку"
        >
          <Trash2 size={15} />
        </button>
      )}
    </div>
  );
}
