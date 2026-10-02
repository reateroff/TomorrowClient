// Package singbox builds sing-box configuration JSON. It serves three uses:
//
//   - Build: the whole tunnel for a profile, run by sing-box or TodayCore (the
//     same JSON format; TodayCore adds XHTTP and VLESS Encryption).
//   - BuildFront: the tunnel in front of Xray or mihomo — TUN, DNS and routing
//     here, with the proxied traffic handed to that core's loopback SOCKS
//     listener.
//   - BuildProbe: a throwaway instance exposing one profile on a local port.
//
// The TUN inbound uses auto_route, so sing-box manages the routing table
// itself.
package singbox

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/model"
)

// ClashAPIAddr is the local Clash-compatible API address of the tunnel core.
const ClashAPIAddr = "127.0.0.1:19090"

// APISecret is a per-process credential; never persisted or sent to the frontend.
var APISecret string

// Flavor selects which sing-box build a config is written for.
type Flavor int

const (
	Upstream  Flavor = iota // github.com/sagernet/sing-box
	TodayCore               // TodayCore: sing-box plus XHTTP and VLESS Encryption
)

// Build renders a full config for the given profile and settings.
func Build(p model.Profile, s model.AppSettings, f Flavor) ([]byte, error) {
	proxy, err := outbound(p, f)
	if err != nil {
		return nil, err
	}
	cache := "sing-box.db"
	if f == TodayCore {
		cache = "todaycore.db"
	}
	cfg := tunnel(s, proxy, nil, cache)
	return json.MarshalIndent(cfg, "", "  ")
}

// CacheDir is where the tunnel keeps its cache file (downloaded rule-sets
// among it). Set once at start-up; empty disables the cache.
var CacheDir string

// SocksUpstream is the loopback SOCKS5 listener of a core that runs behind the
// front tunnel.
type SocksUpstream struct {
	Port     int
	Username string
	Password string
	// Server is the proxy server's host. Its name is resolved outside the
	// tunnel: the core behind the front looks it up through the system
	// resolver, which the tunnel hijacks, and answering that query through the
	// proxy would wait on the very connection it is needed for.
	Server string
	// Self is this executable's path. The core behind the front lives in the
	// same process; anything it sends that still reaches the tunnel goes out
	// directly instead of looping back into itself.
	Self string
}

// BuildFront renders the tunnel that fronts a SOCKS-speaking core.
func BuildFront(s model.AppSettings, up SocksUpstream) ([]byte, error) {
	proxy := map[string]any{
		"type":        "socks",
		"tag":         "proxy",
		"server":      "127.0.0.1",
		"server_port": up.Port,
		"version":     "5",
		"username":    up.Username,
		"password":    up.Password,
	}
	// Each fronted core keeps its own cache; no state bleeds into native boxes.
	cache := "front-tun.db"
	switch s.Core {
	case model.CoreXray:
		cache = "xray-tun.db"
	case model.CoreMihomo:
		cache = "mihomo-tun.db"
	}
	cfg := tunnel(s, proxy, &up, cache)
	return json.MarshalIndent(cfg, "", "  ")
}

