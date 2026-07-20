// Package link parses proxy share links (vless://, vmess://, trojan://, ss://)
// into a model.Profile.
package link

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"TomorrowClient/internal/model"
)

// Parse detects the scheme of a share link and dispatches to the matching
// parser. The returned profile always has a fresh ID and the original link
// stored in Raw.
func Parse(raw string) (model.Profile, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "vless://"):
		return parseVLESS(raw)
	case strings.HasPrefix(raw, "vmess://"):
		return parseVMess(raw)
	case strings.HasPrefix(raw, "trojan://"):
		return parseTrojan(raw)
	case strings.HasPrefix(raw, "ss://"):
		return parseShadowsocks(raw)
	default:
		return model.Profile{}, fmt.Errorf("unsupported link scheme")
	}
}

func newID() string { return uuid.NewString() }

// nameFromFragment returns the URL fragment (#name) decoded, or a fallback.
func nameFromFragment(u *url.URL, fallback string) string {
	if u.Fragment != "" {
		if dec, err := url.QueryUnescape(u.Fragment); err == nil {
			return dec
		}
		return u.Fragment
	}
	return fallback
}

// parseVLESS handles vless://uuid@host:port?params#name
func parseVLESS(raw string) (model.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return model.Profile{}, err
	}
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	p := model.Profile{
		ID:          newID(),
		Protocol:    model.ProtoVLESS,
		Address:     u.Hostname(),
		Port:        port,
		UUID:        u.User.Username(),
		Network:     firstNonEmpty(q.Get("type"), "tcp"),
		Security:    firstNonEmpty(q.Get("security"), "none"),
		SNI:         q.Get("sni"),
		ALPN:        q.Get("alpn"),
		Fingerprint: q.Get("fp"),
		Flow:        q.Get("flow"),
		PublicKey:   q.Get("pbk"),
		ShortID:     q.Get("sid"),
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		ServiceName: q.Get("serviceName"),
		Raw:         raw,
	}
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// vmessJSON is the standard base64-encoded JSON payload of a vmess:// link.
type vmessJSON struct {
	PS   string `json:"ps"`
	Add  string `json:"add"`
	Port any    `json:"port"`
	ID   string `json:"id"`
	Aid  any    `json:"aid"`
	Net  string `json:"net"`
	TLS  string `json:"tls"`
	SNI  string `json:"sni"`
	Host string `json:"host"`
	Path string `json:"path"`
	Type string `json:"type"`
}

func parseVMess(raw string) (model.Profile, error) {
	payload := strings.TrimPrefix(raw, "vmess://")
	dec, err := decodeBase64(payload)
	if err != nil {
		return model.Profile{}, fmt.Errorf("vmess: %w", err)
	}
	var v vmessJSON
	if err := json.Unmarshal([]byte(dec), &v); err != nil {
		return model.Profile{}, fmt.Errorf("vmess json: %w", err)
	}
	p := model.Profile{
		ID:       newID(),
		Name:     firstNonEmpty(v.PS, v.Add),
		Protocol: model.ProtoVMess,
		Address:  v.Add,
		Port:     toInt(v.Port),
		UUID:     v.ID,
		AlterID:  toInt(v.Aid),
		Network:  firstNonEmpty(v.Net, "tcp"),
		Security: firstNonEmpty(v.TLS, "none"),
		SNI:      v.SNI,
		Host:     v.Host,
		Path:     v.Path,
		Raw:      raw,
	}
	if p.Security == "" {
		p.Security = "none"
	}
	return p, nil
}

func parseTrojan(raw string) (model.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return model.Profile{}, err
	}
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	p := model.Profile{
		ID:          newID(),
		Protocol:    model.ProtoTrojan,
		Address:     u.Hostname(),
		Port:        port,
		Password:    u.User.Username(),
		Network:     firstNonEmpty(q.Get("type"), "tcp"),
		Security:    firstNonEmpty(q.Get("security"), "tls"),
		SNI:         q.Get("sni"),
		ALPN:        q.Get("alpn"),
		Fingerprint: q.Get("fp"),
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		Raw:         raw,
	}
	p.Name = nameFromFragment(u, u.Hostname())
	return p, nil
}

// parseShadowsocks handles both ss://base64(method:pass)@host:port and the
// SIP002 form ss://base64(method:pass@host:port).
func parseShadowsocks(raw string) (model.Profile, error) {
	body := strings.TrimPrefix(raw, "ss://")
	name := ""
	if i := strings.Index(body, "#"); i >= 0 {
		name, _ = url.QueryUnescape(body[i+1:])
		body = body[:i]
	}
	// Strip any query string (plugin params) — not supported yet.
	if i := strings.Index(body, "?"); i >= 0 {
		body = body[:i]
	}

	var method, password, host string
	var port int

	if at := strings.LastIndex(body, "@"); at >= 0 {
		// SIP002: userinfo may be base64(method:pass), host:port is plain.
		userinfo := body[:at]
		hostport := body[at+1:]
		if dec, err := decodeBase64(userinfo); err == nil && strings.Contains(dec, ":") {
			userinfo = dec
		}
		method, password = splitColon(userinfo)
		host, port = splitHostPort(hostport)
	} else {
		// Legacy: whole thing is base64(method:pass@host:port).
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
		ID:       newID(),
		Name:     firstNonEmpty(name, host),
		Protocol: model.ProtoShadowsocks,
		Address:  host,
		Port:     port,
		Method:   method,
		Password: password,
		Network:  "tcp",
		Security: "none",
		Raw:      raw,
	}
	return p, nil
}

// --- helpers ---

func decodeBase64(s string) (string, error) {
	// Try the URL-safe no-padding variant first, then standard.
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

func splitColon(s string) (a, b string) {
	if i := strings.Index(s, ":"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func splitHostPort(s string) (host string, port int) {
	if i := strings.LastIndex(s, ":"); i >= 0 {
		host = s[:i]
		port, _ = strconv.Atoi(s[i+1:])
		return
	}
	return s, 0
}

// toInt converts a JSON value that may be a string or number to an int.
func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	case int:
		return t
	}
	return 0
}
