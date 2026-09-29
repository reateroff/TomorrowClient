// Package link parses proxy share links into a model.Profile: vless://,
// vmess://, trojan://, ss://, hysteria2:// (hy2://), hysteria:// (hy://),
// tuic://, wireguard:// (wg://), anytls:// and socks:// (socks5://).
//
// Parameter names follow what the common clients export — the Xray/v2rayN
// query format for the V2Ray family, and each QUIC protocol's own URI scheme —
// including their aliases, since subscriptions mix exporters freely.
package link

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"TomorrowClient/internal/model"
)

// Schemes lists every share-link prefix Parse understands.
var Schemes = []string{
	"vless://", "vmess://", "trojan://", "ss://",
	"hysteria2://", "hy2://", "hysteria://", "hy://", "tuic://",
	"wireguard://", "wg://", "anytls://", "socks://", "socks5://",
}

// IsLink reports whether s starts with a supported share-link scheme.
func IsLink(s string) bool {
	s = strings.TrimSpace(s)
	for _, scheme := range Schemes {
		if len(s) >= len(scheme) && strings.EqualFold(s[:len(scheme)], scheme) {
			return true
		}
	}
	return false
}

// Parse detects the scheme of a share link and dispatches to the matching
// parser. The returned profile always has a fresh ID and the original link
// stored in Raw.
func Parse(raw string) (model.Profile, error) {
	raw = strings.TrimSpace(raw)
	scheme, _, ok := strings.Cut(raw, "://")
	if !ok {
		return model.Profile{}, fmt.Errorf("unsupported link scheme")
	}
	var (
		p   model.Profile
		err error
	)
	switch strings.ToLower(scheme) {
	case "vless":
		p, err = parseVLESS(raw)
	case "vmess":
		p, err = parseVMess(raw)
	case "trojan":
		p, err = parseTrojan(raw)
	case "ss":
		p, err = parseShadowsocks(raw)
	case "hysteria2", "hy2":
		p, err = parseHysteria2(raw)
	case "hysteria", "hy":
		p, err = parseHysteria(raw)
	case "tuic":
		p, err = parseTUIC(raw)
	case "wireguard", "wg":
		p, err = parseWireGuard(raw)
	case "anytls":
		p, err = parseAnyTLS(raw)
	case "socks", "socks5":
		p, err = parseSOCKS(raw)
	default:
		return model.Profile{}, fmt.Errorf("unsupported link scheme")
	}
	if err != nil {
		return model.Profile{}, err
	}
	if p.Address == "" {
		return model.Profile{}, fmt.Errorf("%s: no server address", scheme)
	}
	p.ID = uuid.NewString()
	p.Raw = raw
	return p, nil
}

// nameFromFragment returns the URL fragment (#name) decoded, or a fallback.
func nameFromFragment(u *url.URL, fallback string) string {
	if u.Fragment != "" {
		return u.Fragment // url.Parse already unescaped it
	}
	return fallback
}

// parseURL parses a hierarchical share link and its port.
func parseURL(raw string) (*url.URL, int, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, 0, err
	}
	port, _ := strconv.Atoi(u.Port())
	return u, port, nil
}

// applyStream reads the V2Ray-family transport and security parameters that
// vless:// and trojan:// share (Xray's share-link format, with v2rayN's
// aliases).
func applyStream(p *model.Profile, q url.Values, defaultSecurity string) {
	p.Network = normalizeNetwork(firstNonEmpty(q.Get("type"), q.Get("network"), "tcp"))
	p.Security = strings.ToLower(firstNonEmpty(q.Get("security"), defaultSecurity))
	if p.Security == "xtls" {
		p.Security = "tls"
	}
	p.SNI = firstNonEmpty(q.Get("sni"), q.Get("peer"), q.Get("serverName"))
	p.ALPN = q.Get("alpn")
	p.Fingerprint = firstNonEmpty(q.Get("fp"), q.Get("fingerprint"))
	p.AllowInsecure = truthy(firstNonEmpty(q.Get("allowInsecure"), q.Get("insecure"), q.Get("allow_insecure")))
	p.PublicKey = firstNonEmpty(q.Get("pbk"), q.Get("publicKey"))
	p.ShortID = firstNonEmpty(q.Get("sid"), q.Get("shortId"))
	p.SpiderX = firstNonEmpty(q.Get("spx"), q.Get("spiderX"))
	p.Path = q.Get("path")
	p.Host = q.Get("host")
	p.HeaderType = firstNonEmpty(q.Get("headerType"), q.Get("header"))
	p.PacketEncoding = firstNonEmpty(q.Get("packetEncoding"), q.Get("packet-encoding"))

	switch p.Network {
	case "grpc":
		p.ServiceName = firstNonEmpty(q.Get("serviceName"), q.Get("path"))
		p.Path = ""
		p.Mode = q.Get("mode") // gun / multi
		p.Host = firstNonEmpty(q.Get("authority"), p.Host)
	case "kcp":
		p.Seed = firstNonEmpty(q.Get("seed"), q.Get("path"))
		p.Path = ""
	case "xhttp":
		p.Mode = q.Get("mode")
		if extra := q.Get("extra"); extra != "" && json.Valid([]byte(extra)) {
			p.Extra = extra
		}
	}
	cleanHeaderType(p)
}

