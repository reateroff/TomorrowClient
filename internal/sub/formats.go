package sub

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"

	"TomorrowClient/internal/link"
	"TomorrowClient/internal/model"
)

// Parse turns a subscription body — or anything pasted into the app — into
// profiles. Panels pick the format by User-Agent, so a body may be any of:
//
//   - share links, one per line, plain or base64 of the list;
//   - a sing-box config (JSON with "outbounds" / "endpoints");
//   - a Clash / mihomo config (YAML with "proxies").
//
// Entries that cannot be read are skipped; it is an error only when nothing
// is left.
func Parse(body string) ([]model.Profile, error) {
	var out []model.Profile
	var lastErr error
	for _, l := range ExtractLinks(body) {
		p, err := link.Parse(l)
		if err != nil {
			lastErr = err
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		trimmed := strings.TrimSpace(body)
		if strings.HasPrefix(trimmed, "{") {
			out = parseSingBox(trimmed)
		} else if strings.Contains(body, "proxies:") {
			out = parseClash(trimmed)
		}
	}
	for i := range out {
		if out[i].ID == "" {
			out[i].ID = uuid.NewString()
		}
		if out[i].Name == "" {
			out[i].Name = out[i].Address
		}
	}
	if len(out) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("no supported servers found")
	}
	return out, nil
}

// --- Clash / mihomo YAML ---

func parseClash(body string) []model.Profile {
	var doc struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if yaml.Unmarshal([]byte(body), &doc) != nil {
		return nil
	}
	var out []model.Profile
	for _, m := range doc.Proxies {
		if p, ok := clashProxy(obj(m)); ok {
			p.Raw = rawJSON(m)
			out = append(out, p)
		}
	}
	return out
}

