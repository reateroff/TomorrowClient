// Helpers for rendering a profile's protocol stack as a readable chain, e.g.
// "VLESS · XHTTP · REALITY" or "Hysteria2 · TLS", instead of the bare protocol.
import type { Profile } from "./types";

const PROTO_LABEL: Record<string, string> = {
  vless: "VLESS",
  vmess: "VMess",
  trojan: "Trojan",
  shadowsocks: "Shadowsocks",
  hysteria: "Hysteria",
  hysteria2: "Hysteria2",
  tuic: "TUIC",
  wireguard: "WireGuard",
  anytls: "AnyTLS",
  socks: "SOCKS5",
  http: "HTTP",
};

const NET_LABEL: Record<string, string> = {
  tcp: "TCP",
  ws: "WS",
  grpc: "gRPC",
  http: "HTTP",
  httpupgrade: "HTTPUpgrade",
  xhttp: "XHTTP",
  h2: "H2",
  quic: "QUIC",
  kcp: "mKCP",
  udp: "UDP",
};

const SEC_LABEL: Record<string, string> = {
  tls: "TLS",
  reality: "REALITY",
  xtls: "XTLS",
};

// QUIC-based protocols carry their own transport and always run over TLS, so we
// don't repeat the network segment for them.
const UDP_BASED = new Set(["hysteria", "hysteria2", "tuic"]);
// Protocols with no V2Ray transport segment at all.
const NO_TRANSPORT = new Set(["wireguard", "anytls", "socks", "http"]);

// CORE_LABEL names the cores in the UI.
export const CORE_LABEL: Record<string, string> = {
  auto: "Авто",
  "sing-box": "sing-box",
  todaycore: "TodayCore",
  xray: "Xray",
  mihomo: "mihomo",
};

// protoLabel returns the human-readable protocol name.
export function protoLabel(p: Profile): string {
  return PROTO_LABEL[p.protocol] ?? p.protocol.toUpperCase();
}

// describeChain builds the "protocol · transport · security" string shown under
// each server name.
export function describeChain(p: Profile): string {
  const parts: string[] = [protoLabel(p)];

  if (!UDP_BASED.has(p.protocol) && !NO_TRANSPORT.has(p.protocol)) {
    const net = (p.network ?? "").toLowerCase();
    if (net && net !== "tcp") {
      parts.push(NET_LABEL[net] ?? net.toUpperCase());
    }
  }

  const sec = (p.security ?? "").toLowerCase();
  if (sec && sec !== "none") {
    parts.push(SEC_LABEL[sec] ?? sec.toUpperCase());
  } else if (UDP_BASED.has(p.protocol) || p.protocol === "anytls") {
    parts.push("TLS");
  }
  if (p.protocol === "vless" && p.encryption) parts.push("ENC");

  return parts.join(" · ");
}

// Current linked sing-box/TodayCore default to Go; Xray/mihomo are fronted.
// This is read-only, never a selection written to settings.stack.
export function tunStackInfo(core: string): {label: string; hint: string} {
  if (core === "auto") return {label: "По выбранному ядру", hint: "Нативный стек определяется ядром профиля. Ручной выбор отключён."};
  if (core === "todaycore") return {label: "TodayCore · Go", hint: "Встроенный стек TodayCore. Клиент не переопределяет его."};
  if (core === "sing-box") return {label: "sing-box · Go", hint: "Встроенный Go-стек sing-box. Ручной выбор отключён."};
  return {label: `${CORE_LABEL[core] || "Xray"} · sing-box TUN / Go`, hint: "Прокси обслуживает выбранное ядро, TUN — отдельный адаптер sing-box с нативным Go-стеком и кэшем этого ядра."};
}
