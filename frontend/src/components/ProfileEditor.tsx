import { useEffect, useMemo, useState } from "react";
import { X, Copy, Check, FileJson, Link2, Lock, CircleCheck, CircleX } from "lucide-react";
import type { Profile, ProfileCores } from "../types";
import { GetProfileCores, SaveProfile, SaveProfileJSON } from "../../wailsjs/go/main/App";
import { protoLabel, CORE_LABEL } from "../proto";
import { push } from "./Toasts";

// ProfileEditor shows a server as it was imported and lets it be edited in the
// same form. A server that came from a sing-box or Clash config opens as that
// JSON; one that came from a share link opens as a form, and saving rebuilds
// the link from the fields.

interface Props {
  profile: Profile;
  hideData: boolean; // demo mode: secrets stay off screen
  onClose: () => void;
  onSaved: () => void;
}

export default function ProfileEditor({ profile, hideData, onClose, onSaved }: Props) {
  const isJSON = (profile.raw ?? "").trim().startsWith("{");

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6 backdrop-blur-sm" onClick={onClose}>
      <div
        className="animate-view flex max-h-full w-full max-w-4xl flex-col overflow-hidden rounded-xl border border-border bg-surface shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 border-b border-border px-5 py-3.5">
          <span className="grid h-8 w-8 place-items-center rounded-md bg-accent/15 text-accent">
            {isJSON ? <FileJson size={16} /> : <Link2 size={16} />}
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-medium text-text">{profile.name}</div>
            <div className="font-mono text-[11px] text-text-faint">
              {protoLabel(profile)} · {isJSON ? "из JSON-конфига" : "из ссылки"}
            </div>
          </div>
          <button onClick={onClose} className="no-drag rounded-md p-1.5 text-text-faint transition hover:bg-surface-2 hover:text-text">
            <X size={16} />
          </button>
        </div>

        {hideData ? (
          <div className="flex flex-col items-center gap-2 px-6 py-14 text-center text-sm text-text-muted">
            <Lock size={20} className="text-text-faint" />
            Конфигурация скрыта в демо-режиме
          </div>
        ) : isJSON ? (
          <JSONEditor profile={profile} onClose={onClose} onSaved={onSaved} />
        ) : (
          <FormEditor profile={profile} onClose={onClose} onSaved={onSaved} />
        )}
      </div>
    </div>
  );
}

// ------------------------------------------------------------------ JSON

