// Package model holds the data types shared between the Go backend and the
// React frontend (they are exported through the Wails bindings).
package model

// Core selects which proxy engine handles the traffic.
type Core string

const (
	CoreSingBox Core = "sing-box" // default: native TUN + auto_route, manages its own routing
	CoreXray    Core = "xray"     // xray SOCKS + tun2socks WinTun adapter + manual routes
)

// ConnState is the high level connection state reported to the UI.
type ConnState string

const (
	StateDisconnected ConnState = "disconnected"
	StateConnecting   ConnState = "connecting"
	StateConnected    ConnState = "connected"
	StateError        ConnState = "error"
)

// RoutingMode controls how much traffic is sent through the tunnel.
type RoutingMode string

const (
	RoutingGlobal RoutingMode = "global" // everything through the proxy
	RoutingRules  RoutingMode = "rules"  // bypass LAN + direct-list, proxy the rest
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
)

// Profile describes a single server / outbound. Fields are a superset that
// covers the protocols we support; unused ones stay empty.
type Profile struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Protocol Protocol `json:"protocol"`
	Address  string   `json:"address"`
	Port     int      `json:"port"`

	// Auth / identity
	UUID     string `json:"uuid,omitempty"`     // vless / vmess
	Password string `json:"password,omitempty"` // trojan / shadowsocks
	Method   string `json:"method,omitempty"`   // shadowsocks cipher
	AlterID  int    `json:"alterId,omitempty"`  // vmess

	// Transport
	Network    string `json:"network,omitempty"`    // tcp / ws / grpc / http
	Security   string `json:"security,omitempty"`   // none / tls / reality
	SNI        string `json:"sni,omitempty"`        // tls server name
	ALPN       string `json:"alpn,omitempty"`       // comma separated
	Fingerprint string `json:"fingerprint,omitempty"` // utls fingerprint
	Flow       string `json:"flow,omitempty"`       // vless flow (xtls-rprx-vision)
	PublicKey  string `json:"publicKey,omitempty"`  // reality
	ShortID    string `json:"shortId,omitempty"`    // reality
	Path       string `json:"path,omitempty"`       // ws/http path
	Host       string `json:"host,omitempty"`       // ws/http host header
	ServiceName string `json:"serviceName,omitempty"` // grpc

	// QUIC-based protocols (hysteria / hysteria2 / tuic)
	Obfs          string `json:"obfs,omitempty"`          // hysteria2 salamander / hysteria obfs
	ObfsPassword  string `json:"obfsPassword,omitempty"`  // hysteria2 obfs password
	UpMbps        int    `json:"upMbps,omitempty"`        // hysteria up bandwidth
	DownMbps      int    `json:"downMbps,omitempty"`      // hysteria down bandwidth
	Congestion    string `json:"congestion,omitempty"`    // tuic congestion control (bbr/cubic/new_reno)
	UDPRelayMode  string `json:"udpRelayMode,omitempty"`  // tuic udp relay mode (native/quic)

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
}

// RoutingRule is one user-defined routing entry: match traffic by domain, IP
// or process (application) and send it through the proxy, direct, or block it.
type RoutingRule struct {
	Type   string `json:"type"`   // "domain" | "ip" | "process"
	Value  string `json:"value"`  // domain suffix, IP/CIDR, or process name
	Action string `json:"action"` // "proxy" | "direct" | "block"
}

// AppSettings is the persisted user configuration.
type AppSettings struct {
	// --- Connection ---
	Core            Core        `json:"core"`
	ActiveProfileID string      `json:"activeProfileId"`
	RoutingMode     RoutingMode `json:"routingMode"`
	// DNS server used inside the tunnel.
	DNS string `json:"dns"`
	// TunName is the name of the WinTun adapter both cores create. Empty means
	// the built-in default ("TomorrowTun").
	TunName string `json:"tunName"`
	// Stack is the sing-box TUN network stack: "gvisor" (default) or "system".
	Stack string `json:"stack"`
	// MTU of the TUN interface; 0 means the core default (usually 9000/1500).
	MTU int `json:"mtu"`
	// Rules are user-defined domain/ip/app routing overrides.
	Rules []RoutingRule `json:"rules"`

	// --- Application ---
	// AutoConnect connects to the last active profile on launch.
	AutoConnect bool `json:"autoConnect"`
	// LaunchAtStartup registers a Task Scheduler task to start with Windows.
	LaunchAtStartup bool `json:"launchAtStartup"`

	// --- Appearance ---
	// Theme is the base preset id ("graphite" / "midnight" / "coal").
	Theme string `json:"theme"`
	// Accent is a hex color id from the palette ("indigo", "teal", ...).
	Accent string `json:"accent"`
	// Font is the UI font id ("inter" / "mono" / "geist").
	Font string `json:"font"`
	// Radius is the corner rounding preset ("sharp" / "soft" / "round").
	Radius string `json:"radius"`
}

// TunInterfaceName returns the configured TUN adapter name, falling back to the
// built-in default when unset. This is the single source of truth shared by the
// sing-box config, the xray route setup, and the traffic-stats matcher.
func (s AppSettings) TunInterfaceName() string {
	if s.TunName != "" {
		return s.TunName
	}
	return DefaultTunName
}

// DefaultTunName is the built-in WinTun adapter name.
const DefaultTunName = "TomorrowTun"

// DefaultSettings returns the out-of-the-box configuration. sing-box is the
// default core as requested.
func DefaultSettings() AppSettings {
	return AppSettings{
		Core:        CoreSingBox,
		RoutingMode: RoutingRules,
		AutoConnect: false,
		DNS:         "1.1.1.1",
		TunName:     DefaultTunName,
		Stack:       "gvisor",
		MTU:         0,
		Theme:       "graphite",
		Accent:      "indigo",
		Font:        "inter",
		Radius:      "soft",
	}
}

// Stats holds live traffic counters (cumulative bytes for the session and the
// last measured speed in bytes/second).
type Stats struct {
	Upload      uint64 `json:"upload"`
	Download    uint64 `json:"download"`
	UploadSpeed uint64 `json:"uploadSpeed"`
	DownloadSpeed uint64 `json:"downloadSpeed"`
}

// Status is the full connection snapshot pushed to the UI.
type Status struct {
	State         ConnState `json:"state"`
	Core          Core      `json:"core"`
	ActiveProfile *Profile  `json:"activeProfile"`
	Error         string    `json:"error,omitempty"`
	Stats         Stats     `json:"stats"`
	// ConnectedAt is a unix millisecond timestamp, 0 when not connected.
	ConnectedAt int64 `json:"connectedAt"`
}
