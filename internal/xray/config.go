// Package xray builds Xray-core configuration JSON. Xray runs behind the
// sing-box front tunnel: it does not see the TUN adapter at all, only a
// loopback SOCKS listener the front hands proxied traffic to.
package xray

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/model"
)

// Local is the loopback listener a profile is exposed on.
type Local struct {
	Port     int
	Username string
	Password string
	// Interface binds the proxy's own sockets to this adapter (friendly name),
	// so they leave through the physical network rather than the tunnel.
	Interface string
	// LogLevel is Xray's loglevel; "none" silences the core.
	LogLevel string
}

// Build renders a config exposing p on a password-protected SOCKS5 listener.
func Build(p model.Profile, l Local) ([]byte, error) {
	proxy, err := Outbound(p, l.Interface)
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{
		"log": map[string]any{"loglevel": firstNonEmpty(l.LogLevel, "info")},
		"inbounds": []any{map[string]any{
			"tag":      "in",
			"listen":   "127.0.0.1",
			"port":     l.Port,
			"protocol": "socks",
			"settings": map[string]any{
				"auth":     "password",
				"accounts": []any{map[string]any{"user": l.Username, "pass": l.Password}},
				"udp":      true,
				"ip":       "127.0.0.1",
			},
		}},
		// The first outbound is the default route.
		"outbounds": []any{proxy, map[string]any{"tag": "direct", "protocol": "freedom"}},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// Outbound converts a profile into an Xray outbound object.
func Outbound(p model.Profile, iface string) (map[string]any, error) {
	if err := cores.Supports(model.CoreXray, p); err != nil {
		return nil, err
	}

	out := map[string]any{"tag": "proxy"}
	server := map[string]any{"address": p.Address, "port": p.Port}
	settings := func(extra map[string]any) map[string]any {
		for k, v := range server {
			extra[k] = v
		}
		return extra
	}

	switch p.Protocol {
	case model.ProtoVLESS:
		out["protocol"] = "vless"
		s := settings(map[string]any{"id": p.UUID, "encryption": "none"})
		if p.Flow != "" {
			s["flow"] = p.Flow
		}
		if cores.HasEncryption(p) {
			s["encryption"] = p.Encryption
		}
		out["settings"] = s
	case model.ProtoVMess:
		out["protocol"] = "vmess"
		out["settings"] = settings(map[string]any{"id": p.UUID, "security": firstNonEmpty(p.Method, "auto")})
	case model.ProtoTrojan:
		out["protocol"] = "trojan"
		out["settings"] = settings(map[string]any{"password": p.Password})
	case model.ProtoShadowsocks:
		out["protocol"] = "shadowsocks"
		out["settings"] = settings(map[string]any{"method": p.Method, "password": p.Password})
	case model.ProtoSOCKS:
		out["protocol"] = "socks"
		s := settings(map[string]any{})
		if p.Username != "" {
			s["user"], s["pass"] = p.Username, p.Password
		}
		out["settings"] = s
	case model.ProtoHTTP:
		out["protocol"] = "http"
		s := settings(map[string]any{})
		if p.Username != "" {
			s["user"], s["pass"] = p.Username, p.Password
		}
		out["settings"] = s
	case model.ProtoHysteria2:
		// Xray's "hysteria" protocol is Hysteria 2.
		out["protocol"] = "hysteria"
		out["settings"] = settings(map[string]any{"version": 2})
	case model.ProtoWireGuard:
		out["protocol"] = "wireguard"
		out["settings"] = wireguard(p)
	default:
		return nil, fmt.Errorf("unsupported protocol %q", p.Protocol)
	}

	stream := streamSettings(p)
	if iface != "" {
		stream["sockopt"] = map[string]any{"interface": iface}
	}
	if len(stream) > 0 {
		out["streamSettings"] = stream
	}
	return out, nil
}

func wireguard(p model.Profile) map[string]any {
	peer := map[string]any{
		"publicKey": p.PublicKey,
		"endpoint":  net.JoinHostPort(p.Address, strconv.Itoa(positiveOr(p.Port, 51820))),
	}
	if p.PreSharedKey != "" {
		peer["preSharedKey"] = p.PreSharedKey
	}
	s := map[string]any{
		"secretKey":   p.PrivateKey,
		"address":     splitList(firstNonEmpty(p.LocalAddress, "172.16.0.2/32")),
		"peers":       []any{peer},
		"noKernelTun": true,
	}
	if p.MTU > 0 {
		s["mtu"] = p.MTU
	}
	if r := reservedBytes(p.Reserved); r != nil {
		s["reserved"] = r
	}
	return s
}

// streamSettings renders transport and security. Protocols with a transport of
// their own (WireGuard) and the plain proxy protocols get none.
func streamSettings(p model.Profile) map[string]any {
	st := map[string]any{}
	switch p.Protocol {
	case model.ProtoWireGuard, model.ProtoSOCKS, model.ProtoHTTP:
		return st
	case model.ProtoHysteria2:
		st["network"] = "hysteria"
		st["hysteriaSettings"] = map[string]any{"version": 2, "auth": p.Password}
		st["security"] = "tls"
		st["tlsSettings"] = tlsSettings(p, []string{"h3"})
		if p.Obfs == "salamander" {
			st["finalmask"] = map[string]any{"udp": []any{map[string]any{
				"type":     "salamander",
				"settings": map[string]any{"password": p.ObfsPassword},
			}}}
		}
		return st
	}

	switch n := cores.Network(p); n {
	case "tcp":
		st["network"] = "raw"
		if p.HeaderType == "http" {
			req := map[string]any{"path": []any{firstNonEmpty(p.Path, "/")}}
			if p.Host != "" {
				req["headers"] = map[string]any{"Host": splitList(p.Host)}
			}
			st["rawSettings"] = map[string]any{"header": map[string]any{"type": "http", "request": req}}
		}
	case "ws":
		st["network"] = "ws"
		st["wsSettings"] = map[string]any{"path": p.Path, "host": p.Host}
	case "httpupgrade":
		st["network"] = "httpupgrade"
		st["httpupgradeSettings"] = map[string]any{"path": p.Path, "host": p.Host}
	case "grpc":
		st["network"] = "grpc"
		st["grpcSettings"] = map[string]any{"serviceName": p.ServiceName, "multiMode": p.Mode == "multi"}
	case "xhttp":
		st["network"] = "xhttp"
		x := map[string]any{"path": p.Path, "host": p.Host, "mode": firstNonEmpty(p.Mode, "auto")}
		if p.Extra != "" && json.Valid([]byte(p.Extra)) {
			x["extra"] = json.RawMessage(p.Extra)
		}
		st["xhttpSettings"] = x
	case "kcp":
		st["network"] = "kcp"
		k := map[string]any{}
		if p.Seed != "" {
			k["seed"] = p.Seed
		}
		if p.HeaderType != "" && p.HeaderType != "none" {
			k["header"] = map[string]any{"type": p.HeaderType}
		}
		st["kcpSettings"] = k
	}

	switch p.Security {
	case "tls":
		st["security"] = "tls"
		st["tlsSettings"] = tlsSettings(p, nil)
	case "reality":
		st["security"] = "reality"
		r := map[string]any{
			"serverName":  firstNonEmpty(p.SNI, p.Address),
			"fingerprint": firstNonEmpty(p.Fingerprint, "chrome"),
			"publicKey":   p.PublicKey,
			"shortId":     p.ShortID,
		}
		if p.SpiderX != "" {
			r["spiderX"] = p.SpiderX
		}
		st["realitySettings"] = r
	}
	return st
}

func tlsSettings(p model.Profile, defaultALPN []string) map[string]any {
	t := map[string]any{"serverName": firstNonEmpty(p.SNI, p.Address)}
	if p.ALPN != "" {
		t["alpn"] = splitList(p.ALPN)
	} else if defaultALPN != nil {
		t["alpn"] = defaultALPN
	}
	if p.Fingerprint != "" && p.Fingerprint != "none" {
		t["fingerprint"] = p.Fingerprint
	}
	return t
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