// tunnel assembles the TUN, DNS and routing around a proxy outbound.
func tunnel(s model.AppSettings, proxy map[string]any, front *SocksUpstream, cacheFile string) map[string]any {
	// TUN honours the adapter name and MTU, but never a user stack override.
	// sing-box 1.15/TodayCore use the native Go stack when stack is omitted;
	// the old mixed/gvisor/system field is deprecated. Let each linked core
	// own its stack instead of pinning a legacy implementation.
	// No "sniff" field here: it is a legacy inbound option that sing-box
	// rejects outright. Sniffing is requested by the route rule instead.
	tun := map[string]any{
		"type":           "tun",
		"tag":            "tun-in",
		"interface_name": s.TunInterfaceName(),
		"address":        []any{"172.19.0.1/30"},
		"auto_route":     true,
		"strict_route":   s.StrictRoute,
	}
	if s.IPv6 {
		tun["address"] = []any{"172.19.0.1/30", "fdfe:dcba:9876::1/126"}
	}
	if s.MTU > 0 {
		tun["mtu"] = s.MTU
	}
	r := buildRoute(s, front)

	// Two resolvers: the primary answers through the tunnel and is the default,
	// the fallback answers the names that bypass it (LAN, direct rules, the
	// proxy server itself). sing-box does not fail over between servers on its
	// own, so these are two roles rather than a retry chain.
	dns := map[string]any{
		"servers": []any{
			dnsServer("remote", firstNonEmpty(s.DNS, "1.1.1.1"), "proxy"),
			dnsServer("local", firstNonEmpty(s.DNSFallback, "8.8.8.8"), ""),
		},
		"final":    "remote",
		"strategy": "ipv4_only",
	}
	if s.IPv6 {
		dns["strategy"] = "prefer_ipv4"
	}
	var dnsRules []any
	if front != nil && net.ParseIP(front.Server) == nil && front.Server != "" {
		dnsRules = append(dnsRules, map[string]any{"domain": []any{front.Server}, "server": "local"})
	}
	dnsRules = append(dnsRules, r.dnsRules...)
	if len(dnsRules) > 0 {
		dns["rules"] = dnsRules
	}

	outbounds := []any{proxy, map[string]any{"type": "direct", "tag": "direct"}}
	var endpoints []any
	// WireGuard is an endpoint, not an outbound, since sing-box 1.11.
	if proxy["type"] == "wireguard" {
		outbounds = outbounds[1:]
		endpoints = []any{proxy}
	}

	experimental := map[string]any{
		"clash_api": map[string]any{
			"external_controller": ClashAPIAddr,
			"secret":              APISecret,
		},
	}
	if CacheDir != "" {
		experimental["cache_file"] = map[string]any{
			"enabled": true,
			"path":    filepath.Join(CacheDir, cacheFile),
		}
	}

	cfg := map[string]any{
		"log": map[string]any{
			"level":     "info",
			"timestamp": true,
		},
		"experimental": experimental,
		"dns":          dns,
		"inbounds":     []any{tun},
		"outbounds":    outbounds,
		"route":        r.block,
	}
	if endpoints != nil {
		cfg["endpoints"] = endpoints
	}
	return cfg
}

// dnsServer renders a resolver in the typed server format. The setting may be
// a bare IP (plain UDP) or a URL: tcp://, tls://, https://, h3://, quic://.
func dnsServer(tag, addr, detour string) map[string]any {
	srv := map[string]any{"tag": tag, "type": "udp", "server": addr}
	if u, err := url.Parse(addr); err == nil && u.Scheme != "" && u.Host != "" {
		srv["type"] = map[string]string{"tcp": "tcp", "tls": "tls", "dot": "tls", "https": "https", "doh": "https", "h3": "h3", "quic": "quic", "doq": "quic", "udp": "udp"}[u.Scheme]
		if srv["type"] == "" {
			srv["type"] = "udp"
		}
		srv["server"] = u.Hostname()
		if port, err := strconv.Atoi(u.Port()); err == nil {
			srv["server_port"] = port
		}
		if (u.Scheme == "https" || u.Scheme == "doh" || u.Scheme == "h3") && u.Path != "" && u.Path != "/dns-query" {
			srv["path"] = u.Path
		}
	}
	if detour != "" {
		srv["detour"] = detour
	}
	return srv
}

// normalizeCIDR appends /32 to a bare IPv4 address so ip_cidr always gets a
// prefix length.
func normalizeCIDR(v string) string {
	if strings.Contains(v, "/") {
		return v
	}
	if strings.Contains(v, ":") {
		return v + "/128"
	}
	return v + "/32"
}

