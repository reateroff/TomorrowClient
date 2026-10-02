import ModalPortal from "./ModalPortal";
import { useMemo, useState } from "react";
import { X, Loader2, Link2, Rss, FileCode2 } from "lucide-react";
import { plural } from "../format";

interface Props {
  onClose: () => void;
  onImportLink: (raw: string) => Promise<void>;
  onAddSub: (name: string, url: string) => Promise<void>;
}

// Kind of the pasted text, decided automatically from its content.
type Kind = "link" | "config" | "sub" | "empty" | "unknown";

// Mirrors link.Schemes on the Go side.
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
  "wireguard://",
  "wg://",
  "anytls://",
  "socks://",
  "socks5://",
];

const isLink = (line: string) => {
  const t = line.trim().toLowerCase();
  return LINK_SCHEMES.some((s) => t.startsWith(s));
};

// countLinks counts share links in pasted text — one per line.
function countLinks(text: string): number {
  return text.split(/\r?\n/).filter(isLink).length;
}

// looksLikeConfig spots a pasted sing-box config (JSON with outbounds), a
// Clash / mihomo config (YAML with proxies) or a base64 subscription body.
// The backend does the actual parsing; this only picks the hint and button.
function looksLikeConfig(t: string): boolean {
  if (t.startsWith("{") && /"(outbounds|endpoints)"/.test(t)) return true;
  if (/^proxies:/m.test(t)) return true;
  const compact = t.replace(/\s+/g, "");
  return compact.length > 32 && /^[A-Za-z0-9+/=_-]+$/.test(compact);
}

// detectKind decides whether the input is share links, a pasted config or a
// subscription URL. Share-link schemes win; http(s) is a subscription.
function detectKind(text: string): Kind {
  const t = text.trim();
  const lower = t.toLowerCase();
  if (!t) return "empty";
  if (countLinks(text) > 0) return "link";
  if (lower.startsWith("http://") || lower.startsWith("https://")) return "sub";
  if (looksLikeConfig(t)) return "config";
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
  const links = useMemo(() => countLinks(text), [text]);
  const canSubmit = kind === "link" || kind === "config" || kind === "sub";

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
      if (kind === "link" || kind === "config") {
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
    <ModalPortal onClose={onClose}>
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-view w-full max-w-md rounded-xl border border-border bg-surface p-5 shadow-2xl"
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
          placeholder="Вставьте ссылки (vless://, hysteria2://, wireguard://, … — по одной в строке) или адрес подписки https://…"
          rows={4}
          autoFocus
          className="w-full resize-none rounded-lg border border-border bg-bg px-3 py-2.5 font-mono text-xs text-text outline-none transition placeholder:text-text-faint focus:border-accent/60"
        />

        {/* Live detection hint */}
        <div className="mt-2 h-4 text-xs">
          {kind === "link" && (
            <span className="inline-flex items-center gap-1.5 text-ok">
              <Link2 size={12} />
              {links === 1
                ? "Определено: ссылка на сервер"
                : `Определено: ${links} ${plural(links, "ссылка", "ссылки", "ссылок")}`}
            </span>
          )}
          {kind === "config" && (
            <span className="inline-flex items-center gap-1.5 text-ok">
              <FileCode2 size={12} /> Определено: конфигурация (sing-box, Clash
              или base64)
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
            className="no-drag flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-on-accent transition hover:bg-accent-soft disabled:opacity-50"
          >
            {busy && <Loader2 size={14} className="animate-spin" />}
            {kind === "sub" ? "Загрузить" : "Добавить"}
          </button>
        </div>
      </div>
    </div>
    </ModalPortal>
  );
}
