// Package model holds the data types shared between the Go backend and the
// React frontend (they are exported through the Wails bindings).
package model

// Core names the proxy engine handling the traffic. Every core is linked into
// the app and runs in-process; there are no core executables to ship.
//
//   - sing-box runs the whole tunnel itself: TUN inbound with auto_route, DNS,
//     routing and the proxy outbound in one instance.
//   - TodayCore is sing-box with client features ported from Xray (XHTTP, VLESS
//     Encryption, current REALITY). It runs the tunnel the same way.
//   - Xray and mihomo only speak the proxy protocol. A sing-box instance owns
//     the TUN adapter, DNS and routing in front of them and hands proxied
//     traffic to the core over a loopback SOCKS link.
//
// CoreAuto is a setting, never a running core: it picks one of the above per
// profile (see internal/cores).
type Core string

const (
	CoreAuto      Core = "auto"
	CoreSingBox   Core = "sing-box"
	CoreTodayCore Core = "todaycore"
	CoreXray      Core = "xray"
	CoreMihomo    Core = "mihomo"
)

// ConnState is the high level connection state reported to the UI.
type ConnState string

const (
	StateDisconnected ConnState = "disconnected"
	StateConnecting   ConnState = "connecting"
	StateConnected    ConnState = "connected"
	StateError        ConnState = "error"
)

// Protocol is the outbound proxy protocol of a profile.
type Protocol string

const (
	ProtoVLESS       Protocol = "vless"
	ProtoVMess       Protocol = "vmess"
	ProtoTrojan      Protocol = "trojan"
	ProtoShadowsocks Protocol = "shadowsocks"
	ProtoHysteria    Protocol = "hysteria"  // Hysteria v1 (QUIC)
	ProtoHysteria2   Protocol = "hysteria2" // Hysteria2 (QUIC)
	ProtoTUIC        Protocol = "tuic"      // TUIC v5 (QUIC)
	ProtoWireGuard   Protocol = "wireguard"
	ProtoAnyTLS      Protocol = "anytls"
	ProtoSOCKS       Protocol = "socks"
	ProtoHTTP        Protocol = "http"
)

// Profile describes a single server / outbound. Fields are a superset that
// covers the protocols we support; unused ones stay empty.
type ProfileSpeed struct {
	DownloadMbps float64 `json:"downloadMbps"`
	Bytes        int64   `json:"bytes"`
	DurationMs   int64   `json:"durationMs"`
	Core         string  `json:"core"`
	MeasuredAt   int64   `json:"measuredAt"`
}

type Profile struct {
	Speed    *ProfileSpeed `json:"speed,omitempty"`
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Protocol Protocol      `json:"protocol"`
	Address  string        `json:"address"`
	Port     int           `json:"port"`

	// Auth / identity
	UUID     string `json:"uuid,omitempty"`     // vless / vmess / tuic
	Username string `json:"username,omitempty"` // socks / http
	Password string `json:"password,omitempty"` // trojan / shadowsocks / hysteria / tuic / anytls / socks / http
	Method   string `json:"method,omitempty"`   // shadowsocks cipher, vmess security
	AlterID  int    `json:"alterId,omitempty"`  // vmess

	// Encryption is VLESS Encryption ("mlkem768x25519plus.…"); empty or "none"
	// is plain VLESS.
	Encryption string `json:"encryption,omitempty"`
	// PacketEncoding is the VLESS/VMess UDP encapsulation ("xudp" /
	// "packetaddr"); empty means the core default.
	PacketEncoding string `json:"packetEncoding,omitempty"`

	// Transport
	Network       string `json:"network,omitempty"`       // tcp / ws / grpc / http / httpupgrade / xhttp / kcp / quic
	Security      string `json:"security,omitempty"`      // none / tls / reality
	SNI           string `json:"sni,omitempty"`           // tls server name
	ALPN          string `json:"alpn,omitempty"`          // comma separated
	Fingerprint   string `json:"fingerprint,omitempty"`   // utls fingerprint
	AllowInsecure bool   `json:"allowInsecure,omitempty"` // skip certificate verification
	Flow          string `json:"flow,omitempty"`          // vless flow (xtls-rprx-vision)
	PublicKey     string `json:"publicKey,omitempty"`     // reality public key / wireguard peer key
	ShortID       string `json:"shortId,omitempty"`       // reality
	SpiderX       string `json:"spiderX,omitempty"`       // reality
	Path          string `json:"path,omitempty"`          // ws/http/httpupgrade/xhttp path
	Host          string `json:"host,omitempty"`          // ws/http/httpupgrade/xhttp host header
	ServiceName   string `json:"serviceName,omitempty"`   // grpc
	HeaderType    string `json:"headerType,omitempty"`    // tcp "http" obfuscation / kcp header
	Seed          string `json:"seed,omitempty"`          // kcp seed

	// XHTTP (SplitHTTP)
	Mode  string `json:"mode,omitempty"`  // auto / packet-up / stream-up / stream-one
	Extra string `json:"extra,omitempty"` // Xray "extra" JSON object, as found in share links

	// Shadowsocks SIP003 plugin ("obfs-local" / "v2ray-plugin" / ...) and its
	// options string, as in the share link.
	Plugin     string `json:"plugin,omitempty"`
	PluginOpts string `json:"pluginOpts,omitempty"`

	// QUIC-based protocols (hysteria / hysteria2 / tuic)
	Obfs         string `json:"obfs,omitempty"`         // hysteria2 salamander / hysteria obfs
	ObfsPassword string `json:"obfsPassword,omitempty"` // hysteria2 obfs password
	UpMbps       int    `json:"upMbps,omitempty"`       // hysteria up bandwidth
	DownMbps     int    `json:"downMbps,omitempty"`     // hysteria down bandwidth
	Congestion   string `json:"congestion,omitempty"`   // tuic congestion control (bbr/cubic/new_reno)
	UDPRelayMode string `json:"udpRelayMode,omitempty"` // tuic udp relay mode (native/quic)
	// Ports is a Hysteria2 port-hopping range ("20000-30000" or "443,8443").
	Ports string `json:"ports,omitempty"`

	// WireGuard
	PrivateKey   string `json:"privateKey,omitempty"`
	PreSharedKey string `json:"preSharedKey,omitempty"`
	LocalAddress string `json:"localAddress,omitempty"` // interface addresses, comma separated CIDRs
	Reserved     string `json:"reserved,omitempty"`     // "1,2,3"
	MTU          int    `json:"mtu,omitempty"`

	// The original share link, kept so we can re-export / debug.
	Raw string `json:"raw,omitempty"`

	// SubID links this profile to the subscription it came from (empty for
	// profiles imported from a single link). Profiles sharing a SubID are
	// replaced wholesale when their subscription is updated.
	SubID string `json:"subId,omitempty"`
}