func clashProxy(m obj) (model.Profile, bool) {
	p := model.Profile{
		Name:          m.str("name"),
		Address:       m.str("server"),
		Port:          m.int("port"),
		SNI:           firstNonEmpty(m.str("servername"), m.str("sni")),
		ALPN:          strings.Join(m.strs("alpn"), ","),
		Fingerprint:   m.str("client-fingerprint"),
		AllowInsecure: m.bool("skip-cert-verify"),
		Security:      "none",
	}
	if m.bool("tls") {
		p.Security = "tls"
	}
	if r := m.obj("reality-opts"); r != nil {
		p.Security = "reality"
		p.PublicKey, p.ShortID = r.str("public-key"), r.str("short-id")
	}

	switch m.str("type") {
	case "ss":
		p.Protocol = model.ProtoShadowsocks
		p.Method, p.Password = m.str("cipher"), m.str("password")
		opts := m.obj("plugin-opts")
		switch m.str("plugin") {
		case "obfs":
			p.Plugin = "obfs-local"
			p.PluginOpts = "obfs=" + firstNonEmpty(opts.str("mode"), "http")
			if h := opts.str("host"); h != "" {
				p.PluginOpts += ";obfs-host=" + h
			}
		case "v2ray-plugin":
			p.Plugin = "v2ray-plugin"
			parts := []string{"mode=" + firstNonEmpty(opts.str("mode"), "websocket")}
			if h := opts.str("host"); h != "" {
				parts = append(parts, "host="+h)
			}
			if path := opts.str("path"); path != "" {
				parts = append(parts, "path="+path)
			}
			if opts.bool("tls") {
				parts = append(parts, "tls")
			}
			p.PluginOpts = strings.Join(parts, ";")
		case "":
		default:
			return p, false
		}
	case "vmess":
		p.Protocol = model.ProtoVMess
		p.UUID, p.AlterID, p.Method = m.str("uuid"), m.int("alterId"), m.str("cipher")
		p.PacketEncoding = m.str("packet-encoding")
		clashTransport(&p, m)
	case "vless":
		p.Protocol = model.ProtoVLESS
		p.UUID, p.Flow = m.str("uuid"), m.str("flow")
		p.PacketEncoding = m.str("packet-encoding")
		if enc := m.str("encryption"); enc != "" && enc != "none" {
			p.Encryption = enc
		}
		clashTransport(&p, m)
	case "trojan":
		p.Protocol = model.ProtoTrojan
		p.Password = m.str("password")
		if p.Security == "none" {
			p.Security = "tls"
		}
		clashTransport(&p, m)
	case "hysteria2", "hy2":
		p.Protocol = model.ProtoHysteria2
		p.Password = m.str("password")
		p.Obfs, p.ObfsPassword = m.str("obfs"), m.str("obfs-password")
		p.Ports = m.str("ports")
		p.UpMbps, p.DownMbps = mbps(m.str("up")), mbps(m.str("down"))
		p.Network, p.Security = "udp", "tls"
	case "hysteria":
		p.Protocol = model.ProtoHysteria
		p.Password = firstNonEmpty(m.str("auth-str"), m.str("auth"))
		p.Obfs = m.str("obfs")
		p.UpMbps, p.DownMbps = mbps(m.str("up")), mbps(m.str("down"))
		p.Network, p.Security = "udp", "tls"
	case "tuic":
		p.Protocol = model.ProtoTUIC
		p.UUID, p.Password = m.str("uuid"), m.str("password")
		p.Congestion, p.UDPRelayMode = m.str("congestion-controller"), m.str("udp-relay-mode")
		p.Network, p.Security = "udp", "tls"
	case "anytls":
		p.Protocol = model.ProtoAnyTLS
		p.Password = m.str("password")
		p.Security = "tls"
	case "wireguard":
		p.Protocol = model.ProtoWireGuard
		p.Network = "udp"
		p.PrivateKey = m.str("private-key")
		peer := m
		if peers := m.objs("peers"); len(peers) > 0 {
			peer = peers[0]
			p.Address, p.Port = firstNonEmpty(peer.str("server"), p.Address), firstNonZero(peer.int("port"), p.Port)
		}
		p.PublicKey, p.PreSharedKey = peer.str("public-key"), peer.str("pre-shared-key")
		p.Reserved = joinInts(peer.list("reserved"))
		var addrs []string
		if ip := m.str("ip"); ip != "" {
			addrs = append(addrs, withPrefix(ip, "/32"))
		}
		if ip := m.str("ipv6"); ip != "" {
			addrs = append(addrs, withPrefix(ip, "/128"))
		}
		p.LocalAddress = strings.Join(addrs, ",")
		p.MTU = m.int("mtu")
	case "socks5":
		p.Protocol = model.ProtoSOCKS
		p.Username, p.Password = m.str("username"), m.str("password")
	case "http":
		p.Protocol = model.ProtoHTTP
		p.Username, p.Password = m.str("username"), m.str("password")
	default:
		return p, false
	}
	return p, p.Address != ""
}

// clashTransport reads the V2Ray-family network options.
func clashTransport(p *model.Profile, m obj) {
	switch n := m.str("network"); n {
	case "", "tcp":
		p.Network = "tcp"
	case "ws":
		p.Network = "ws"
		o := m.obj("ws-opts")
		p.Path = o.str("path")
		p.Host = o.obj("headers").str("Host")
		if ed := o.int("max-early-data"); ed > 0 {
			sep := "?"
			if strings.Contains(p.Path, "?") {
				sep = "&"
			}
			p.Path += sep + "ed=" + strconv.Itoa(ed)
		}
		if o.bool("v2ray-http-upgrade") {
			p.Network = "httpupgrade"
		}
	case "grpc":
		p.Network = "grpc"
		p.ServiceName = m.obj("grpc-opts").str("grpc-service-name")
	case "h2":
		p.Network = "http"
		o := m.obj("h2-opts")
		p.Path = o.str("path")
		p.Host = strings.Join(o.strs("host"), ",")
	case "http":
		p.Network = "tcp"
		p.HeaderType = "http"
		o := m.obj("http-opts")
		p.Path = firstOf(o.strs("path"))
		p.Host = strings.Join(o.obj("headers").strs("Host"), ",")
	case "xhttp":
		p.Network = "xhttp"
		o := m.obj("xhttp-opts")
		p.Path, p.Host, p.Mode = o.str("path"), o.str("host"), o.str("mode")
	default:
		p.Network = n
	}
}

// --- sing-box JSON ---

