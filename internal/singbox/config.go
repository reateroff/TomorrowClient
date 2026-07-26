// Package singbox builds a sing-box configuration JSON from a model.Profile.
// sing-box is linked into the app and runs in-process; the config it gets here
// is the same JSON the standalone binary would take. The TUN inbound uses
// auto_route, so sing-box manages the system routing table itself.
package singbox

import (
	"encoding/json"
	"strings"

	"TomorrowClient/internal/model"
)

// ClashAPIAddr is the local Clash-compatible API address used to poll traffic.
const ClashAPIAddr = "127.0.0.1:19090"

// Build renders a full sing-box config for the given profile and settings.
func Build(p model.Profile, s model.AppSettings) ([]byte, error) {
	// TUN inbound honours the user-chosen adapter name, network stack and MTU.
	// No "sniff" field here: it is a legacy inbound option that sing-box 1.13
	// rejects outright. Sniffing is requested by the route rule instead.
	tun := map[string]any{
		"type":           "tun",
		"tag":            "tun-in",
		"interface_name": s.TunInterfaceName(),
		"address":        []any{"172.19.0.1/30"},
		"auto_route":     true,
		"strict_route":   true,
		"stack":          firstNonEmpty(s.Stack, "mixed"),
	}
	if s.MTU > 0 {
		tun["mtu"] = s.MTU
	}

	cfg := map[string]any{
		"log": map[string]any{
			"level":     "info",
			"timestamp": true,
		},
		"experimental": map[string]any{
			"clash_api": map[string]any{
				"external_controller": ClashAPIAddr,
			},
		},
		// Two resolvers: the primary answers through the tunnel and is the
		// default, the fallback answers the names that bypass it (LAN, direct
		// rules). sing-box does not fail over between servers on its own, so
		// these are two roles rather than a retry chain.
		"dns": map[string]any{
			"servers": []any{
				map[string]any{"tag": "remote", "address": firstNonEmpty(s.DNS, "1.1.1.1"), "detour": "proxy"},
				map[string]any{"tag": "local", "address": firstNonEmpty(s.DNSFallback, "8.8.8.8"), "detour": "direct"},
			},
			"final":    "remote",
			"strategy": "ipv4_only",
		},
		"inbounds": []any{
			tun,
		},
		// No "block" outbound: blocking is expressed as the "reject" rule action
		// (see userRule), and the legacy block outbound type is on its way out.
		"outbounds": []any{
			outbound(p),
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": route(s),
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// route builds the routing rules block. User-defined rules are applied first
// (domain / ip / process → proxy, direct or block), then the routing-mode
// defaults. In "rules" mode private ranges go direct; in "global" mode
// everything except DNS hijack goes through proxy.
func route(s model.AppSettings) map[string]any {
	// New-style rule actions (sing-box 1.11+) instead of legacy dns/block
	// special outbounds, so the config stays valid on 1.12/1.13.
	rules := []any{
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
	}

	// User-defined overrides take priority over the defaults.
	for _, r := range s.Rules {
		if rule := userRule(r); rule != nil {
			rules = append(rules, rule)
		}
	}

	// Local/LAN traffic (loopback, private ranges) always goes direct.
	rules = append(rules,
		map[string]any{"ip_is_private": true, "outbound": "direct"},
	)
	return map[string]any{
		"rules":                 rules,
		"final":                 "proxy",
		"auto_detect_interface": true,
	}
}

// userRule converts a single RoutingRule into a sing-box route rule. The "block"
// action maps to the new-style reject action; proxy/direct map to an outbound.
func userRule(r model.RoutingRule) map[string]any {
	if r.Value == "" {
		return nil
	}
	rule := map[string]any{}
	switch r.Type {
	case "domain":
		rule["domain_suffix"] = []any{r.Value}
	case "ip":
		rule["ip_cidr"] = []any{normalizeCIDR(r.Value)}
	case "process":
		rule["process_name"] = []any{r.Value}
	default:
		return nil
	}
	switch r.Action {
	case "block":
		rule["action"] = "reject"
	case "direct":
		rule["outbound"] = "direct"
	default: // proxy
		rule["outbound"] = "proxy"
	}
	return rule
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

// outbound converts a profile into the matching sing-box outbound object.
func outbound(p model.Profile) map[string]any {
	base := map[string]any{"tag": "proxy", "server": p.Address, "server_port": p.Port}

	switch p.Protocol {
	case model.ProtoVLESS:
		base["type"] = "vless"
		base["uuid"] = p.UUID
		if p.Flow != "" {
			base["flow"] = p.Flow
		}
	case model.ProtoVMess:
		base["type"] = "vmess"
		base["uuid"] = p.UUID
		base["alter_id"] = p.AlterID
		base["security"] = "auto"
	case model.ProtoTrojan:
		base["type"] = "trojan"
		base["password"] = p.Password
	case model.ProtoShadowsocks:
		base["type"] = "shadowsocks"
		base["method"] = p.Method
		base["password"] = p.Password
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
	case model.ProtoHysteria:
		base["type"] = "hysteria"
		base["auth_str"] = p.Password
		if p.Obfs != "" {
			base["obfs"] = p.Obfs
		}
		if p.UpMbps > 0 {
			base["up_mbps"] = p.UpMbps
		}
		if p.DownMbps > 0 {
			base["down_mbps"] = p.DownMbps
		}
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
	}

	if tls := tlsBlock(p); tls != nil {
		base["tls"] = tls
	}
	if tr := transportBlock(p); tr != nil {
		base["transport"] = tr
	}
	return base
}

// tlsBlock returns the sing-box tls object, or nil when TLS is off.
func tlsBlock(p model.Profile) map[string]any {
	if p.Security != "tls" && p.Security != "reality" {
		return nil
	}
	tls := map[string]any{
		"enabled":     true,
		"server_name": firstNonEmpty(p.SNI, p.Address),
	}
	if p.ALPN != "" {
		tls["alpn"] = strings.Split(p.ALPN, ",")
	}
	if p.Fingerprint != "" {
		tls["utls"] = map[string]any{"enabled": true, "fingerprint": p.Fingerprint}
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

// transportBlock returns the ws/grpc/http transport object, or nil for plain tcp.
func transportBlock(p model.Profile) map[string]any {
	switch p.Network {
	case "ws":
		t := map[string]any{"type": "ws"}
		if p.Path != "" {
			t["path"] = p.Path
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
			t["host"] = []any{p.Host}
		}
		if p.Path != "" {
			t["path"] = p.Path
		}
		return t
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
