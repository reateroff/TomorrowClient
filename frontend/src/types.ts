// Frontend mirror of the Go model types. The Wails-generated bindings return
// wider `string` types for some enums, so we narrow them here and cast at the
// binding boundary in App.tsx.

export type Core = "sing-box" | "xray";
export type ConnState = "disconnected" | "connecting" | "connected" | "error";
export type RoutingMode = "global" | "rules";
export type Protocol = "vless" | "vmess" | "trojan" | "shadowsocks";

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
  raw?: string;
  subId?: string;
}

export interface Subscription {
  id: string;
  name: string;
  url: string;
  updatedAt: number;
  count: number;
}

export interface AppInfo {
  version: string;
  copyright: string;
  builtWith: string;
}

export interface AppSettings {
  core: Core;
  activeProfileId: string;
  routingMode: RoutingMode;
  autoConnect: boolean;
  dns: string;
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

export type ViewKey = "connection" | "profiles" | "settings";