func parseSingBox(body string) []model.Profile {
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
		Endpoints []map[string]any `json:"endpoints"`
	}
	if json.Unmarshal([]byte(body), &doc) != nil {
		return nil
	}
	var out []model.Profile
	for _, m := range append(doc.Outbounds, doc.Endpoints...) {
		if p, ok := singBoxOutbound(obj(m)); ok {
			p.Raw = rawJSON(m)
			out = append(out, p)
		}
	}
	return out
}

func singBoxOutbound(m obj) (model.Profile, bool) {
	p := model.Profile{
		Name:     m.str("tag"),
		Address:  m.str("server"),
		Port:     m.int("server_port"),
		Security: "none",
	}
	if tls := m.obj("tls"); tls.bool("enabled") {
		p.Security = "tls"
		p.SNI = tls.str("server_name")
		p.AllowInsecure = tls.bool("insecure")
		p.ALPN = strings.Join(tls.strs("alpn"), ",")
		if u := tls.obj("utls"); u.bool("enabled") {
			p.Fingerprint = u.str("fingerprint")
		}
		if r := tls.obj("reality"); r.bool("enabled") {
			p.Security = "reality"
			p.PublicKey, p.ShortID = r.str("public_key"), r.str("short_id")
		}
	}
	tr := m.obj("transport")
	switch tr.str("type") {
	case "":
		p.Network = "tcp"
	case "ws":
		p.Network = "ws"
		p.Path = tr.str("path")
		p.Host = tr.obj("headers").str("Host")
		if ed := tr.int("max_early_data"); ed > 0 {
			p.Path += "?ed=" + strconv.Itoa(ed)
		}
	case "grpc":
		p.Network, p.ServiceName = "grpc", tr.str("service_name")
	case "http":
		p.Network, p.Path = "http", tr.str("path")
		p.Host = strings.Join(tr.strs("host"), ",")
	case "httpupgrade":
		p.Network, p.Path, p.Host = "httpupgrade", tr.str("path"), tr.str("host")
	case "quic":
		p.Network = "quic"
	case "xhttp": // TodayCore
		p.Network, p.Path, p.Host, p.Mode = "xhttp", tr.str("path"), tr.str("host"), tr.str("mode")
	default:
		return p, false
	}

	switch m.str("type") {
	case "vless":
		p.Protocol = model.ProtoVLESS
		p.UUID, p.Flow, p.PacketEncoding = m.str("uuid"), m.str("flow"), m.str("packet_encoding")
		if enc := m.str("encryption"); enc != "" && enc != "none" {
			p.Encryption = enc
		}
	case "vmess":
		p.Protocol = model.ProtoVMess
		p.UUID, p.AlterID, p.Method = m.str("uuid"), m.int("alter_id"), m.str("security")
		p.PacketEncoding = m.str("packet_encoding")
	case "trojan":
		p.Protocol = model.ProtoTrojan
		p.Password = m.str("password")
	case "shadowsocks":
		p.Protocol = model.ProtoShadowsocks
		p.Method, p.Password = m.str("method"), m.str("password")
		p.Plugin, p.PluginOpts = m.str("plugin"), m.str("plugin_opts")
	case "hysteria2":
		p.Protocol = model.ProtoHysteria2
		p.Password = m.str("password")
		if o := m.obj("obfs"); o != nil {
			p.Obfs, p.ObfsPassword = o.str("type"), o.str("password")
		}
		p.UpMbps, p.DownMbps = m.int("up_mbps"), m.int("down_mbps")
		var ports []string
		for _, r := range m.strs("server_ports") {
			a, b, _ := strings.Cut(r, ":")
			if a == b || b == "" {
				ports = append(ports, a)
			} else {
				ports = append(ports, a+"-"+b)
			}
		}
		p.Ports = strings.Join(ports, ",")
		p.Network, p.Security = "udp", "tls"
	case "hysteria":
		p.Protocol = model.ProtoHysteria
		p.Password = firstNonEmpty(m.str("auth_str"), m.str("auth"))
		p.Obfs = m.str("obfs")
		p.UpMbps, p.DownMbps = m.int("up_mbps"), m.int("down_mbps")
		p.Network, p.Security = "udp", "tls"
	case "tuic":
		p.Protocol = model.ProtoTUIC
		p.UUID, p.Password = m.str("uuid"), m.str("password")
		p.Congestion, p.UDPRelayMode = m.str("congestion_control"), m.str("udp_relay_mode")
		p.Network, p.Security = "udp", "tls"
	case "anytls":
		p.Protocol = model.ProtoAnyTLS
		p.Password = m.str("password")
	case "socks":
		p.Protocol = model.ProtoSOCKS
		p.Username, p.Password = m.str("username"), m.str("password")
	case "http":
		p.Protocol = model.ProtoHTTP
		p.Username, p.Password, p.Path = m.str("username"), m.str("password"), m.str("path")
	case "wireguard":
		p.Protocol = model.ProtoWireGuard
		p.Network, p.Security = "udp", "none"
		p.PrivateKey = m.str("private_key")
		p.LocalAddress = strings.Join(m.strs("address"), ",")
		p.MTU = m.int("mtu")
		peers := m.objs("peers")
		if len(peers) == 0 {
			return p, false
		}
		peer := peers[0]
		p.Address, p.Port = peer.str("address"), peer.int("port")
		p.PublicKey, p.PreSharedKey = peer.str("public_key"), peer.str("pre_shared_key")
		p.Reserved = joinInts(peer.list("reserved"))
	default:
		return p, false // selector, urltest, direct, block, dns, …
	}
	return p, p.Address != ""
}

