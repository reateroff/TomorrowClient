// Package xray builds an xray-core configuration JSON from a model.Profile.
// xray runs as an external subprocess exposing a local SOCKS inbound; a
// separate tun2socks process forwards the WinTun adapter into that SOCKS port.
package xray

import (
	"encoding/json"
	"strings"

	"TomorrowClient/internal/model"
)

// SocksPort is the local SOCKS5 port xray listens on for tun2socks.
const SocksPort = 10808

// APIPort is the xray gRPC/stats API port (reserved for future stats use).
const APIPort = 10809

// Build renders a full xray config for the given profile.
func Build(p model.Profile, s model.AppSettings) ([]byte, error) {
	cfg := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{
			map[string]any{
				"tag":      "socks-in",
				"listen":   "127.0.0.1",
				"port":     SocksPort,
				"protocol": "socks",
				"settings": map[string]any{"udp": true},
				"sniffing": map[string]any{
					"enabled":      true,
					"destOverride": []any{"http", "tls"},
				},
			},
		},
		"outbounds": []any{
			outbound(p),
			map[string]any{"tag": "direct", "protocol": "freedom"},
			map[string]any{"tag": "block", "protocol": "blackhole"},
		},
		"routing": routing(s),
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// privateCIDRs are the RFC1918 / loopback / link-local ranges kept direct in
// "rules" mode. Listed explicitly so we don't depend on a bundled geoip.dat.
var privateCIDRs = []any{
	"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
	"169.254.0.0/16", "172.16.0.0/12", "192.168.0.0/16",
	"224.0.0.0/4", "240.0.0.0/4", "::1/128", "fc00::/7", "fe80::/10",
}

func routing(s model.AppSettings) map[string]any {
	rules := []any{}
	if s.RoutingMode == model.RoutingRules {
		rules = append(rules, map[string]any{
			"type":        "field",
			"ip":          privateCIDRs,
			"outboundTag": "direct",
		})
	}
	return map[string]any{
		"domainStrategy": "IPIfNonMatch",
		"rules":          rules,
	}
}

// outbound converts a profile into an xray outbound object.
func outbound(p model.Profile) map[string]any {
	out := map[string]any{"tag": "proxy"}
	switch p.Protocol {
	case model.ProtoVLESS:
		out["protocol"] = "vless"
		user := map[string]any{"id": p.UUID, "encryption": "none"}
		if p.Flow != "" {
			user["flow"] = p.Flow
		}
		out["settings"] = map[string]any{
			"vnext": []any{map[string]any{
				"address": p.Address, "port": p.Port,
				"users": []any{user},
			}},
		}
	case model.ProtoVMess:
		out["protocol"] = "vmess"
		out["settings"] = map[string]any{
			"vnext": []any{map[string]any{
				"address": p.Address, "port": p.Port,
				"users": []any{map[string]any{"id": p.UUID, "alterId": p.AlterID, "security": "auto"}},
			}},
		}
	case model.ProtoTrojan:
		out["protocol"] = "trojan"
		out["settings"] = map[string]any{
			"servers": []any{map[string]any{
				"address": p.Address, "port": p.Port, "password": p.Password,
			}},
		}
	case model.ProtoShadowsocks:
		out["protocol"] = "shadowsocks"
		out["settings"] = map[string]any{
			"servers": []any{map[string]any{
				"address": p.Address, "port": p.Port,
				"method": p.Method, "password": p.Password,
			}},
		}
	}
	out["streamSettings"] = streamSettings(p)
	return out
}

// streamSettings builds the transport + security block for an outbound.
func streamSettings(p model.Profile) map[string]any {
	ss := map[string]any{"network": firstNonEmpty(p.Network, "tcp")}

	switch p.Security {
	case "tls":
		ss["security"] = "tls"
		tls := map[string]any{"serverName": firstNonEmpty(p.SNI, p.Address)}
		if p.ALPN != "" {
			tls["alpn"] = strings.Split(p.ALPN, ",")
		}
		if p.Fingerprint != "" {
			tls["fingerprint"] = p.Fingerprint
		}
		ss["tlsSettings"] = tls
	case "reality":
		ss["security"] = "reality"
		ss["realitySettings"] = map[string]any{
			"serverName":  firstNonEmpty(p.SNI, p.Address),
			"publicKey":   p.PublicKey,
			"shortId":     p.ShortID,
			"fingerprint": firstNonEmpty(p.Fingerprint, "chrome"),
		}
	}

	switch p.Network {
	case "ws":
		ws := map[string]any{}
		if p.Path != "" {
			ws["path"] = p.Path
		}
		if p.Host != "" {
			ws["headers"] = map[string]any{"Host": p.Host}
		}
		ss["wsSettings"] = ws
	case "grpc":
		ss["grpcSettings"] = map[string]any{"serviceName": p.ServiceName}
	case "http":
		h := map[string]any{}
		if p.Host != "" {
			h["host"] = []any{p.Host}
		}
		if p.Path != "" {
			h["path"] = p.Path
		}
		ss["httpSettings"] = h
	}
	return ss
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
