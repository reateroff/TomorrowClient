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

// AppSettings is the persisted user configuration.
type AppSettings struct {
	Core            Core        `json:"core"`
	ActiveProfileID string      `json:"activeProfileId"`
	RoutingMode     RoutingMode `json:"routingMode"`
	AutoConnect     bool        `json:"autoConnect"`
	// DNS server used inside the tunnel.
	DNS string `json:"dns"`
}

// DefaultSettings returns the out-of-the-box configuration. sing-box is the
// default core as requested.
func DefaultSettings() AppSettings {
	return AppSettings{
		Core:        CoreSingBox,
		RoutingMode: RoutingRules,
		AutoConnect: false,
		DNS:         "1.1.1.1",
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
