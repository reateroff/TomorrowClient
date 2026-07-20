import { useState } from "react";
import {
  Plus,
  Trash2,
  CheckCircle2,
  Server,
  Zap,
  Rss,
  RefreshCw,
  Loader2,
} from "lucide-react";
import type { Profile, Subscription } from "../types";
import AddModal from "./AddModal";

interface Props {
  profiles: Profile[];
  subscriptions: Subscription[];
  activeId: string;
  connected: boolean;
  onImportLink: (raw: string) => Promise<void>;
  onAddSub: (name: string, url: string) => Promise<void>;
  onUpdateSub: (id: string) => Promise<void>;
  onDeleteSub: (id: string) => void;
  onDelete: (id: string) => void;
  onActivate: (id: string) => void;
}

// The server list, grouped by subscription with manually-added servers on top.
export default function ProfilesView({
  profiles,
  subscriptions,
  activeId,
  connected,
  onImportLink,
  onAddSub,
  onUpdateSub,
  onDeleteSub,
  onDelete,
  onActivate,
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
    <div className="animate-fade-up flex h-full flex-col p-6">
      <div className="mb-5 flex items-center justify-between">
        <div>
          <h1 className="text-base font-medium text-text">Профили</h1>
          <p className="font-mono text-xs text-text-faint">
            {profiles.length} сервер(ов) · {subscriptions.length} подписк(и)
          </p>
        </div>
        <button
          onClick={() => setAdding(true)}
          className="flex items-center gap-2 rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft"
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
        <div className="flex flex-1 flex-col gap-4 overflow-y-auto pr-1">
          {manual.length > 0 && (
            <div className="flex flex-col gap-2">
              {manual.map((p) => (
                <ProfileRow
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

          {subscriptions.map((sub) => {
            const servers = bySub(sub.id);
            const isUpdating = updating === sub.id;
            return (
              <div key={sub.id} className="flex flex-col gap-2">
                <div className="flex items-center gap-2 px-1">
                  <Rss size={13} className="shrink-0 text-accent" />
                  <span className="truncate text-xs font-medium text-text-muted">
                    {sub.name}
                  </span>
                  <span className="shrink-0 font-mono text-[10px] text-text-faint">
                    {servers.length}
                  </span>
                  <div className="ml-auto flex items-center gap-1">
                    <button
                      onClick={() => runUpdate(sub.id)}
                      disabled={isUpdating}
                      className="rounded-md p-1.5 text-text-faint transition hover:bg-surface-2 hover:text-text disabled:opacity-50"
                      aria-label="Обновить"
                    >
                      {isUpdating ? (
                        <Loader2 size={14} className="animate-spin" />
                      ) : (
                        <RefreshCw size={14} />
                      )}
                    </button>
                    <button
                      onClick={() => onDeleteSub(sub.id)}
                      className="rounded-md p-1.5 text-text-faint transition hover:bg-danger/15 hover:text-danger"
                      aria-label="Удалить подписку"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>
                {servers.map((p) => (
                  <ProfileRow
                    key={p.id}
                    profile={p}
                    active={p.id === activeId}
                    connected={connected}
                    onActivate={onActivate}
                    onDelete={onDelete}
                  />
                ))}
              </div>
            );
          })}
        </div>
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

function ProfileRow({
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
  return (
    <div
      className={`group flex items-center gap-3 rounded-lg border px-4 py-3 transition ${
        active
          ? "border-accent/50 bg-surface-2"
          : "border-border bg-surface hover:border-border hover:bg-surface-2/60"
      }`}
    >
      <button
        onClick={() => onActivate(p.id)}
        className="flex min-w-0 flex-1 items-center gap-3 text-left"
      >
        {active ? (
          <CheckCircle2 size={18} className="shrink-0 text-accent" />
        ) : (
          <Server size={18} className="shrink-0 text-text-faint" />
        )}
        <div className="min-w-0">
          <div className="truncate text-sm text-text">{p.name}</div>
          <div className="truncate font-mono text-xs text-text-faint">
            {p.protocol} · {p.address}:{p.port}
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