// outbound converts a profile into the matching sing-box outbound (or, for
// WireGuard, endpoint) object.
func outbound(p model.Profile, f Flavor) (map[string]any, error) {
	core := model.CoreSingBox
	if f == TodayCore {
		core = model.CoreTodayCore
	}
	if err := cores.Supports(core, p); err != nil {
		return nil, err
	}

	base := map[string]any{"tag": "proxy", "server": p.Address, "server_port": p.Port}

	switch p.Protocol {
	case model.ProtoVLESS:
		base["type"] = "vless"
		base["uuid"] = p.UUID
		if p.Flow != "" {
			base["flow"] = p.Flow
		}
		if p.PacketEncoding != "" {
			base["packet_encoding"] = p.PacketEncoding
		}
		if cores.HasEncryption(p) {
			base["encryption"] = p.Encryption
		}
	case model.ProtoVMess:
		base["type"] = "vmess"
		base["uuid"] = p.UUID
		base["alter_id"] = p.AlterID
		base["security"] = firstNonEmpty(p.Method, "auto")
		if p.PacketEncoding != "" {
			base["packet_encoding"] = p.PacketEncoding
		}
	case model.ProtoTrojan:
		base["type"] = "trojan"
		base["password"] = p.Password
	case model.ProtoShadowsocks:
		base["type"] = "shadowsocks"
		base["method"] = p.Method
		base["password"] = p.Password
		if p.Plugin != "" {
			base["plugin"] = map[string]string{"simple-obfs": "obfs-local"}[p.Plugin]
			if base["plugin"] == "" {
				base["plugin"] = p.Plugin
			}
			base["plugin_opts"] = p.PluginOpts
		}
	case model.ProtoHysteria2:
		base["type"] = "hysteria2"
		base["password"] = p.Password
		if p.Obfs != "" {
			base["obfs"] = map[string]any{
				"type":     p.Obfs,
				"password": p.ObfsPassword,
			}
		}
		if p.UpMbps > 0 {
			base["up_mbps"] = p.UpMbps
		}
		if p.DownMbps > 0 {
			base["down_mbps"] = p.DownMbps
		}
		if ports := hopPorts(p.Ports); len(ports) > 0 {
			base["server_ports"] = ports
		}
	case model.ProtoHysteria:
		base["type"] = "hysteria"
		base["auth_str"] = p.Password
		if p.Obfs != "" {
			base["obfs"] = p.Obfs
		}
		// Hysteria v1 refuses to start without both bandwidths; links often
		// leave them out, and these are what its own client defaults to.
		base["up_mbps"] = positiveOr(p.UpMbps, 10)
		base["down_mbps"] = positiveOr(p.DownMbps, 50)
	case model.ProtoTUIC:
		base["type"] = "tuic"
		base["uuid"] = p.UUID
		base["password"] = p.Password
		if p.Congestion != "" {
			base["congestion_control"] = p.Congestion
		}
		if p.UDPRelayMode != "" {
			base["udp_relay_mode"] = p.UDPRelayMode
		}
	case model.ProtoAnyTLS:
		base["type"] = "anytls"
		base["password"] = p.Password
	case model.ProtoSOCKS:
		base["type"] = "socks"
		base["version"] = "5"
		if p.Username != "" {
			base["username"] = p.Username
			base["password"] = p.Password
		}
	case model.ProtoHTTP:
		base["type"] = "http"
		if p.Username != "" {
			base["username"] = p.Username
			base["password"] = p.Password
		}
		if p.Path != "" {
			base["path"] = p.Path
		}
	case model.ProtoWireGuard:
		return wireguard(p)
	default:
		return nil, fmt.Errorf("unsupported protocol %q", p.Protocol)
	}

	if tls := tlsBlock(p); tls != nil {
		base["tls"] = tls
	}
	if tr := transportBlock(p); tr != nil {
		base["transport"] = tr
	}
	return base, nil
}

// wireguard renders a WireGuard endpoint with the profile as its single peer.
func wireguard(p model.Profile) (map[string]any, error) {
	peer := map[string]any{
		"address":     p.Address,
		"port":        positiveOr(p.Port, 51820),
		"public_key":  p.PublicKey,
		"allowed_ips": []any{"0.0.0.0/0", "::/0"},
	}
	if p.PreSharedKey != "" {
		peer["pre_shared_key"] = p.PreSharedKey
	}
	if r := reservedBytes(p.Reserved); r != nil {
		peer["reserved"] = r
	}
	ep := map[string]any{
		"type":        "wireguard",
		"tag":         "proxy",
		"address":     splitList(firstNonEmpty(p.LocalAddress, "172.16.0.2/32")),
		"private_key": p.PrivateKey,
		"peers":       []any{peer},
	}
	if p.MTU > 0 {
		ep["mtu"] = p.MTU
	}
	return ep, nil
}

