import { useMemo, useState } from "react";
import { X, Loader2, Link2, Rss } from "lucide-react";

interface Props {
  onClose: () => void;
  onImportLink: (raw: string) => Promise<void>;
  onAddSub: (name: string, url: string) => Promise<void>;
}

// Kind of the pasted text, decided automatically from its content.
type Kind = "link" | "sub" | "empty" | "unknown";

const LINK_SCHEMES = [
  "vless://",
  "vmess://",
  "trojan://",
  "ss://",
  "hysteria2://",
  "hy2://",
  "hysteria://",
  "hy://",
  "tuic://",
];

// detectKind decides whether the input is a share link or a subscription URL.
// Share-link schemes win; anything http(s) is treated as a subscription.
function detectKind(text: string): Kind {
  const t = text.trim().toLowerCase();
  if (!t) return "empty";
  if (LINK_SCHEMES.some((s) => t.startsWith(s))) return "link";
  if (t.startsWith("http://") || t.startsWith("https://")) return "sub";
  return "unknown";
}

// Modal to add servers. A single field auto-detects a share link vs a
// subscription URL, so the user never picks a type manually. An optional name
// is used only for subscriptions.
export default function AddModal({ onClose, onImportLink, onAddSub }: Props) {
  const [text, setText] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const kind = useMemo(() => detectKind(text), [text]);
  const canSubmit = kind === "link" || kind === "sub";

  const submit = async () => {
    setErr("");
    const value = text.trim();
    if (kind === "empty") return;
    if (kind === "unknown") {
      setErr("Не распознан формат: вставьте ссылку vless:// … или адрес подписки https://…");
      return;
    }
    setBusy(true);
    try {
      if (kind === "link") {
        await onImportLink(value);
      } else {
        await onAddSub(name.trim(), value);
      }
      onClose();
    } catch (e: any) {
      setErr(String(e?.message ?? e));
    } finally {
      setBusy(false);
    }
  };

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
            className="no-drag text-text-faint transition hover:text-text"
          >
            <X size={16} />
          </button>
        </div>

        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Вставьте ссылку (vless://, vmess://, hysteria2://, …) или адрес подписки https://…"
          rows={4}
          autoFocus
          className="w-full resize-none rounded-lg border border-border bg-bg px-3 py-2.5 font-mono text-xs text-text outline-none transition placeholder:text-text-faint focus:border-accent/60"
        />

        {/* Live detection hint */}
        <div className="mt-2 h-4 text-xs">
          {kind === "link" && (
            <span className="inline-flex items-center gap-1.5 text-ok">
              <Link2 size={12} /> Определено: ссылка на сервер
            </span>
          )}
          {kind === "sub" && (
            <span className="inline-flex items-center gap-1.5 text-accent">
              <Rss size={12} /> Определено: подписка
            </span>
          )}
        </div>

        {/* Optional name only makes sense for a subscription. */}
        {kind === "sub" && (
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Название подписки (необязательно)"
            className="mt-2 w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60"
          />
        )}

        {err && <div className="mt-2 text-xs text-danger">{err}</div>}

        <div className="mt-4 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="no-drag rounded-lg border border-border px-3 py-2 text-sm text-text-muted transition hover:bg-surface-2"
          >
            Отмена
          </button>
          <button
            onClick={submit}
            disabled={busy || !canSubmit}
            className="no-drag flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft disabled:opacity-50"
          >
            {busy && <Loader2 size={14} className="animate-spin" />}
            {kind === "sub" ? "Загрузить" : "Добавить"}
          </button>
        </div>
      </div>
    </div>
  );
}
