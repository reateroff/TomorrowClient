// Package model holds the data types shared between the Go backend and the
// React frontend (they are exported through the Wails bindings).
package model

// Core names the proxy engine handling the traffic. sing-box is the only one:
// it is linked into the app and runs in-process with a native TUN inbound and
// auto_route, managing the system routing table itself. The type is kept so the
// UI and status snapshots can still report which core is running.
type Core string

const CoreSingBox Core = "sing-box"

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
	Network     string `json:"network,omitempty"`     // tcp / ws / grpc / http
	Security    string `json:"security,omitempty"`    // none / tls / reality
	SNI         string `json:"sni,omitempty"`         // tls server name
	ALPN        string `json:"alpn,omitempty"`        // comma separated
	Fingerprint string `json:"fingerprint,omitempty"` // utls fingerprint
	Flow        string `json:"flow,omitempty"`        // vless flow (xtls-rprx-vision)
	PublicKey   string `json:"publicKey,omitempty"`   // reality
	ShortID     string `json:"shortId,omitempty"`     // reality
	Path        string `json:"path,omitempty"`        // ws/http path
	Host        string `json:"host,omitempty"`        // ws/http host header
	ServiceName string `json:"serviceName,omitempty"` // grpc

	// QUIC-based protocols (hysteria / hysteria2 / tuic)
	Obfs         string `json:"obfs,omitempty"`         // hysteria2 salamander / hysteria obfs
	ObfsPassword string `json:"obfsPassword,omitempty"` // hysteria2 obfs password
	UpMbps       int    `json:"upMbps,omitempty"`       // hysteria up bandwidth
	DownMbps     int    `json:"downMbps,omitempty"`     // hysteria down bandwidth
	Congestion   string `json:"congestion,omitempty"`   // tuic congestion control (bbr/cubic/new_reno)
	UDPRelayMode string `json:"udpRelayMode,omitempty"` // tuic udp relay mode (native/quic)

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
	Type   string `json:"type"`   // "domain" | "ip" | "process"
	Value  string `json:"value"`  // domain suffix, IP/CIDR, or process name
	Action string `json:"action"` // "proxy" | "direct" | "block"
	Icon   string `json:"icon"`   // process rules only: PNG data URL of the app icon
}

// SettingsVersion is bumped whenever a stored setting needs a one-time rewrite
// on load. See Store.migrate: recording the version is what keeps a migration
// from running twice and overwriting a later deliberate choice.
const SettingsVersion = 1

// AppSettings is the persisted user configuration.
type AppSettings struct {
	// SettingsVersion records which migrations have already been applied to
	// this file. Absent in files written before migrations existed, which
	// reads as 0.
	SettingsVersion int `json:"settingsVersion"`

	// --- Connection ---
	Core            Core   `json:"core"`
	ActiveProfileID string `json:"activeProfileId"`
	// DNS is the primary resolver, queried through the tunnel.
	DNS string `json:"dns"`
	// DNSFallback resolves the names that are routed around the tunnel
	// (LAN, direct rules). sing-box has no automatic failover between
	// resolvers, so this is a second resolver rather than a stand-in.
	DNSFallback string `json:"dnsFallback"`
	// TunName is the name of the WinTun adapter both cores create. Empty means
	// the built-in default ("TomorrowTun").
	TunName string `json:"tunName"`
	// Stack is the sing-box TUN network stack: "mixed" (default), "gvisor" or
	// "system".
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
	// NavPosition places the navigation tabs on the "left" (default) or "top".
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

// DefaultSettings returns the out-of-the-box configuration. sing-box is the
// default core as requested.
func DefaultSettings() AppSettings {
	return AppSettings{
		SettingsVersion: SettingsVersion,

		Core:        CoreSingBox,
		AutoConnect: false,
		DNS:         "1.1.1.1",
		DNSFallback: "8.8.8.8",
		TunName:     DefaultTunName,
		Stack:       "mixed",
		MTU:         0,
		Theme:       "graphite",
		Accent:      "indigo",
		Font:        "inter",
		Radius:      "soft",
		NavPosition: "left",
		Animation:   "rise",
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
	State         ConnState `json:"state"`
	Core          Core      `json:"core"`
	ActiveProfile *Profile  `json:"activeProfile"`
	Error         string    `json:"error,omitempty"`
	Stats         Stats     `json:"stats"`
	// ConnectedAt is a unix millisecond timestamp, 0 when not connected.
	ConnectedAt int64 `json:"connectedAt"`
}