// rawJSON keeps an imported entry as indented JSON in Profile.Raw, so the
// server can be shown — and edited — in the form it arrived in.
func rawJSON(m map[string]any) string {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

// ParseEntry reads a single entry as edited by hand: a sing-box outbound or
// endpoint object, or a Clash proxy mapping, both as JSON.
func ParseEntry(text string) (model.Profile, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		return model.Profile{}, fmt.Errorf("некорректный JSON: %w", err)
	}
	var (
		p  model.Profile
		ok bool
	)
	// sing-box names the port server_port and the entry tag; Clash says port
	// and name.
	if _, sb := m["server_port"]; sb || m["type"] == "wireguard" && m["peers"] != nil && m["private_key"] != nil {
		p, ok = singBoxOutbound(obj(m))
	} else {
		p, ok = clashProxy(obj(m))
	}
	if !ok {
		return model.Profile{}, fmt.Errorf("не удалось разобрать сервер: неизвестный тип или нет адреса")
	}
	p.Raw = rawJSON(m)
	if p.Name == "" {
		p.Name = p.Address
	}
	return p, nil
}

// --- loosely typed access to decoded YAML/JSON ---

type obj map[string]any

func (m obj) str(k string) string {
	switch v := m[k].(type) {
	case string:
		return v
	case int, int64, float64, bool:
		return fmt.Sprint(v)
	}
	return ""
}

func (m obj) int(k string) int {
	switch v := m[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	}
	return 0
}

func (m obj) bool(k string) bool {
	switch v := m[k].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	}
	return false
}

func (m obj) obj(k string) obj {
	if v, ok := m[k].(map[string]any); ok {
		return v
	}
	return nil
}

func (m obj) list(k string) []any {
	v, _ := m[k].([]any)
	return v
}

func (m obj) objs(k string) []obj {
	var out []obj
	for _, v := range m.list(k) {
		if o, ok := v.(map[string]any); ok {
			out = append(out, o)
		}
	}
	return out
}

// strs reads a string or a list of strings.
func (m obj) strs(k string) []string {
	if s, ok := m[k].(string); ok && s != "" {
		return []string{s}
	}
	var out []string
	for _, v := range m.list(k) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func joinInts(vs []any) string {
	var parts []string
	for _, v := range vs {
		switch n := v.(type) {
		case int:
			parts = append(parts, strconv.Itoa(n))
		case float64:
			parts = append(parts, strconv.Itoa(int(n)))
		}
	}
	return strings.Join(parts, ",")
}

func withPrefix(ip, prefix string) string {
	if strings.Contains(ip, "/") {
		return ip
	}
	return ip + prefix
}

func mbps(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(s, "mbps"), "m"))
	n, _ := strconv.Atoi(s)
	return n
}

func firstOf(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonZero(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