// tlsBlock returns the sing-box tls object, or nil when TLS is off. The QUIC
// protocols always run TLS.
func tlsBlock(p model.Profile) map[string]any {
	switch p.Protocol {
	case model.ProtoHysteria, model.ProtoHysteria2, model.ProtoTUIC, model.ProtoAnyTLS:
	default:
		if p.Security != "tls" && p.Security != "reality" {
			return nil
		}
	}
	tls := map[string]any{
		"enabled":     true,
		"server_name": firstNonEmpty(p.SNI, p.Address),
	}
	if p.AllowInsecure {
		tls["insecure"] = true
	}
	if p.ALPN != "" {
		tls["alpn"] = splitList(p.ALPN)
	}
	fp := p.Fingerprint
	if p.Security == "reality" && fp == "" {
		fp = "chrome" // REALITY needs uTLS
	}
	if fp != "" && fp != "none" {
		tls["utls"] = map[string]any{"enabled": true, "fingerprint": fp}
	}
	if p.Security == "reality" {
		tls["reality"] = map[string]any{
			"enabled":    true,
			"public_key": p.PublicKey,
			"short_id":   p.ShortID,
		}
	}
	return tls
}

// transportBlock returns the v2ray transport object, or nil for plain tcp.
func transportBlock(p model.Profile) map[string]any {
	switch cores.Network(p) {
	case "ws":
		t := map[string]any{"type": "ws"}
		path, ed := earlyData(p.Path)
		if path != "" {
			t["path"] = path
		}
		if ed > 0 {
			t["max_early_data"] = ed
			t["early_data_header_name"] = "Sec-WebSocket-Protocol"
		}
		if p.Host != "" {
			t["headers"] = map[string]any{"Host": p.Host}
		}
		return t
	case "grpc":
		return map[string]any{"type": "grpc", "service_name": p.ServiceName}
	case "http":
		t := map[string]any{"type": "http"}
		if p.Host != "" {
			t["host"] = splitList(p.Host)
		}
		if p.Path != "" {
			t["path"] = p.Path
		}
		return t
	case "httpupgrade":
		t := map[string]any{"type": "httpupgrade"}
		if p.Host != "" {
			t["host"] = p.Host
		}
		if p.Path != "" {
			t["path"] = p.Path
		}
		return t
	case "quic":
		return map[string]any{"type": "quic"}
	case "xhttp":
		// TodayCore takes Xray's own field names, so the share link's extra
		// object merges in unchanged.
		t := map[string]any{}
		for k, v := range cores.Extra(p) {
			t[k] = v
		}
		t["type"] = "xhttp"
		if p.Host != "" {
			t["host"] = p.Host
		}
		if p.Path != "" {
			t["path"] = p.Path
		}
		if p.Mode != "" {
			t["mode"] = p.Mode
		}
		return t
	}
	return nil
}

// earlyData splits Xray's "?ed=2048" early-data marker off a WebSocket path.
func earlyData(path string) (string, int) {
	u, err := url.Parse(path)
	if err != nil {
		return path, 0
	}
	ed, err := strconv.Atoi(u.Query().Get("ed"))
	if err != nil || ed <= 0 {
		return path, 0
	}
	q := u.Query()
	q.Del("ed")
	u.RawQuery = q.Encode()
	return u.String(), ed
}

// hopPorts converts "20000-30000,443" into sing-box's ["20000:30000", "443:443"].
func hopPorts(s string) []any {
	var out []any
	for _, part := range splitList(s) {
		part := part.(string)
		if a, b, ok := strings.Cut(part, "-"); ok {
			out = append(out, a+":"+b)
		} else {
			out = append(out, part+":"+part)
		}
	}
	return out
}

// reservedBytes parses WireGuard's "1,2,3" reserved field.
func reservedBytes(s string) []any {
	var out []any
	for _, part := range splitList(s) {
		n, err := strconv.Atoi(part.(string))
		if err != nil || n < 0 || n > 255 {
			return nil
		}
		out = append(out, n)
	}
	if len(out) != 3 {
		return nil
	}
	return out
}

// splitList splits a comma separated value, dropping blanks.
func splitList(s string) []any {
	var out []any
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func positiveOr(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