function JSONEditor({ profile, onClose, onSaved }: Omit<Props, "hideData">) {
  const [text, setText] = useState(profile.raw ?? "");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const save = async () => {
    try {
      JSON.parse(text);
    } catch (e: any) {
      setErr(`Некорректный JSON: ${e.message}`);
      return;
    }
    setBusy(true);
    try {
      await SaveProfileJSON(profile.id, text);
      push("Сервер сохранён", "ok");
      onSaved();
      onClose();
    } catch (e: any) {
      setErr(String(e?.message ?? e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="min-h-0 flex-1 overflow-hidden p-4">
        <textarea
          value={text}
          onChange={(e) => (setText(e.target.value), setErr(""))}
          spellCheck={false}
          className="h-[52vh] w-full resize-none rounded-lg border border-border bg-bg p-3.5 font-mono text-xs leading-relaxed text-text outline-none transition focus:border-accent/60"
        />
        {err && <div className="mt-2 text-xs text-danger">{err}</div>}
      </div>
      <Footer copyText={text} copyLabel="JSON" busy={busy} onCancel={onClose} onSave={save} />
    </>
  );
}

// ------------------------------------------------------------------ form

const V2RAY = new Set(["vless", "vmess", "trojan"]);
const QUIC = new Set(["hysteria", "hysteria2", "tuic"]);

const opt = (...v: string[]) => v.map((x) => ({ id: x, label: x || "—" }));

const NETWORKS = [
  { id: "tcp", label: "tcp (raw)" },
  { id: "ws", label: "ws" },
  { id: "grpc", label: "grpc" },
  { id: "http", label: "http (h2)" },
  { id: "httpupgrade", label: "httpupgrade" },
  { id: "xhttp", label: "xhttp" },
  { id: "kcp", label: "mkcp" },
  { id: "quic", label: "quic" },
];
const FINGERPRINTS = opt("", "chrome", "firefox", "safari", "ios", "android", "edge", "360", "qq", "random", "randomized");
const SS_METHODS = opt(
  "aes-128-gcm", "aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305",
  "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305", "none"
);

function FormEditor({ profile, onClose, onSaved }: Omit<Props, "hideData">) {
  const [p, setP] = useState<Profile>(profile);
  const [busy, setBusy] = useState(false);
  const [cores, setCores] = useState<ProfileCores | null>(null);
  const set = (patch: Partial<Profile>) => setP((x) => ({ ...x, ...patch }));

  useEffect(() => {
    GetProfileCores(profile.id).then((c) => setCores(c as unknown as ProfileCores));
  }, [profile.id]);

  const v2ray = V2RAY.has(p.protocol);
  const quic = QUIC.has(p.protocol);
  const net = p.network || "tcp";
  const extraOK = useMemo(() => {
    if (!p.extra) return true;
    try {
      JSON.parse(p.extra);
      return true;
    } catch {
      return false;
    }
  }, [p.extra]);

  const save = async () => {
    if (!p.address.trim() || !p.port) {
      push("Нужны адрес и порт", "warn");
      return;
    }
    if (!extraOK) {
      push("Extra XHTTP — некорректный JSON", "warn");
      return;
    }
    setBusy(true);
    try {
      await SaveProfile(p as any);
      push("Сервер сохранён", "ok", { description: "Изменения вступят в силу при следующем подключении" });
      onSaved();
      onClose();
    } catch (e: any) {
      push("Не удалось сохранить", "error", { description: String(e?.message ?? e) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="grid min-h-0 flex-1 grid-cols-2 gap-4 overflow-y-auto p-4">
        <div className="flex flex-col gap-4">
          <Group title="Основные">
            <Field label="Название">
              <Input value={p.name} onChange={(v) => set({ name: v })} />
            </Field>
            <Field label="Адрес">
              <Input value={p.address} onChange={(v) => set({ address: v })} mono />
            </Field>
            <Field label="Порт">
              <Input value={String(p.port || "")} onChange={(v) => set({ port: parseInt(v) || 0 })} mono />
            </Field>
          </Group>

          <Group title={protoLabel(p)}>
            <ProtocolFields p={p} set={set} />
          </Group>

          {v2ray && (
            <Group title="Транспорт">
              <Field label="Сеть">
                <Select value={net} options={NETWORKS} onChange={(v) => set({ network: v })} />
              </Field>
              {net === "tcp" && (
                <Field label="Обфускация">
                  <Select value={p.headerType || ""} options={[{ id: "", label: "нет" }, { id: "http", label: "http" }]} onChange={(v) => set({ headerType: v })} />
                </Field>
              )}
              {(["ws", "httpupgrade", "http", "xhttp"].includes(net) || (net === "tcp" && p.headerType === "http")) && (
                <>
                  <Field label="Путь">
                    <Input value={p.path ?? ""} onChange={(v) => set({ path: v })} mono placeholder="/" />
                  </Field>
                  <Field label="Host">
                    <Input value={p.host ?? ""} onChange={(v) => set({ host: v })} mono />
                  </Field>
                </>
              )}
              {net === "grpc" && (
                <>
                  <Field label="Service name">
                    <Input value={p.serviceName ?? ""} onChange={(v) => set({ serviceName: v })} mono />
                  </Field>
                  <Field label="Режим">
                    <Select value={p.mode || "gun"} options={opt("gun", "multi")} onChange={(v) => set({ mode: v })} />
                  </Field>
                </>
              )}
              {net === "xhttp" && (
                <>
                  <Field label="Режим">
                    <Select value={p.mode || "auto"} options={opt("auto", "packet-up", "stream-up", "stream-one")} onChange={(v) => set({ mode: v })} />
                  </Field>
                  <Field label="Extra" top>
                    <textarea
                      value={p.extra ?? ""}
                      onChange={(e) => set({ extra: e.target.value })}
                      spellCheck={false}
                      rows={3}
                      placeholder='{"xPaddingBytes": "100-1000"}'
                      className={`w-full resize-y rounded-md border bg-bg px-2.5 py-1.5 font-mono text-xs text-text outline-none placeholder:text-text-faint focus:border-accent/60 ${
                        extraOK ? "border-border" : "border-danger/60"
                      }`}
                    />
                  </Field>
                </>
              )}
              {net === "kcp" && (
                <>
                  <Field label="Заголовок">
                    <Select value={p.headerType || "none"} options={opt("none", "srtp", "utp", "wechat-video", "dtls", "wireguard")} onChange={(v) => set({ headerType: v })} />
                  </Field>
                  <Field label="Seed">
                    <Input value={p.seed ?? ""} onChange={(v) => set({ seed: v })} mono />
                  </Field>
                </>
              )}
            </Group>
          )}
        </div>

        <div className="flex flex-col gap-4">
          {p.protocol !== "wireguard" && (
            <Group title="Настройки безопасности">
              {(v2ray || p.protocol === "socks" || p.protocol === "http") && (
                <Field label="Безопасность">
                  <Select
                    value={p.security || "none"}
                    options={v2ray ? opt("none", "tls", "reality") : opt("none", "tls")}
                    onChange={(v) => set({ security: v })}
                  />
                </Field>
              )}
              {(quic || p.protocol === "anytls" || (p.security && p.security !== "none")) && (
                <>
                  <Field label="SNI">
                    <Input value={p.sni ?? ""} onChange={(v) => set({ sni: v })} mono />
                  </Field>
                  <Field label="Отпечаток (fingerprint)">
                    <Select value={p.fingerprint ?? ""} options={FINGERPRINTS} onChange={(v) => set({ fingerprint: v })} />
                  </Field>
                  <Field label="ALPN">
                    <Input value={p.alpn ?? ""} onChange={(v) => set({ alpn: v })} mono placeholder="h2,http/1.1" />
                  </Field>
                  {p.security !== "reality" && (
                    <Field label="Без проверки сертификата">
                      <Switch on={!!p.allowInsecure} onChange={(v) => set({ allowInsecure: v })} />
                    </Field>
                  )}
                </>
              )}
              {p.security === "reality" && (
                <>
                  <Field label="Reality Pbk">
                    <Input value={p.publicKey ?? ""} onChange={(v) => set({ publicKey: v })} mono />
                  </Field>
                  <Field label="Reality SID">
                    <Input value={p.shortId ?? ""} onChange={(v) => set({ shortId: v })} mono />
                  </Field>
                  <Field label="Reality SpiderX">
                    <Input value={p.spiderX ?? ""} onChange={(v) => set({ spiderX: v })} mono />
                  </Field>
                </>
              )}
            </Group>
          )}

          <Group title="Ядра">
            {cores ? (
              <div className="flex flex-col gap-1.5">
                {cores.cores.map((c) => (
                  <div key={c.core} className="flex items-start gap-2 text-xs">
                    {c.supported ? (
                      <CircleCheck size={14} className="mt-px shrink-0 text-ok" />
                    ) : (
                      <CircleX size={14} className="mt-px shrink-0 text-text-faint" />
                    )}
                    <span className={c.supported ? "text-text" : "text-text-faint"}>
                      {CORE_LABEL[c.core] ?? c.core}
                      {c.core === cores.selected && (
                        <span className="ml-1.5 rounded bg-accent/15 px-1.5 py-px font-mono text-[10px] text-accent">сейчас</span>
                      )}
                      {!c.supported && c.reason && <span className="block text-[11px] text-text-faint">{c.reason}</span>}
                    </span>
                  </div>
                ))}
                <p className="mt-1 text-[11px] text-text-faint">Совместимость сохранённой версии сервера</p>
              </div>
            ) : (
              <div className="text-xs text-text-faint">…</div>
            )}
          </Group>
        </div>
      </div>
      <Footer copyText={profile.raw ?? ""} copyLabel="ссылку" busy={busy} onCancel={onClose} onSave={save} />
    </>
  );
}

// ProtocolFields are the credentials and knobs specific to one protocol.
function ProtocolFields({ p, set }: { p: Profile; set: (x: Partial<Profile>) => void }) {
  switch (p.protocol) {
    case "vless":
      return (
        <>
          <Field label="UUID"><Input value={p.uuid ?? ""} onChange={(v) => set({ uuid: v })} mono /></Field>
          <Field label="Шифрование"><Input value={p.encryption || "none"} onChange={(v) => set({ encryption: v === "none" ? "" : v })} mono /></Field>
          <Field label="Flow">
            <Select value={p.flow ?? ""} options={opt("", "xtls-rprx-vision", "xtls-rprx-vision-udp443")} onChange={(v) => set({ flow: v })} />
          </Field>
          <Field label="Упаковка UDP">
            <Select value={p.packetEncoding ?? ""} options={opt("", "xudp", "packetaddr")} onChange={(v) => set({ packetEncoding: v })} />
          </Field>
        </>
      );
    case "vmess":
      return (
        <>
          <Field label="UUID"><Input value={p.uuid ?? ""} onChange={(v) => set({ uuid: v })} mono /></Field>
          <Field label="Alter ID"><Input value={String(p.alterId ?? 0)} onChange={(v) => set({ alterId: parseInt(v) || 0 })} mono /></Field>
          <Field label="Шифрование">
            <Select value={p.method || "auto"} options={opt("auto", "aes-128-gcm", "chacha20-poly1305", "none", "zero")} onChange={(v) => set({ method: v })} />
          </Field>
          <Field label="Упаковка UDP">
            <Select value={p.packetEncoding ?? ""} options={opt("", "xudp", "packetaddr")} onChange={(v) => set({ packetEncoding: v })} />
          </Field>
        </>
      );
    case "trojan":
    case "anytls":
      return <Field label="Пароль"><Input value={p.password ?? ""} onChange={(v) => set({ password: v })} mono /></Field>;
    case "shadowsocks":
      return (
        <>
          <Field label="Метод"><Select value={p.method ?? ""} options={SS_METHODS} onChange={(v) => set({ method: v })} /></Field>
          <Field label="Пароль"><Input value={p.password ?? ""} onChange={(v) => set({ password: v })} mono /></Field>
          <Field label="Плагин">
            <Select value={p.plugin ?? ""} options={opt("", "obfs-local", "v2ray-plugin")} onChange={(v) => set({ plugin: v })} />
          </Field>
          {p.plugin && (
            <Field label="Параметры"><Input value={p.pluginOpts ?? ""} onChange={(v) => set({ pluginOpts: v })} mono placeholder="obfs=http;obfs-host=…" /></Field>
          )}
        </>
      );
    case "hysteria2":
    case "hysteria":
      return (
        <>
          <Field label={p.protocol === "hysteria" ? "Auth" : "Пароль"}><Input value={p.password ?? ""} onChange={(v) => set({ password: v })} mono /></Field>
          {p.protocol === "hysteria2" ? (
            <>
              <Field label="Obfs"><Select value={p.obfs ?? ""} options={opt("", "salamander")} onChange={(v) => set({ obfs: v })} /></Field>
              {p.obfs && <Field label="Пароль obfs"><Input value={p.obfsPassword ?? ""} onChange={(v) => set({ obfsPassword: v })} mono /></Field>}
              <Field label="Смена портов"><Input value={p.ports ?? ""} onChange={(v) => set({ ports: v })} mono placeholder="20000-30000" /></Field>
            </>
          ) : (
            <Field label="Obfs"><Input value={p.obfs ?? ""} onChange={(v) => set({ obfs: v })} mono /></Field>
          )}
          <Field label="Up / Down, Мбит/с">
            <div className="flex gap-2">
              <Input value={String(p.upMbps || "")} onChange={(v) => set({ upMbps: parseInt(v) || 0 })} mono placeholder="up" />
              <Input value={String(p.downMbps || "")} onChange={(v) => set({ downMbps: parseInt(v) || 0 })} mono placeholder="down" />
            </div>
          </Field>
        </>
      );
    case "tuic":
      return (
        <>
          <Field label="UUID"><Input value={p.uuid ?? ""} onChange={(v) => set({ uuid: v })} mono /></Field>
          <Field label="Пароль"><Input value={p.password ?? ""} onChange={(v) => set({ password: v })} mono /></Field>
          <Field label="Congestion"><Select value={p.congestion ?? ""} options={opt("", "bbr", "cubic", "new_reno")} onChange={(v) => set({ congestion: v })} /></Field>
          <Field label="UDP relay"><Select value={p.udpRelayMode ?? ""} options={opt("", "native", "quic")} onChange={(v) => set({ udpRelayMode: v })} /></Field>
        </>
      );
    case "wireguard":
      return (
        <>
          <Field label="Приватный ключ"><Input value={p.privateKey ?? ""} onChange={(v) => set({ privateKey: v })} mono /></Field>
          <Field label="Ключ сервера"><Input value={p.publicKey ?? ""} onChange={(v) => set({ publicKey: v })} mono /></Field>
          <Field label="PSK"><Input value={p.preSharedKey ?? ""} onChange={(v) => set({ preSharedKey: v })} mono /></Field>
          <Field label="Адреса"><Input value={p.localAddress ?? ""} onChange={(v) => set({ localAddress: v })} mono placeholder="10.0.0.2/32" /></Field>
          <Field label="Reserved"><Input value={p.reserved ?? ""} onChange={(v) => set({ reserved: v })} mono placeholder="1,2,3" /></Field>
          <Field label="MTU"><Input value={String(p.mtu || "")} onChange={(v) => set({ mtu: parseInt(v) || 0 })} mono placeholder="1420" /></Field>
        </>
      );
    case "socks":
    case "http":
      return (
        <>
          <Field label="Логин"><Input value={p.username ?? ""} onChange={(v) => set({ username: v })} mono /></Field>
          <Field label="Пароль"><Input value={p.password ?? ""} onChange={(v) => set({ password: v })} mono /></Field>
        </>
      );
  }
  return null;
}

// ------------------------------------------------------------------ parts

function Footer({
  copyText,
  copyLabel,
  busy,
  onCancel,
  onSave,
}: {
  copyText: string;
  copyLabel: string;
  busy: boolean;
  onCancel: () => void;
  onSave: () => void;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="flex items-center gap-2 border-t border-border px-4 py-3">
      {copyText && (
        <>
          <button
            onClick={async () => {
              await navigator.clipboard.writeText(copyText);
              setCopied(true);
              setTimeout(() => setCopied(false), 1200);
            }}
            className="no-drag flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-text-muted transition hover:bg-surface-2 hover:text-text"
          >
            {copied ? <Check size={13} className="text-ok" /> : <Copy size={13} />}
            Копировать {copyLabel}
          </button>
          {copyLabel === "ссылку" && (
            <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-text-faint">{copyText}</span>
          )}
        </>
      )}
      <div className="ml-auto flex gap-2">
        <button onClick={onCancel} className="no-drag rounded-lg border border-border px-4 py-1.5 text-sm text-text-muted transition hover:bg-surface-2">
          Отмена
        </button>
        <button
          onClick={onSave}
          disabled={busy}
          className="no-drag rounded-lg bg-accent px-5 py-1.5 text-sm font-medium text-bg transition hover:bg-accent-soft disabled:opacity-50"
        >
          Сохранить
        </button>
      </div>
    </div>
  );
}

function Group({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <fieldset className="flex flex-col gap-2 rounded-lg border border-border px-3.5 pb-3.5 pt-1">
      <legend className="px-1.5 text-xs text-text-muted">{title}</legend>
      {children}
    </fieldset>
  );
}

function Field({ label, children, top }: { label: string; children: React.ReactNode; top?: boolean }) {
  return (
    <label className={`grid grid-cols-[132px_1fr] gap-3 ${top ? "items-start" : "items-center"}`}>
      <span className={`text-xs text-text-muted ${top ? "pt-1.5" : ""}`}>{label}</span>
      {children}
    </label>
  );
}

function Input({
  value,
  onChange,
  mono,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  mono?: boolean;
  placeholder?: string;
}) {
  return (
    <input
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      spellCheck={false}
      className={`w-full min-w-0 rounded-md border border-border bg-bg px-2.5 py-1.5 text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 ${
        mono ? "font-mono text-xs" : ""
      }`}
    />
  );
}

// Select is native here: a dense form with many dropdowns reads better with
// the system list than with a popover per field.
function Select({
  value,
  options,
  onChange,
}: {
  value: string;
  options: { id: string; label: string }[];
  onChange: (v: string) => void;
}) {
  const known = options.some((o) => o.id === value);
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-md border border-border bg-bg px-2 py-1.5 font-mono text-xs text-text outline-none transition [color-scheme:dark] focus:border-accent/60"
    >
      {!known && <option value={value}>{value}</option>}
      {options.map((o) => (
        <option key={o.id} value={o.id} className="bg-surface-2">
          {o.label}
        </option>
      ))}
    </select>
  );
}

function Switch({ on, onChange }: { on: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!on)}
      className={`relative h-5 w-9 rounded-full transition ${on ? "bg-accent" : "bg-surface-2 border border-border"}`}
    >
      <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-text transition-all ${on ? "left-[18px]" : "left-0.5"}`} />
    </button>
  );
}
