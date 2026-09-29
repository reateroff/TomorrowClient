package link

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"strings"

	"TomorrowClient/internal/model"
)

// Encode renders a profile back into the share link Parse reads, in the same
// dialects: Xray's query format for the V2Ray family, v2rayN's JSON for VMess,
// SIP002 for Shadowsocks and each QUIC protocol's own scheme. It is what keeps
// a profile's link current after it is edited by hand. Returns "" for a
// protocol with no link form.
func Encode(p model.Profile) string {
	hostport := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	frag := ""
	if p.Name != "" {
		frag = "#" + url.PathEscape(p.Name)
	}

	switch p.Protocol {
	case model.ProtoVLESS:
		q := streamQuery(p)
		q.Set("encryption", firstNonEmpty(p.Encryption, "none"))
		set(q, "flow", p.Flow)
		return "vless://" + url.PathEscape(p.UUID) + "@" + hostport + "?" + q.Encode() + frag

	case model.ProtoTrojan:
		q := streamQuery(p)
		set(q, "flow", p.Flow)
		return "trojan://" + url.PathEscape(p.Password) + "@" + hostport + "?" + q.Encode() + frag

	case model.ProtoVMess:
		v := map[string]any{
			"v": "2", "ps": p.Name, "add": p.Address, "port": strconv.Itoa(p.Port),
			"id": p.UUID, "aid": strconv.Itoa(p.AlterID), "scy": firstNonEmpty(p.Method, "auto"),
			"net": firstNonEmpty(p.Network, "tcp"), "type": firstNonEmpty(p.HeaderType, "none"),
			"host": p.Host, "path": p.Path, "tls": "", "sni": p.SNI, "alpn": p.ALPN, "fp": p.Fingerprint,
		}
		if p.Security == "tls" {
			v["tls"] = "tls"
		}
		switch p.Network {
		case "grpc":
			v["path"], v["type"] = p.ServiceName, firstNonEmpty(p.Mode, "gun")
		case "kcp":
			v["path"] = p.Seed
		case "xhttp":
			v["type"] = p.Mode
			if p.Extra != "" {
				v["extra"] = p.Extra
			}
		}
		if p.AllowInsecure {
			v["allowInsecure"] = 1
		}
		b, _ := json.Marshal(v)
		return "vmess://" + base64.StdEncoding.EncodeToString(b)

	case model.ProtoShadowsocks:
		user := base64.RawURLEncoding.EncodeToString([]byte(p.Method + ":" + p.Password))
		if strings.HasPrefix(p.Method, "2022-") {
			// 2022 keys are base64 already; SIP022 writes the userinfo plain.
			user = url.PathEscape(p.Method) + ":" + url.PathEscape(p.Password)
		}
		s := "ss://" + user + "@" + hostport
		if p.Plugin != "" {
			plugin := p.Plugin
			if p.PluginOpts != "" {
				plugin += ";" + p.PluginOpts
			}
			s += "/?plugin=" + url.QueryEscape(plugin)
		}
		return s + frag

	case model.ProtoHysteria2:
		q := url.Values{}
		set(q, "sni", p.SNI)
		set(q, "alpn", p.ALPN)
		set(q, "obfs", p.Obfs)
		set(q, "obfs-password", p.ObfsPassword)
		set(q, "mport", p.Ports)
		setInt(q, "upmbps", p.UpMbps)
		setInt(q, "downmbps", p.DownMbps)
		setBool(q, "insecure", p.AllowInsecure)
		return "hysteria2://" + url.PathEscape(p.Password) + "@" + hostport + "/" + query(q) + frag

	case model.ProtoHysteria:
		q := url.Values{}
		set(q, "auth", p.Password)
		set(q, "peer", p.SNI)
		set(q, "alpn", p.ALPN)
		set(q, "obfsParam", p.Obfs)
		setInt(q, "upmbps", p.UpMbps)
		setInt(q, "downmbps", p.DownMbps)
		setBool(q, "insecure", p.AllowInsecure)
		return "hysteria://" + hostport + query(q) + frag

	case model.ProtoTUIC:
		q := url.Values{}
		set(q, "sni", p.SNI)
		set(q, "alpn", p.ALPN)
		set(q, "congestion_control", p.Congestion)
		set(q, "udp_relay_mode", p.UDPRelayMode)
		setBool(q, "allow_insecure", p.AllowInsecure)
		return "tuic://" + url.PathEscape(p.UUID) + ":" + url.PathEscape(p.Password) + "@" + hostport + query(q) + frag

	case model.ProtoWireGuard:
		q := url.Values{}
		set(q, "publickey", p.PublicKey)
		set(q, "presharedkey", p.PreSharedKey)
		set(q, "address", p.LocalAddress)
		set(q, "reserved", p.Reserved)
		setInt(q, "mtu", p.MTU)
		return "wireguard://" + url.PathEscape(p.PrivateKey) + "@" + hostport + query(q) + frag

	case model.ProtoAnyTLS:
		q := url.Values{}
		set(q, "sni", p.SNI)
		set(q, "alpn", p.ALPN)
		set(q, "fp", p.Fingerprint)
		setBool(q, "insecure", p.AllowInsecure)
		return "anytls://" + url.PathEscape(p.Password) + "@" + hostport + query(q) + frag

	case model.ProtoSOCKS:
		auth := ""
		if p.Username != "" {
			auth = url.PathEscape(p.Username) + ":" + url.PathEscape(p.Password) + "@"
		}
		return "socks://" + auth + hostport + frag
	}
	return ""
}

// streamQuery writes the V2Ray-family transport and security parameters.
func streamQuery(p model.Profile) url.Values {
	q := url.Values{}
	q.Set("type", firstNonEmpty(p.Network, "tcp"))
	q.Set("security", firstNonEmpty(p.Security, "none"))
	set(q, "sni", p.SNI)
	set(q, "alpn", p.ALPN)
	set(q, "fp", p.Fingerprint)
	setBool(q, "allowInsecure", p.AllowInsecure)
	set(q, "pbk", p.PublicKey)
	set(q, "sid", p.ShortID)
	set(q, "spx", p.SpiderX)
	set(q, "host", p.Host)
	set(q, "headerType", p.HeaderType)
	set(q, "packetEncoding", p.PacketEncoding)
	switch p.Network {
	case "grpc":
		set(q, "serviceName", p.ServiceName)
		set(q, "mode", p.Mode)
	case "kcp":
		set(q, "seed", p.Seed)
	case "xhttp":
		set(q, "path", p.Path)
		set(q, "mode", p.Mode)
		set(q, "extra", p.Extra)
	default:
		set(q, "path", p.Path)
	}
	return q
}

func set(q url.Values, k, v string) {
	if v != "" {
		q.Set(k, v)
	}
}

func setInt(q url.Values, k string, v int) {
	if v != 0 {
		q.Set(k, strconv.Itoa(v))
	}
}

func setBool(q url.Values, k string, v bool) {
	if v {
		q.Set(k, "1")
	}
}

func query(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}
