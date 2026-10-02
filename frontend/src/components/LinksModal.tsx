import ModalPortal from "./ModalPortal";
import { useState } from "react";
import { X, Copy, Check, Search, Trash2 } from "lucide-react";
import type { Profile } from "../types";
import { stripCountryPrefix } from "../flags";
import { plural, maskLink } from "../format";
import { push } from "./Toasts";
import {confirmAction} from "./ClientConfirm";

interface Props {
  title: string;
  profiles: Profile[];
  hideData: boolean; // demo mode: show only each link's scheme
  onDelete?: (id:string)=>Promise<void>;
  onClose: () => void;
}

// LinksModal lists the original share links (vless://, vmess://, hysteria2://…)
// of a group's servers. Profile.Raw keeps the link the profile was parsed from,
// so this is the real thing rather than something re-serialised.
export default function LinksModal({
  title,
  profiles,
  hideData,
  onClose,
  onDelete,
}: Props) {
  const [q, setQ] = useState("");
  const [deleting,setDeleting]=useState<string|null>(null);
  const remove=async(p:Profile)=>{if(!onDelete||deleting)return;if(!await confirmAction({title:"Удалить сервер?",description:`«${hideData?"Выбранный сервер":p.name}» будет удалён из добавленных вручную.`,confirmLabel:"Удалить",danger:true}))return;setDeleting(p.id);try{await onDelete(p.id);push("Сервер удалён","ok")}catch(e){push(String(e),"error")}finally{setDeleting(null)}};

  // A profile edited by hand may have no original link; those cannot be listed.
  const withLink = profiles.filter((p) => p.raw);
  const missing = profiles.length - withLink.length;

  const needle = q.trim().toLowerCase();
  const shown = needle
    ? withLink.filter(
        (p) =>
          p.name.toLowerCase().includes(needle) ||
          (p.raw ?? "").toLowerCase().includes(needle)
      )
    : withLink;

  const copyAll = async () => {
    await navigator.clipboard.writeText(shown.map((p) => p.raw).join("\n"));
    push(
      `Скопировано ${shown.length} ${plural(
        shown.length,
        "ссылка",
        "ссылки",
        "ссылок"
      )}`,
      "ok"
    );
  };

  return (
    <ModalPortal onClose={onClose}>
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/60 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-view flex max-h-[75vh] w-full max-w-lg flex-col rounded-xl border border-border bg-surface p-4 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-3 flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="truncate text-sm font-medium text-text">
              Ссылки серверов
            </h2>
            <p className="truncate font-mono text-[11px] text-text-faint">
              {title} · {withLink.length}{" "}
              {plural(withLink.length, "ссылка", "ссылки", "ссылок")}
            </p>
          </div>
          <button
            onClick={onClose}
            className="no-drag shrink-0 text-text-faint transition hover:text-text"
          >
            <X size={16} />
          </button>
        </div>

        {withLink.length === 0 ? (
          <div className="py-8 text-center text-sm text-text-muted">
            У серверов этой подписки не сохранены исходные ссылки.
          </div>
        ) : (
          <>
            <div className="mb-2 flex items-center gap-2">
              <div className="flex min-w-0 flex-1 items-center gap-2 rounded-lg border border-border bg-bg px-3">
                <Search size={14} className="shrink-0 text-text-faint" />
                <input
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  placeholder="Поиск по имени или ссылке…"
                  spellCheck={false}
                  className="min-w-0 flex-1 bg-transparent py-2 font-mono text-xs text-text outline-none placeholder:text-text-faint"
                />
              </div>
              <button
                onClick={copyAll}
                disabled={shown.length === 0}
                className="flex shrink-0 items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text-muted transition hover:bg-surface-2 hover:text-text disabled:opacity-50"
              >
                <Copy size={13} />
                Копировать все
              </button>
            </div>

            <div className="-mr-2 flex min-h-0 flex-1 flex-col gap-1.5 overflow-y-auto pr-1.5">
              {shown.length === 0 ? (
                <div className="py-8 text-center text-sm text-text-muted">
                  Ничего не найдено.
                </div>
              ) : (
                shown.map((p) => (
                  <LinkRow key={p.id} profile={p} hideData={hideData} onDelete={onDelete&&!p.subId?()=>void remove(p):undefined} busy={deleting===p.id}/>
                ))
              )}
            </div>

            {missing > 0 && (
              <p className="mt-2 text-[11px] text-text-faint">
                Ещё {missing}{" "}
                {plural(missing, "сервер", "сервера", "серверов")} без исходной
                ссылки.
              </p>
            )}
          </>
        )}
      </div>
    </div>
    </ModalPortal>
  );
}

function LinkRow({
  profile: p,
  hideData,
  onDelete,
  busy,
}: {
  profile: Profile;
  hideData: boolean;
  onDelete?:()=>void;
  busy:boolean;
}) {
  const [copied, setCopied] = useState(false);

  // Copying stays enabled while masked: the clipboard is not on screen, and
  // getting the link out is the whole reason this dialog exists.
  const copy = async () => {
    await navigator.clipboard.writeText(p.raw ?? "");
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const shownLink = maskLink(hideData, p.raw ?? "");

  return (
    <div className="flex items-center gap-2 rounded-lg border border-border bg-bg px-3 py-2">
      <div className="min-w-0 flex-1">
        <div className="truncate text-xs text-text">
          {stripCountryPrefix(p.name)}
        </div>
        <div
          className="truncate font-mono text-[11px] text-text-faint"
          title={hideData ? undefined : p.raw}
        >
          {shownLink}
        </div>
      </div>
      <button
        onClick={copy}
        title="Копировать ссылку"
        className="no-drag shrink-0 rounded-md p-1.5 text-text-faint transition hover:bg-surface-2 hover:text-text"
      >
        {copied ? <Check size={14} /> : <Copy size={14} />}
      </button>
      {onDelete&&<button aria-label="Удалить сервер" disabled={busy} onClick={onDelete} className="no-drag shrink-0 rounded-md p-1.5 text-text-faint transition hover:bg-danger/10 hover:text-danger disabled:opacity-30"><Trash2 size={14}/></button>}
    </div>
  );
}