// cleanHeaderType keeps a header type only where it means something: HTTP
// obfuscation over TCP, and the mKCP packet headers.
func cleanHeaderType(p *model.Profile) {
	switch {
	case p.HeaderType == "none",
		p.Network == "tcp" && p.HeaderType != "http",
		p.Network != "tcp" && p.Network != "kcp":
		p.HeaderType = ""
	}
}

// normalizeNetwork maps transport spellings onto the names the generators use.
func normalizeNetwork(n string) string {
	switch n = strings.ToLower(n); n {
	case "raw", "":
		return "tcp"
	case "websocket":
		return "ws"
	case "h2":
		return "http"
	case "splithttp":
		return "xhttp"
	case "mkcp":
		return "kcp"
	}
	return n
}

// parseVLESS handles vless://uuid@host:port?params#name
func parseVLESS(raw string) (model.Profile, error) {
	u, port, err := parseURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	q := u.Query()
	p := model.Profile{
		Protocol: model.ProtoVLESS,
		Address:  u.Hostname(),
		Port:     port,
		UUID:     u.User.Username(),
		Flow:     q.Get("flow"),
	}
	if enc := q.Get("encryption"); enc != "" && enc != "none" {
		p.Encryption = enc
	}
	applyStream(&p, q, "none")
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// vmessJSON is the v2rayN base64-encoded JSON payload of a vmess:// link.
// Numbers arrive as either strings or numbers depending on the exporter.
type vmessJSON struct {
	PS            string `json:"ps"`
	Add           string `json:"add"`
	Port          any    `json:"port"`
	ID            string `json:"id"`
	Aid           any    `json:"aid"`
	Scy           string `json:"scy"`
	Net           string `json:"net"`
	Type          string `json:"type"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	TLS           string `json:"tls"`
	SNI           string `json:"sni"`
	ALPN          string `json:"alpn"`
	FP            string `json:"fp"`
	AllowInsecure any    `json:"allowInsecure"`
	Insecure      any    `json:"insecure"`
	Mode          string `json:"mode"`
	Extra         any    `json:"extra"`
}

func parseVMess(raw string) (model.Profile, error) {
	payload := raw[len("vmess://"):]
	if i := strings.IndexByte(payload, '#'); i >= 0 {
		payload = payload[:i]
	}
	dec, err := decodeBase64(payload)
	if err != nil {
		return model.Profile{}, fmt.Errorf("vmess: %w", err)
	}
	var v vmessJSON
	if err := json.Unmarshal([]byte(dec), &v); err != nil {
		return model.Profile{}, fmt.Errorf("vmess json: %w", err)
	}
	p := model.Profile{
		Name:          firstNonEmpty(v.PS, v.Add),
		Protocol:      model.ProtoVMess,
		Address:       v.Add,
		Port:          toInt(v.Port),
		UUID:          v.ID,
		AlterID:       toInt(v.Aid),
		Method:        v.Scy,
		Network:       normalizeNetwork(v.Net),
		Security:      strings.ToLower(firstNonEmpty(v.TLS, "none")),
		SNI:           v.SNI,
		ALPN:          v.ALPN,
		Fingerprint:   v.FP,
		AllowInsecure: truthy(fmt.Sprint(v.AllowInsecure)) || truthy(fmt.Sprint(v.Insecure)),
		Host:          v.Host,
		Path:          v.Path,
		HeaderType:    v.Type,
	}
	if p.Security != "tls" && p.Security != "reality" {
		p.Security = "none"
	}
	switch p.Network {
	case "grpc":
		// v2rayN puts the service name in path and the mode in type.
		p.ServiceName, p.Path = p.Path, ""
		p.Mode, p.HeaderType = v.Type, ""
	case "kcp":
		p.Seed, p.Path = p.Path, ""
	case "xhttp":
		p.Mode, p.HeaderType = firstNonEmpty(v.Mode, v.Type), ""
		switch e := v.Extra.(type) {
		case string:
			if json.Valid([]byte(e)) {
				p.Extra = e
			}
		case map[string]any:
			if b, err := json.Marshal(e); err == nil {
				p.Extra = string(b)
			}
		}
	}
	cleanHeaderType(&p)
	return p, nil
}

func parseTrojan(raw string) (model.Profile, error) {
	u, port, err := parseURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	q := u.Query()
	p := model.Profile{
		Protocol: model.ProtoTrojan,
		Address:  u.Hostname(),
		Port:     port,
		Password: userinfo(u),
		Flow:     q.Get("flow"),
	}
	applyStream(&p, q, "tls")
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// parseShadowsocks handles both the SIP002 form
// ss://base64(method:pass)@host:port/?plugin=...#name (userinfo may also be
// plain or percent-encoded, as Shadowsocks 2022 links use) and the legacy
// ss://base64(method:pass@host:port)#name.
func parseShadowsocks(raw string) (model.Profile, error) {
	body := raw[len("ss://"):]
	name := ""
	if i := strings.Index(body, "#"); i >= 0 {
		name, _ = url.PathUnescape(body[i+1:])
		body = body[:i]
	}
	var query url.Values
	if i := strings.Index(body, "?"); i >= 0 {
		query, _ = url.ParseQuery(body[i+1:])
		body = body[:i]
	}
	body = strings.TrimSuffix(body, "/")

	var method, password, host string
	var port int

	if at := strings.LastIndex(body, "@"); at >= 0 {
		userinfo := body[:at]
		if dec, err := decodeBase64(userinfo); err == nil && strings.Contains(dec, ":") {
			userinfo = dec
		} else if un, err := url.PathUnescape(userinfo); err == nil {
			userinfo = un
		}
		method, password = splitColon(userinfo)
		host, port = splitHostPort(body[at+1:])
	} else {
		dec, err := decodeBase64(body)
		if err != nil {
			return model.Profile{}, fmt.Errorf("ss decode: %w", err)
		}
		at := strings.LastIndex(dec, "@")
		if at < 0 {
			return model.Profile{}, fmt.Errorf("ss: malformed link")
		}
		method, password = splitColon(dec[:at])
		host, port = splitHostPort(dec[at+1:])
	}

	p := model.Profile{
		Name:     firstNonEmpty(name, host),
		Protocol: model.ProtoShadowsocks,
		Address:  host,
		Port:     port,
		Method:   method,
		Password: password,
		Network:  "tcp",
		Security: "none",
	}
	// plugin=obfs-local;obfs=http;obfs-host=example.com
	if plugin := query.Get("plugin"); plugin != "" {
		name, opts, _ := strings.Cut(plugin, ";")
		p.Plugin, p.PluginOpts = name, opts
	}
	return p, nil
}

// parseHysteria2 handles hysteria2://auth@host:port/?params#name (also hy2://).
// The port part may be a hopping spec ("443,20000-30000"), which url.Parse
// refuses, so the authority is split by hand.
func parseHysteria2(raw string) (model.Profile, error) {
	u, host, ports, err := parseHopURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	q := u.Query()
	p := model.Profile{
		Protocol:      model.ProtoHysteria2,
		Address:       host,
		Password:      firstNonEmpty(userinfo(u), q.Get("auth"), q.Get("password")), // "user:pass" auth stays whole
		Network:       "udp",
		Security:      "tls",
		SNI:           firstNonEmpty(q.Get("sni"), q.Get("peer")),
		ALPN:          q.Get("alpn"),
		AllowInsecure: truthy(q.Get("insecure")),
		Obfs:          q.Get("obfs"),
		ObfsPassword:  q.Get("obfs-password"),
		UpMbps:        mbps(firstNonEmpty(q.Get("upmbps"), q.Get("up"))),
		DownMbps:      mbps(firstNonEmpty(q.Get("downmbps"), q.Get("down"))),
	}
	p.Port, p.Ports = firstPort(ports)
	if mport := q.Get("mport"); mport != "" {
		p.Ports = mport
	}
	if p.Port == 0 {
		p.Port = 443
	}
	p.Name = nameFromFragment(u, host)
	return p, nil
}

// parseHopURL parses scheme://userinfo@host:portspec/?query#frag, where
// portspec may list several ports and ranges.
func parseHopURL(raw string) (*url.URL, string, string, error) {
	scheme, rest, _ := strings.Cut(raw, "://")
	authority := rest
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		authority = rest[:i]
	}
	userPart, hostport := "", authority
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		userPart, hostport = authority[:at+1], authority[at+1:]
	}
	host, portSpec := hostport, ""
	if strings.HasPrefix(hostport, "[") {
		if end := strings.Index(hostport, "]"); end >= 0 {
			host = hostport[1:end]
			portSpec = strings.TrimPrefix(hostport[end+1:], ":")
		}
	} else if i := strings.LastIndex(hostport, ":"); i >= 0 {
		host, portSpec = hostport[:i], hostport[i+1:]
	}
	// Reparse with the port spec removed so url.Parse accepts it.
	clean := scheme + "://" + userPart + "placeholder" + rest[len(authority):]
	u, err := url.Parse(clean)
	if err != nil {
		return nil, "", "", err
	}
	return u, host, portSpec, nil
}

// firstPort splits "443,20000-30000" into the first single port and the full
// hopping spec. A spec that is just one port is no hopping at all.
func firstPort(spec string) (int, string) {
	if spec == "" {
		return 0, ""
	}
	parts := strings.Split(spec, ",")
	port := 0
	for _, part := range parts {
		a, _, _ := strings.Cut(part, "-")
		if n, err := strconv.Atoi(strings.TrimSpace(a)); err == nil {
			port = n
			break
		}
	}
	if len(parts) == 1 && !strings.Contains(spec, "-") {
		return port, ""
	}
	return port, spec
}

// parseHysteria handles the legacy Hysteria v1 hysteria://host:port?params#name
// (also hy://). Auth is carried in the auth/auth_str query parameter.
func parseHysteria(raw string) (model.Profile, error) {
	u, port, err := parseURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	if port == 0 {
		port = 443
	}
	q := u.Query()
	p := model.Profile{
		Protocol:      model.ProtoHysteria,
		Address:       u.Hostname(),
		Port:          port,
		Password:      firstNonEmpty(q.Get("auth"), q.Get("auth_str"), u.User.Username()),
		Network:       "udp",
		Security:      "tls",
		SNI:           firstNonEmpty(q.Get("sni"), q.Get("peer")),
		ALPN:          q.Get("alpn"),
		AllowInsecure: truthy(q.Get("insecure")),
		Obfs:          firstNonEmpty(q.Get("obfsParam"), q.Get("obfs")),
		UpMbps:        mbps(firstNonEmpty(q.Get("upmbps"), q.Get("up"))),
		DownMbps:      mbps(firstNonEmpty(q.Get("downmbps"), q.Get("down"))),
	}
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// parseTUIC handles tuic://uuid:password@host:port?params#name (TUIC v5).
func parseTUIC(raw string) (model.Profile, error) {
	u, port, err := parseURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	if port == 0 {
		port = 443
	}
	q := u.Query()
	pass, _ := u.User.Password()
	p := model.Profile{
		Protocol:      model.ProtoTUIC,
		Address:       u.Hostname(),
		Port:          port,
		UUID:          u.User.Username(),
		Password:      firstNonEmpty(pass, q.Get("password")),
		Network:       "udp",
		Security:      "tls",
		SNI:           firstNonEmpty(q.Get("sni"), q.Get("peer")),
		ALPN:          q.Get("alpn"),
		AllowInsecure: truthy(firstNonEmpty(q.Get("allow_insecure"), q.Get("insecure"), q.Get("allowInsecure"))),
		Congestion:    firstNonEmpty(q.Get("congestion_control"), q.Get("congestion"), q.Get("congestion-controller")),
		UDPRelayMode:  firstNonEmpty(q.Get("udp_relay_mode"), q.Get("udp-relay-mode")),
	}
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// parseWireGuard handles wireguard://privatekey@host:port?publickey=...
// &address=10.0.0.2/32&reserved=1,2,3&mtu=1280#name (also wg://), the format
// v2rayN and sing-box based clients export.
func parseWireGuard(raw string) (model.Profile, error) {
	u, port, err := parseURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	q := u.Query()
	p := model.Profile{
		Protocol:     model.ProtoWireGuard,
		Address:      u.Hostname(),
		Port:         port,
		PrivateKey:   firstNonEmpty(userinfo(u), q.Get("privatekey"), q.Get("secretKey"), q.Get("private-key")),
		PublicKey:    firstNonEmpty(q.Get("publickey"), q.Get("publicKey"), q.Get("public-key"), q.Get("peer")),
		PreSharedKey: firstNonEmpty(q.Get("presharedkey"), q.Get("preSharedKey"), q.Get("pre-shared-key"), q.Get("psk")),
		LocalAddress: firstNonEmpty(q.Get("address"), q.Get("ip"), q.Get("local_address")),
		Reserved:     q.Get("reserved"),
		MTU:          atoiDefault(q.Get("mtu"), 0),
		Network:      "udp",
	}
	if p.Port == 0 {
		p.Port = 51820
	}
	if p.PrivateKey == "" || p.PublicKey == "" {
		return model.Profile{}, fmt.Errorf("wireguard: key missing")
	}
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// parseAnyTLS handles anytls://password@host:port?sni=...&insecure=1#name.
func parseAnyTLS(raw string) (model.Profile, error) {
	u, port, err := parseURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	q := u.Query()
	p := model.Profile{
		Protocol:      model.ProtoAnyTLS,
		Address:       u.Hostname(),
		Port:          port,
		Password:      userinfo(u),
		Security:      "tls",
		SNI:           firstNonEmpty(q.Get("sni"), q.Get("peer")),
		ALPN:          q.Get("alpn"),
		Fingerprint:   q.Get("fp"),
		AllowInsecure: truthy(firstNonEmpty(q.Get("insecure"), q.Get("allowInsecure"))),
	}
	if p.Port == 0 {
		p.Port = 443
	}
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// parseSOCKS handles socks://[user:pass@]host:port#name (also socks5://). The
// userinfo is plain in some exporters and base64 of "user:pass" in v2rayN.
func parseSOCKS(raw string) (model.Profile, error) {
	u, port, err := parseURL(raw)
	if err != nil {
		return model.Profile{}, err
	}
	p := model.Profile{
		Protocol: model.ProtoSOCKS,
		Address:  u.Hostname(),
		Port:     port,
		Security: "none",
	}
	if u.User != nil {
		user := u.User.Username()
		pass, hasPass := u.User.Password()
		if !hasPass {
			if dec, err := decodeBase64(user); err == nil && strings.Contains(dec, ":") {
				user, pass = splitColon(dec)
			}
		}
		p.Username, p.Password = user, pass
	}
	if p.Port == 0 {
		p.Port = 1080
	}
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// --- helpers ---

// userinfo returns the whole userinfo — username, or "user:pass" when a
// password is present — unescaped.
func userinfo(u *url.URL) string {
	if u.User == nil {
		return ""
	}
	if pw, ok := u.User.Password(); ok {
		return u.User.Username() + ":" + pw
	}
	return u.User.Username()
}

func decodeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{
		base64.RawURLEncoding, base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("invalid base64")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// truthy reads the boolean spellings share links use.
func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// mbps reads a bandwidth value, tolerating a unit suffix ("100 Mbps").
func mbps(s string) int {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimSuffix(strings.TrimSuffix(s, "mbps"), "m")
	return atoiDefault(strings.TrimSpace(s), 0)
}

// atoiDefault parses s as an int, returning def on failure or empty input.
func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func splitColon(s string) (a, b string) {
	if i := strings.Index(s, ":"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func splitHostPort(s string) (host string, port int) {
	h, p, err := net.SplitHostPort(s)
	if err != nil {
		return strings.Trim(s, "[]"), 0
	}
	port, _ = strconv.Atoi(p)
	return h, port
}

// toInt converts a JSON value that may be a string or number to an int.
func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	case int:
		return t
	}
	return 0
}
