// Frontend mirror of the Go model types. The Wails-generated bindings return
// wider `string` types for some enums, so we narrow them here and cast at the
// binding boundary in App.tsx.

// A concrete core, as the engine reports it.
export type Core = "sing-box" | "todaycore" | "xray" | "mihomo";
// The setting may also leave the choice to the app, per profile.
export type CoreSetting = Core | "auto";
export type ConnState = "disconnected" | "connecting" | "connected" | "error";
export type Protocol =
  | "vless"
  | "vmess"
  | "trojan"
  | "shadowsocks"
  | "hysteria"
  | "hysteria2"
  | "tuic"
  | "wireguard"
  | "anytls"
  | "socks"
  | "http";

export interface ProfileSpeed {downloadMbps:number;bytes:number;durationMs:number;core:string;measuredAt:number}

export interface Profile {
  speed?:ProfileSpeed;
  id: string;
  name: string;
  protocol: Protocol;
  address: string;
  port: number;
  uuid?: string;
  username?: string;
  password?: string;
  method?: string;
  alterId?: number;
  encryption?: string; // VLESS Encryption
  packetEncoding?: string;
  network?: string;
  security?: string;
  sni?: string;
  alpn?: string;
  fingerprint?: string;
  allowInsecure?: boolean;
  flow?: string;
  publicKey?: string;
  shortId?: string;
  spiderX?: string;
  path?: string;
  host?: string;
  serviceName?: string;
  headerType?: string;
  seed?: string;
  mode?: string; // XHTTP mode / gRPC mode
  extra?: string; // XHTTP extra JSON
  plugin?: string;
  pluginOpts?: string;
  obfs?: string;
  obfsPassword?: string;
  upMbps?: number;
  downMbps?: number;
  congestion?: string;
  udpRelayMode?: string;
  ports?: string; // Hysteria2 port hopping
  privateKey?: string;
  preSharedKey?: string;
  localAddress?: string;
  reserved?: string;
  mtu?: number;
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
  license: string;
}

export interface CoreInfo {
  id: Core;
  name: string;
  version: string;
  description: string;
}

export interface CoreSupport {
  core: Core;
  supported: boolean;
  reason?: string;
}

export interface ProfileCores {
  selected: Core;
  error?: string;
  cores: CoreSupport[];
}

export interface RoutingRule {
  type: "domain" | "ip" | "process" | "geosite" | "geoip";
  value: string;
  action: "proxy" | "direct" | "block";
  icon?: string; // process rules only: PNG data URL of the app icon
}

export type RouteAction = "proxy" | "direct" | "block";

export type NodeType =
  | "domain"
  | "domain_full"
  | "domain_keyword"
  | "domain_regex"
  | "ip"
  | "port"
  | "process"
  | "process_path"
  | "network"
  | "protocol"
  | "geosite"
  | "geoip";

// RouteNode is one matcher node of the Pro routing graph. Nodes are evaluated
// in array order; the first match decides.
export interface RouteNode {
  id: string;
  type: NodeType;
  name: string;
  values: string[];
  icons?: Record<string, string>; // process icons by value
  action: RouteAction | ""; // "" = not wired
  x: number;
  y: number;
}

export interface Point {
  x: number;
  y: number;
}

export interface RouteNote {id:string;text:string;x:number;y:number}

export interface RouteGraph {
  notes?: RouteNote[];
  finalHidden?: boolean;
  nodes: RouteNode[];
  final: RouteAction | ""; // traffic nothing matched; "" = unwired (goes through the proxy)
  layout: Record<string, Point>; // "final", "proxy", "direct", "block"
}

export interface CustomTheme { id:string; name:string; colors:Record<string,string>; accent:string }
export interface AppSettings {
  clientPreset?: string; clientVersion?: string;
  customThemes?: CustomTheme[];
  uiScale?: number;
  density?: string;
  customRadius?: number;
  autoUpdate?: boolean;
  // Connection
  core: CoreSetting;
  activeProfileId: string;
  dns: string; // primary resolver, queried through the tunnel
  dnsFallback: string; // resolver for names that bypass the tunnel
  tunName: string;
  stack: string; // Legacy field, always empty; the core owns its native TUN stack.
  mtu: number;
  rules: RoutingRule[];
  routingMode: "simple" | "pro";
  simpleFinal?: RouteAction;
  graph: RouteGraph;
  // Tunnel (advanced)
  ipv6: boolean;
  strictRoute: boolean;
  sniff: boolean;
  // Device (HWID); empty strings mean the real value
  hwidEnabled: boolean;
  hwid: string;
  deviceOs: string;
  osVersion: string;
  deviceModel: string;
  userAgent: string;
  // Server checks
  pingMethod: "icmp" | "tcp" | "get" | "head";
  pingUrl: string;
  pingTimeout: number; // ms
  // Application
  autoConnect: boolean;
  launchAtStartup: boolean;
  minimizeToTray: boolean;
  devMode: boolean; // unlocked by tapping the client name 10x in About
  demoMode: boolean; // masks addresses, keys and raw dumps on screen
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