// Subscription is a remote URL that returns a list of share links (usually
// base64-encoded, one link per line). Importing it creates one profile per
// link; updating re-fetches and replaces those profiles.
type Subscription struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	// UpdatedAt is a unix millisecond timestamp of the last successful fetch.
	UpdatedAt int64 `json:"updatedAt"`
	// Count is how many profiles the last fetch produced.
	Count int `json:"count"`

	// Traffic and expiry parsed from the provider's Subscription-Userinfo
	// header. All zero means the provider reported nothing — the UI then shows
	// the values as unlimited (∞).
	Upload   int64 `json:"upload"`   // bytes used, upstream
	Download int64 `json:"download"` // bytes used, downstream
	Total    int64 `json:"total"`    // total quota in bytes, 0 = unlimited
	Expire   int64 `json:"expire"`   // unix seconds, 0 = never expires
}

// RoutingRule is one user-defined routing entry: match traffic by domain, IP
// or process (application) and send it through the proxy, direct, or block it.
type RoutingRule struct {
	Type   string `json:"type"`   // "domain" | "ip" | "process" | "geosite" | "geoip"
	Value  string `json:"value"`  // domain suffix, IP/CIDR, or process name
	Action string `json:"action"` // "proxy" | "direct" | "block"
	Icon   string `json:"icon"`   // process rules only: PNG data URL of the app icon
}

// Routing modes. Simple is the flat rule list; Pro is the node graph the user
// wires up by hand.
const (
	RoutingSimple = "simple"
	RoutingPro    = "pro"
)

// Latency check methods.
const (
	PingICMP = "icmp"
	PingTCP  = "tcp"
	PingGET  = "get"
	PingHEAD = "head"
)

// DefaultPingURL answers 204 with no body, so a check times the round trip
// rather than a download.
const DefaultPingURL = "http://cp.cloudflare.com/generate_204"

// Route actions a node or the catch-all can send traffic to.
const (
	ActionProxy  = "proxy"
	ActionDirect = "direct"
	ActionBlock  = "block"
)

