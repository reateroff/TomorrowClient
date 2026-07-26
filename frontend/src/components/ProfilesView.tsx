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
  ListTree,
} from "lucide-react";
import type { Profile, Subscription } from "../types";
import { formatTraffic, formatExpiry, plural } from "../format";
import AddModal from "./AddModal";
import LinksModal from "./LinksModal";
import Menu from "./Menu";
import type { MenuItem } from "./Menu";
import { push } from "./Toasts";

interface Props {
  profiles: Profile[];
  subscriptions: Subscription[];
  selectedGroup: string;
  hideData: boolean; // demo mode: mask the share links shown in the links dialog
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
  hideData,
  onImportLink,
  onAddSub,
  onUpdateSub,
  onDeleteSub,
  onSelectGroup,
}: Props) {
  const [adding, setAdding] = useState(false);
  const [updating, setUpdating] = useState<string | null>(null);
  // Which group's share links are open: a subscription id, "manual", or null.
  const [linksOf, setLinksOf] = useState<string | null>(null);

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

  const copyUrl = async (url: string) => {
    await navigator.clipboard.writeText(url);
    push("Ссылка подписки скопирована", "ok");
  };

  const linksTitle =
    linksOf === "manual"
      ? "Добавленные вручную"
      : subscriptions.find((s) => s.id === linksOf)?.name ?? "";
  const linksProfiles =
    linksOf === "manual" ? manual : linksOf ? bySub(linksOf) : [];

  const empty = profiles.length === 0 && subscriptions.length === 0;

  return (
    <div className="animate-view flex h-full min-h-0 flex-col">
      {/* Header and list centre columns of the same width; the scroller itself
          spans the window so its bar still sits at the edge. */}
      <div className="px-6 pt-6">
        <div className="mx-auto mb-5 flex w-full max-w-2xl items-center justify-between">
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
        <div className="min-h-0 flex-1 overflow-y-auto px-6 pb-6 [scrollbar-gutter:stable_both-edges]">
          <p className="mx-auto mb-3 w-full max-w-2xl text-xs text-text-faint">
            Выберите профиль — его локации появятся во вкладке «Конфигурации».
          </p>
          <div className="mx-auto flex w-full max-w-2xl flex-col gap-2">
            {manual.length > 0 && (
              <GroupCard
                icon={<Link2 size={16} />}
                name="Добавленные вручную"
                count={manual.length}
                selected={selectedGroup === "manual"}
                onSelect={() => onSelectGroup("manual")}
                menu={[
                  {
                    label: "Ссылки серверов",
                    icon: <ListTree size={14} />,
                    onClick: () => setLinksOf("manual"),
                  },
                ]}
              />
            )}
            {subscriptions.map((sub) => (
              <GroupCard
                key={sub.id}
                icon={<Rss size={16} />}
                name={sub.name}
                count={bySub(sub.id).length}
                sub={sub}
                selected={selectedGroup === sub.id}
                onSelect={() => onSelectGroup(sub.id)}
                updating={updating === sub.id}
                menu={[
                  {
                    label: "Обновить",
                    icon: <RefreshCw size={14} />,
                    onClick: () => runUpdate(sub.id),
                  },
                  {
                    label: "Копировать ссылку",
                    icon: <Link2 size={14} />,
                    onClick: () => copyUrl(sub.url),
                  },
                  {
                    label: "Ссылки серверов",
                    icon: <ListTree size={14} />,
                    onClick: () => setLinksOf(sub.id),
                  },
                  {
                    label: "Удалить",
                    icon: <Trash2 size={14} />,
                    onClick: () => onDeleteSub(sub.id),
                    danger: true,
                  },
                ]}
              />
            ))}
          </div>
        </div>
      )}

      {adding && (
        <AddModal
          onClose={() => setAdding(false)}
          onImportLink={onImportLink}
          onAddSub={onAddSub}
        />
      )}

      {linksOf && (
        <LinksModal
          title={linksTitle}
          profiles={linksProfiles}
          hideData={hideData}
          onClose={() => setLinksOf(null)}
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
  count,
  sub,
  selected,
  onSelect,
  updating,
  menu,
}: {
  icon: React.ReactNode;
  name: string;
  count: number;
  sub?: Subscription;
  selected: boolean;
  onSelect: () => void;
  updating?: boolean;
  menu?: MenuItem[];
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
          {/* The subscription's own name only — the URL host used to sit next
              to it, which was noise when the name was already derived from it. */}
          <span className="block truncate text-sm text-text">{name}</span>
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

      {/* While a refresh is in flight the spinner replaces the menu, so the row
          still shows what it is doing. */}
      {updating ? (
        <Loader2 size={15} className="shrink-0 animate-spin text-text-faint" />
      ) : (
        menu && <Menu items={menu} />
      )}
    </div>
  );
}
