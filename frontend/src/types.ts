// Frontend mirror of the Go model types. The Wails-generated bindings return
// wider `string` types for some enums, so we narrow them here and cast at the
// binding boundary in App.tsx.

export type Core = "sing-box";
export type ConnState = "disconnected" | "connecting" | "connected" | "error";
export type Protocol =
  | "vless"
  | "vmess"
  | "trojan"
  | "shadowsocks"
  | "hysteria"
  | "hysteria2"
  | "tuic";

export interface Profile {
  id: string;
  name: string;
  protocol: Protocol;
  address: string;
  port: number;
  uuid?: string;
  password?: string;
  method?: string;
  alterId?: number;
  network?: string;
  security?: string;
  sni?: string;
  alpn?: string;
  fingerprint?: string;
  flow?: string;
  publicKey?: string;
  shortId?: string;
  path?: string;
  host?: string;
  serviceName?: string;
  obfs?: string;
  obfsPassword?: string;
  upMbps?: number;
  downMbps?: number;
  congestion?: string;
  udpRelayMode?: string;
  raw?: string;
  subId?: string;
}

export interface Subscription {
  id: string;
  name: string;
  url: string;
  updatedAt: number;
  count: number;
  upload: number;
  download: number;
  total: number; // 0 = unlimited
  expire: number; // unix seconds, 0 = never
}

export interface AppInfo {
  version: string;
  copyright: string;
  builtWith: string;
}

export interface RoutingRule {
  type: "domain" | "ip" | "process";
  value: string;
  action: "proxy" | "direct" | "block";
  icon?: string; // process rules only: PNG data URL of the app icon
}

export interface AppSettings {
  // Connection
  core: Core;
  activeProfileId: string;
  dns: string; // primary resolver, queried through the tunnel
  dnsFallback: string; // resolver for names that bypass the tunnel
  tunName: string;
  stack: string; // "gvisor" | "system"
  mtu: number;
  rules: RoutingRule[];
  // Application
  autoConnect: boolean;
  launchAtStartup: boolean;
  minimizeToTray: boolean;
  devMode: boolean; // unlocked by tapping the client name 10x in About
  // Appearance
  theme: string; // preset id
  accent: string; // palette id or raw #rrggbb
  savedColors: string[]; // user-saved custom accents
  font: string; // font id
  radius: string; // rounding id
  navPosition: string; // "left" | "top"
  animation: string; // entrance animation preset id
}

export interface Stats {
  upload: number;
  download: number;
  uploadSpeed: number;
  downloadSpeed: number;
}

export interface Status {
  state: ConnState;
  core: Core;
  activeProfile: Profile | null;
  error?: string;
  stats: Stats;
  connectedAt: number;
}

export type ViewKey = "connection" | "profiles" | "configs" | "routing" | "settings";
