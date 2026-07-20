// Package singbox builds a sing-box configuration JSON from a model.Profile.
// sing-box runs as an external subprocess with a native TUN inbound and
// auto_route, so it manages the system routing table itself.
package singbox

import (
	"encoding/json"
	"strings"

	"TomorrowClient/internal/model"
)

// ClashAPIAddr is the local Clash-compatible API address used to poll traffic.
const ClashAPIAddr = "127.0.0.1:19090"

// TunName is the name given to the WinTun adapter created by sing-box.
const TunName = "TomorrowTun"

// Build renders a full sing-box config for the given profile and settings.
func Build(p model.Profile, s model.AppSettings) ([]byte, error) {
	cfg := map[string]any{
		"log": map[string]any{
			"level":     "warn",
			"timestamp": true,
		},
		"experimental": map[string]any{
			"clash_api": map[string]any{
				"external_controller": ClashAPIAddr,
			},
		},
		"dns": map[string]any{
			"servers": []any{
				map[string]any{"tag": "remote", "address": firstNonEmpty(s.DNS, "1.1.1.1"), "detour": "proxy"},
				map[string]any{"tag": "local", "address": "223.5.5.5", "detour": "direct"},
			},
			"final":    "remote",
			"strategy": "ipv4_only",
		},
		"inbounds": []any{
			map[string]any{
				"type":           "tun",
				"tag":            "tun-in",
				"interface_name": TunName,
				"address":        []any{"172.19.0.1/30"},
				"auto_route":     true,
				"strict_route":   true,
				"stack":          "gvisor",
				"sniff":          true,
			},
		},
		"outbounds": []any{
			outbound(p),
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": route(s),
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// route builds the routing rules block. In "rules" mode private ranges go
// direct; in "global" mode everything except DNS hijack goes through proxy.
func route(s model.AppSettings) map[string]any {
	// New-style rule actions (sing-box 1.11+) instead of legacy dns/block
	// special outbounds, so the config stays valid on 1.12/1.13.
	rules := []any{
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
	}
	if s.RoutingMode == model.RoutingRules {
		rules = append(rules,
			map[string]any{"ip_is_private": true, "outbound": "direct"},
		)
	}
	return map[string]any{
		"rules":                 rules,
		"final":                 "proxy",
		"auto_detect_interface": true,
	}
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