// RouteNode is one matcher node of the Pro routing graph: a list of values of
// one kind, wired to one action. Nodes are evaluated in slice order — the
// first node whose values match decides — so the order is the priority.
type RouteNode struct {
	ID string `json:"id"`
	// Type is the matcher kind:
	//   domain (suffix), domain_full, domain_keyword, domain_regex,
	//   ip (CIDR), port (443 or 1000-2000), process (exe name),
	//   process_path, network (tcp/udp), protocol (sniffed: tls, http, quic,
	//   bittorrent, …), geosite and geoip (sing-box rule-set names).
	Type   string   `json:"type"`
	Name   string   `json:"name"`   // optional title; the type's name when empty
	Values []string `json:"values"` // matched if any value matches
	// Icons holds app icons (PNG data URLs) for process values, by value.
	Icons map[string]string `json:"icons,omitempty"`
	// Action is where matching traffic goes; empty leaves the node unwired,
	// which disables it without losing its values.
	Action string `json:"action"`
	// X, Y place the node on the canvas.
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Point is a canvas position.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// RouteGraph is the Pro routing setup.
type RouteNote struct {
	ID   string  `json:"id"`
	Text string  `json:"text"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type RouteGraph struct {
	Notes       []RouteNote `json:"notes,omitempty"`
	Nodes       []RouteNode `json:"nodes"`
	FinalHidden bool        `json:"finalHidden"`
	// Final is the action for traffic no node matched ("Остальное"); empty
	// when unwired, which routes it through the proxy.
	Final string `json:"final"`
	// Layout places the catch-all ("final") and the action nodes the user
	// added ("proxy", "direct", "block"); an action without an entry is not
	// on the canvas.
	Layout map[string]Point `json:"layout"`
}

// SettingsVersion is bumped whenever a stored setting needs a one-time rewrite
// on load. See Store.migrate: recording the version is what keeps a migration
// from running twice and overwriting a later deliberate choice.
const SettingsVersion = 3

// AppSettings is the persisted user configuration.
type CustomTheme struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Colors map[string]string `json:"colors"`
	Accent string            `json:"accent"`
}

type AppSettings struct {
	ClientPreset  string        `json:"clientPreset"`
	ClientVersion string        `json:"clientVersion"`
	CustomThemes  []CustomTheme `json:"customThemes"`
	UIScale       float64       `json:"uiScale"`
	Density       string        `json:"density"`
	CustomRadius  float64       `json:"customRadius"`
	AutoUpdate    bool          `json:"autoUpdate"`

	// SettingsVersion records which migrations have already been applied to
	// this file. Absent in files written before migrations existed, which
	// reads as 0.
	SettingsVersion int `json:"settingsVersion"`

	// --- Connection ---
	// Core is the engine to run, or CoreAuto to pick one per profile.
	Core            Core   `json:"core"`
	ActiveProfileID string `json:"activeProfileId"`
	// DNS is the primary resolver, queried through the tunnel.
	DNS string `json:"dns"`
	// DNSFallback resolves the names that are routed around the tunnel
	// (LAN, direct rules). sing-box has no automatic failover between
	// resolvers, so this is a second resolver rather than a stand-in.
	DNSFallback string `json:"dnsFallback"`
	// TunName is the name of the WinTun adapter the tunnel creates. Empty means
	// the built-in default ("TomorrowTun").
	TunName string `json:"tunName"`
	// Stack is a legacy compatibility field. It is cleared on load/save and
	// never passed to a core. Each tunnel uses its linked core's native stack.
	Stack string `json:"stack"`
	// MTU of the TUN interface; 0 means the core default (usually 9000/1500).
	MTU int `json:"mtu"`
	// Rules are user-defined domain/ip/app routing overrides (simple mode).
	Rules []RoutingRule `json:"rules"`
	// RoutingMode is RoutingSimple (Rules) or RoutingPro (Graph).
	RoutingMode string `json:"routingMode"`
	// SimpleFinal is the catch-all for Easy routing; empty keeps the legacy proxy default.
	SimpleFinal string `json:"simpleFinal,omitempty"`
	// Graph is the Pro routing graph.
	Graph RouteGraph `json:"graph"`

	// --- Tunnel (advanced) ---
	// IPv6 gives the TUN adapter an IPv6 address and resolves AAAA records, so
	// IPv6 traffic is tunnelled instead of left to leak around it.
	IPv6 bool `json:"ipv6"`
	// StrictRoute makes the tunnel refuse traffic that tries to leave around
	// it (DNS leaks included). On by default.
	StrictRoute bool `json:"strictRoute"`
	// Sniff reads the domain out of TLS/HTTP/QUIC so domain rules match
	// connections that arrive as bare IPs. On by default.
	Sniff bool `json:"sniff"`

	// --- Server checks ---
	// PingMethod is how latency is measured: PingICMP / PingTCP (reach the
	// server itself) or PingGET / PingHEAD (a real request through the proxy,
	// which also proves the server works).
	PingMethod string `json:"pingMethod"`
	// PingURL is fetched through the proxy by the GET and HEAD methods.
	PingURL string `json:"pingUrl"`
	// PingTimeout bounds one check, in milliseconds.
	PingTimeout int `json:"pingTimeout"`

	// --- Device (HWID) ---
	// HWIDEnabled sends the device headers (x-hwid, x-device-os, x-ver-os,
	// x-device-model) with subscription requests; panels that limit devices
	// per subscription need them. On by default.
	HWIDEnabled bool `json:"hwidEnabled"`
	// The values below override what is detected; empty means the real one.
	HWID        string `json:"hwid"`
	DeviceOS    string `json:"deviceOs"`
	OSVersion   string `json:"osVersion"`
	DeviceModel string `json:"deviceModel"`
	// UserAgent overrides the subscription User-Agent, which panels also use
	// to pick the format they answer with. Sent whether or not HWID is on.
	UserAgent string `json:"userAgent"`

	// --- Application ---
	// AutoConnect connects to the last active profile on launch.
	AutoConnect bool `json:"autoConnect"`
	// LaunchAtStartup registers a Task Scheduler task to start with Windows.
	LaunchAtStartup bool `json:"launchAtStartup"`
	// MinimizeToTray hides the window to the notification area on minimize
	// instead of the taskbar.
	MinimizeToTray bool `json:"minimizeToTray"`
	// DevMode unlocks the developer tools section. It is hidden until the user
	// taps the client name on the About screen ten times.
	DevMode bool `json:"devMode"`
	// DemoMode masks addresses, keys and raw dumps on screen so the app can be
	// shown or recorded without leaking server details.
	DemoMode bool `json:"demoMode"`

	// --- Appearance ---
	// Theme is the base preset id ("graphite" / "midnight" / "coal").
	Theme string `json:"theme"`
	// Accent is a hex color id from the palette ("indigo", "teal", ...) or a
	// raw "#rrggbb" when the user picked a custom one.
	Accent string `json:"accent"`
	// SavedColors are custom hex accents the user chose to keep, in the order
	// they were saved.
	SavedColors []string `json:"savedColors"`
	// Font is the UI font id ("inter" / "mono" / "geist").
	Font string `json:"font"`
	// Radius is the corner rounding preset ("sharp" / "soft" / "round").
	Radius string `json:"radius"`
	// NavPosition places the navigation tabs on the "left", "top" (default), "right" or "bottom".
	NavPosition string `json:"navPosition"`
	// Animation is the entrance animation preset for views and modals
	// ("rise" / "slide" / "fade" / "scale" / "none").
	Animation string `json:"animation"`
}

// TunInterfaceName returns the configured TUN adapter name, falling back to the
// built-in default when unset. This is the single source of truth shared by the
// sing-box config and the traffic-stats matcher.
func (s AppSettings) TunInterfaceName() string {
	if s.TunName != "" {
		return s.TunName
	}
	return DefaultTunName
}

// DefaultTunName is the built-in WinTun adapter name.
const DefaultTunName = "TomorrowTun"

// DefaultSettings pins Xray out of the box. Auto remains an explicit opt-in.
func DefaultSettings() AppSettings {
	return AppSettings{
		SettingsVersion: SettingsVersion,

		Core:        CoreXray,
		AutoUpdate:  true,
		UIScale:     100,
		Density:     "comfortable",
		AutoConnect: false,
		DNS:         "1.1.1.1",
		DNSFallback: "8.8.8.8",
		TunName:     DefaultTunName,
		Stack:       "",
		MTU:         0,
		RoutingMode: RoutingSimple,
		SimpleFinal: ActionProxy,
		Graph:       RouteGraph{Final: ActionProxy},
		StrictRoute: true,
		Sniff:       true,
		HWIDEnabled: true,
		PingMethod:  PingGET,
		PingURL:     DefaultPingURL,
		PingTimeout: 5000,
		Theme:       "smoke",
		Accent:      "sunset-mist",
		Font:        "rubik",
		Radius:      "soft",
		NavPosition: "top",
		Animation:   "fade",
	}
}

// Stats holds live traffic counters (cumulative bytes for the session and the
// last measured speed in bytes/second).
type Stats struct {
	Upload        uint64 `json:"upload"`
	Download      uint64 `json:"download"`
	UploadSpeed   uint64 `json:"uploadSpeed"`
	DownloadSpeed uint64 `json:"downloadSpeed"`
}

// Status is the full connection snapshot pushed to the UI.
type Status struct {
	State ConnState `json:"state"`
	// Core is the engine actually carrying the connection — never CoreAuto.
	Core          Core     `json:"core"`
	ActiveProfile *Profile `json:"activeProfile"`
	Error         string   `json:"error,omitempty"`
	Stats         Stats    `json:"stats"`
	// ConnectedAt is a unix millisecond timestamp, 0 when not connected.
	ConnectedAt int64 `json:"connectedAt"`
}
