// Package mihomo builds mihomo (Clash Meta) configuration. Like Xray, mihomo
// runs behind the sing-box front tunnel and only sees a loopback SOCKS
// listener. The config is written as JSON, which is valid YAML, so mihomo's
// own parser reads it unchanged.
package mihomo

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/model"
)

// ProxyName is the name the profile's proxy carries inside the config.
const ProxyName = "proxy"

// Local is the loopback listener a profile is exposed on.
type Local struct {
	Port     int
	Username string
	Password string
	// Interface binds mihomo's outbound sockets to this adapter (friendly
	// name), so they leave through the physical network rather than the tunnel.
	Interface string
}

// Build renders a config exposing p on a password-protected SOCKS5 listener.
func Build(p model.Profile, l Local) ([]byte, error) {
	proxy, err := Proxy(p)
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{
		"mode":              "rule",
		"log-level":         "info",
		"ipv6":              true,
		"allow-lan":         false,
		"bind-address":      "127.0.0.1",
		"find-process-mode": "off",
		"unified-delay":     true,
		"tcp-concurrent":    true,
		"geo-auto-update":   false,
		"profile":           map[string]any{"store-selected": false, "store-fake-ip": false},
		// Name resolution for the proxy server goes through the system
		// resolver; the front tunnel answers it outside the proxy.
		"dns": map[string]any{"enable": false},
		"listeners": []any{map[string]any{
			"name":   "in",
			"type":   "socks",
			"listen": "127.0.0.1",
			"port":   l.Port,
			"udp":    true,
			"users":  []any{map[string]any{"username": l.Username, "password": l.Password}},
		}},
		"proxies": []any{proxy},
		"rules":   []any{"MATCH," + ProxyName},
	}
	if l.Interface != "" {
		cfg["interface-name"] = l.Interface
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// Proxy converts a profile into a mihomo proxy mapping.
func Proxy(p model.Profile) (map[string]any, error) {
	if err := cores.Supports(model.CoreMihomo, p); err != nil {
		return nil, err
	}

	m := map[string]any{"name": ProxyName, "server": p.Address, "port": p.Port, "udp": true}
	tlsOn := p.Security == "tls" || p.Security == "reality"

	switch p.Protocol {
	case model.ProtoVLESS:
		m["type"] = "vless"
		m["uuid"] = p.UUID
		if p.Flow != "" {
			m["flow"] = p.Flow
		}
		if cores.HasEncryption(p) {
			m["encryption"] = p.Encryption
		}
		if p.PacketEncoding != "" {
			m["packet-encoding"] = p.PacketEncoding
		}
		streamOpts(m, p, tlsOn, "servername")
	case model.ProtoVMess:
		m["type"] = "vmess"
		m["uuid"] = p.UUID
		m["alterId"] = p.AlterID
		m["cipher"] = firstNonEmpty(p.Method, "auto")
		if p.PacketEncoding != "" {
			m["packet-encoding"] = p.PacketEncoding
		}
		streamOpts(m, p, tlsOn, "servername")
	case model.ProtoTrojan:
		m["type"] = "trojan"
		m["password"] = p.Password
		streamOpts(m, p, true, "sni")
		delete(m, "tls") // trojan is always TLS in mihomo
	case model.ProtoShadowsocks:
		m["type"] = "ss"
		m["cipher"] = p.Method
		m["password"] = p.Password
		if err := ssPlugin(m, p); err != nil {
			return nil, err
		}
	case model.ProtoHysteria2:
		m["type"] = "hysteria2"
		m["password"] = p.Password
		if p.Obfs != "" {
			m["obfs"] = p.Obfs
			m["obfs-password"] = p.ObfsPassword
		}
		if p.Ports != "" {
			m["ports"] = p.Ports
		}
		if p.UpMbps > 0 {
			m["up"] = strconv.Itoa(p.UpMbps)
		}
		if p.DownMbps > 0 {
			m["down"] = strconv.Itoa(p.DownMbps)
		}
		quicTLS(m, p)
	case model.ProtoHysteria:
		m["type"] = "hysteria"
		m["auth-str"] = p.Password
		m["up"] = strconv.Itoa(positiveOr(p.UpMbps, 10))
		m["down"] = strconv.Itoa(positiveOr(p.DownMbps, 50))
		if p.Obfs != "" {
			m["obfs"] = p.Obfs
		}
		quicTLS(m, p)
	case model.ProtoTUIC:
		m["type"] = "tuic"
		m["uuid"] = p.UUID
		m["password"] = p.Password
		if p.Congestion != "" {
			m["congestion-controller"] = p.Congestion
		}
		if p.UDPRelayMode != "" {
			m["udp-relay-mode"] = p.UDPRelayMode
		}
		quicTLS(m, p)
	case model.ProtoAnyTLS:
		m["type"] = "anytls"
		m["password"] = p.Password
		quicTLS(m, p)
		if p.Fingerprint != "" && p.Fingerprint != "none" {
			m["client-fingerprint"] = p.Fingerprint
		}
	case model.ProtoWireGuard:
		m["type"] = "wireguard"
		m["port"] = positiveOr(p.Port, 51820)
		m["private-key"] = p.PrivateKey
		m["public-key"] = p.PublicKey
		if p.PreSharedKey != "" {
			m["pre-shared-key"] = p.PreSharedKey
		}
		for _, a := range strings.Split(firstNonEmpty(p.LocalAddress, "172.16.0.2/32"), ",") {
			ip, _, _ := strings.Cut(strings.TrimSpace(a), "/")
			if parsed := net.ParseIP(ip); parsed != nil {
				if parsed.To4() != nil {
					m["ip"] = ip
				} else {
					m["ipv6"] = ip
				}
			}
		}
		if r := reservedBytes(p.Reserved); r != nil {
			m["reserved"] = r
		}
		if p.MTU > 0 {
			m["mtu"] = p.MTU
		}
	case model.ProtoSOCKS:
		m["type"] = "socks5"
		if p.Username != "" {
			m["username"], m["password"] = p.Username, p.Password
		}
		if p.Security == "tls" {
			quicTLS(m, p)
			m["tls"] = true
		}
	case model.ProtoHTTP:
		m["type"] = "http"
		if p.Username != "" {
			m["username"], m["password"] = p.Username, p.Password
		}
		if p.Security == "tls" {
			quicTLS(m, p)
			m["tls"] = true
		}
	default:
		return nil, fmt.Errorf("unsupported protocol %q", p.Protocol)
	}
	return m, nil
}

// streamOpts adds TLS/REALITY and transport options for the V2Ray family.
func streamOpts(m map[string]any, p model.Profile, tlsOn bool, sniKey string) {
	if tlsOn {
		m["tls"] = true
		if p.SNI != "" {
			m[sniKey] = p.SNI
		}
		if p.ALPN != "" {
			m["alpn"] = splitList(p.ALPN)
		}
		if p.AllowInsecure {
			m["skip-cert-verify"] = true
		}
		fp := p.Fingerprint
		if p.Security == "reality" && fp == "" {
			fp = "chrome"
		}
		if fp != "" && fp != "none" {
			m["client-fingerprint"] = fp
		}
		if p.Security == "reality" {
			m["reality-opts"] = map[string]any{"public-key": p.PublicKey, "short-id": p.ShortID}
		}
	}

	switch cores.Network(p) {
	case "tcp":
		if p.HeaderType == "http" {
			m["network"] = "http"
			opts := map[string]any{"path": []any{firstNonEmpty(p.Path, "/")}}
			if p.Host != "" {
				opts["headers"] = map[string]any{"Host": splitList(p.Host)}
			}
			m["http-opts"] = opts
		} else {
			m["network"] = "tcp"
		}
	case "ws", "httpupgrade":
		m["network"] = "ws"
		opts := map[string]any{}
		path, ed := earlyData(p.Path)
		opts["path"] = firstNonEmpty(path, "/")
		if ed > 0 {
			opts["max-early-data"] = ed
			opts["early-data-header-name"] = "Sec-WebSocket-Protocol"
		}
		if p.Host != "" {
			opts["headers"] = map[string]any{"Host": p.Host}
		}
		if cores.Network(p) == "httpupgrade" {
			opts["v2ray-http-upgrade"] = true
		}
		m["ws-opts"] = opts
	case "grpc":
		m["network"] = "grpc"
		m["grpc-opts"] = map[string]any{"grpc-service-name": p.ServiceName}
	case "http":
		m["network"] = "h2"
		opts := map[string]any{"path": firstNonEmpty(p.Path, "/")}
		if p.Host != "" {
			opts["host"] = splitList(p.Host)
		}
		m["h2-opts"] = opts
	case "xhttp":
		m["network"] = "xhttp"
		m["xhttp-opts"] = xhttpOpts(p)
	}
}

// xhttpExtraKeys maps Xray's XHTTP field names onto mihomo's.
var xhttpExtraKeys = map[string]string{
	"headers":              "headers",
	"noGRPCHeader":         "no-grpc-header",
	"xPaddingBytes":        "x-padding-bytes",
	"xPaddingObfsMode":     "x-padding-obfs-mode",
	"xPaddingKey":          "x-padding-key",
	"xPaddingHeader":       "x-padding-header",
	"xPaddingPlacement":    "x-padding-placement",
	"xPaddingMethod":       "x-padding-method",
	"uplinkHTTPMethod":     "uplink-http-method",
	"sessionIDPlacement":   "session-placement",
	"sessionIDKey":         "session-key",
	"sessionIDTable":       "session-table",
	"sessionIDLength":      "session-length",
	"seqPlacement":         "seq-placement",
	"seqKey":               "seq-key",
	"uplinkDataPlacement":  "uplink-data-placement",
	"uplinkDataKey":        "uplink-data-key",
	"uplinkChunkSize":      "uplink-chunk-size",
	"scMaxEachPostBytes":   "sc-max-each-post-bytes",
	"scMinPostsIntervalMs": "sc-min-posts-interval-ms",
}

var xmuxKeys = map[string]string{
	"maxConcurrency":   "max-concurrency",
	"maxConnections":   "max-connections",
	"cMaxReuseTimes":   "c-max-reuse-times",
	"hMaxRequestTimes": "h-max-request-times",
	"hMaxReusableSecs": "h-max-reusable-secs",
	"hKeepAlivePeriod": "h-keep-alive-period",
}

func xhttpOpts(p model.Profile) map[string]any {
	opts := map[string]any{"path": firstNonEmpty(p.Path, "/")}
	if p.Host != "" {
		opts["host"] = p.Host
	}
	if p.Mode != "" {
		opts["mode"] = p.Mode
	}
	for k, v := range cores.Extra(p) {
		if name, ok := xhttpExtraKeys[k]; ok {
			opts[name] = xhttpValue(name, v)
		}
		if k == "xmux" {
			if mux, ok := v.(map[string]any); ok {
				reuse := map[string]any{}
				for mk, mv := range mux {
					if name, ok := xmuxKeys[mk]; ok {
						reuse[name] = xhttpValue(name, mv)
					}
				}
				opts["reuse-settings"] = reuse
			}
		}
	}
	return opts
}

// xhttpValue converts Xray's range values — a number, "a-b" or {from, to} —
// into the "a-b" strings mihomo parses. Other values pass through.
func xhttpValue(name string, v any) any {
	switch name {
	case "headers", "no-grpc-header", "x-padding-obfs-mode", "h-keep-alive-period":
		if f, ok := v.(float64); ok && name == "h-keep-alive-period" {
			return int(f)
		}
		return v
	}
	switch t := v.(type) {
	case float64:
		return strconv.Itoa(int(t))
	case map[string]any:
		from, _ := t["from"].(float64)
		to, _ := t["to"].(float64)
		return fmt.Sprintf("%d-%d", int(from), int(to))
	}
	return v
}

// quicTLS adds the TLS fields of protocols that always run TLS.
func quicTLS(m map[string]any, p model.Profile) {
	if p.SNI != "" {
		m["sni"] = p.SNI
	}
	if p.ALPN != "" {
		m["alpn"] = splitList(p.ALPN)
	}
	if p.AllowInsecure {
		m["skip-cert-verify"] = true
	}
}

// ssPlugin translates a SIP003 plugin string into mihomo's plugin options.
func ssPlugin(m map[string]any, p model.Profile) error {
	if p.Plugin == "" {
		return nil
	}
	opts := map[string]string{}
	flags := map[string]bool{}
	for _, kv := range strings.Split(p.PluginOpts, ";") {
		if k, v, ok := strings.Cut(kv, "="); ok {
			opts[strings.TrimSpace(k)] = strings.TrimSpace(v)
		} else if kv = strings.TrimSpace(kv); kv != "" {
			flags[kv] = true
		}
	}
	switch p.Plugin {
	case "obfs-local", "simple-obfs", "obfs":
		m["plugin"] = "obfs"
		m["plugin-opts"] = map[string]any{"mode": firstNonEmpty(opts["obfs"], "http"), "host": opts["obfs-host"]}
	case "v2ray-plugin":
		m["plugin"] = "v2ray-plugin"
		m["plugin-opts"] = map[string]any{
			"mode": firstNonEmpty(opts["mode"], "websocket"),
			"host": opts["host"],
			"path": firstNonEmpty(opts["path"], "/"),
			"tls":  flags["tls"] || opts["tls"] == "true",
			"mux":  opts["mux"] != "0" && opts["mux"] != "false",
		}
	default:
		return fmt.Errorf("mihomo не поддерживает плагин %s", p.Plugin)
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
