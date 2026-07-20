import { useState } from "react";
import { X, Link2, Rss, Loader2 } from "lucide-react";

interface Props {
  onClose: () => void;
  onImportLink: (raw: string) => Promise<void>;
  onAddSub: (name: string, url: string) => Promise<void>;
}

type Tab = "link" | "sub";

// Modal to add servers either from a single share link or a subscription URL.
export default function AddModal({ onClose, onImportLink, onAddSub }: Props) {
  const [tab, setTab] = useState<Tab>("link");
  const [raw, setRaw] = useState("");
  const [subName, setSubName] = useState("");
  const [subUrl, setSubUrl] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const submit = async () => {
    setErr("");
    setBusy(true);
    try {
      if (tab === "link") {
        if (!raw.trim()) return;
        await onImportLink(raw.trim());
      } else {
        if (!subUrl.trim()) return;
        await onAddSub(subName.trim(), subUrl.trim());
      }
      onClose();
    } catch (e: any) {
      setErr(String(e?.message ?? e));
    } finally {
      setBusy(false);
    }
  };

  const canSubmit = tab === "link" ? !!raw.trim() : !!subUrl.trim();

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-fade-up w-full max-w-md rounded-xl border border-border bg-surface p-5 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-medium text-text">Добавить серверы</h2>
          <button
            onClick={onClose}
            className="text-text-faint transition hover:text-text"
          >
            <X size={16} />
          </button>
        </div>

        {/* Tabs */}
        <div className="mb-4 flex gap-1 rounded-lg border border-border bg-bg p-1">
          <TabButton
            active={tab === "link"}
            onClick={() => setTab("link")}
            icon={<Link2 size={14} />}
            label="По ссылке"
          />
          <TabButton
            active={tab === "sub"}
            onClick={() => setTab("sub")}
            icon={<Rss size={14} />}
            label="Подписка"
          />
        </div>

        {tab === "link" ? (
          <textarea
            value={raw}
            onChange={(e) => setRaw(e.target.value)}
            placeholder="vless://…  vmess://…  trojan://…  ss://…"
            rows={4}
            autoFocus
            className="w-full resize-none rounded-lg border border-border bg-bg px-3 py-2.5 font-mono text-xs text-text outline-none transition placeholder:text-text-faint focus:border-accent/60"
          />
        ) : (
          <div className="flex flex-col gap-2.5">
            <input
              value={subUrl}
              onChange={(e) => setSubUrl(e.target.value)}
              placeholder="https://example.com/sub"
              autoFocus
              className="w-full rounded-lg border border-border bg-bg px-3 py-2.5 font-mono text-xs text-text outline-none transition placeholder:text-text-faint focus:border-accent/60"
            />
            <input
              value={subName}
              onChange={(e) => setSubName(e.target.value)}
              placeholder="Название (необязательно)"
              className="w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60"
            />
            <p className="text-xs text-text-faint">
              Загрузит список серверов и будет обновляться по кнопке.
            </p>
          </div>
        )}

        {err && <div className="mt-2 text-xs text-danger">{err}</div>}

        <div className="mt-4 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-2 text-sm text-text-muted transition hover:bg-surface-2"
          >
            Отмена
          </button>
          <button
            onClick={submit}
            disabled={busy || !canSubmit}
            className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft disabled:opacity-50"
          >
            {busy && <Loader2 size={14} className="animate-spin" />}
            {tab === "link" ? "Добавить" : "Загрузить"}
          </button>
        </div>
      </div>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition ${
        active ? "bg-surface-2 text-text" : "text-text-muted hover:text-text"
      }`}
    >
      {icon}
      {label}
    </button>
  );
}
