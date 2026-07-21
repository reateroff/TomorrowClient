// Package sub fetches subscription URLs and turns them into profiles. A
// subscription body is the common "Base64 list" format: base64 of a newline
// separated list of share links (vless://, vmess://, trojan://, ss://). Some
// providers return the plain list without base64, so we handle both.
package sub

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"TomorrowClient/internal/link"
	"TomorrowClient/internal/model"
)

// userAgent mimics a common client so providers that gate on it still respond.
const userAgent = "TomorrowClient/1.0 (sing-box; xray)"

// fetchTimeout bounds the whole request.
const fetchTimeout = 20 * time.Second

// Userinfo holds the traffic/expiry metadata some providers report via the
// Subscription-Userinfo response header. Zero values mean "not reported".
type Userinfo struct {
	Upload   int64
	Download int64
	Total    int64
	Expire   int64
}

// Fetch downloads the subscription at url and parses it into profiles. Each
// returned profile has its SubID set to subID so it can be replaced on update.
// Individual links that fail to parse are skipped rather than failing the
// whole import. The provider's Subscription-Userinfo header (if any) is
// returned alongside as traffic/expiry metadata.
func Fetch(url, subID string) ([]model.Profile, Userinfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, Userinfo{}, fmt.Errorf("bad subscription url: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, Userinfo{}, fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, Userinfo{}, fmt.Errorf("subscription returned HTTP %d", resp.StatusCode)
	}

	info := parseUserinfo(resp.Header.Get("Subscription-Userinfo"))

	// Cap the body to a sane size (4 MiB) to avoid runaway responses.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, info, fmt.Errorf("read subscription: %w", err)
	}

	links := extractLinks(string(body))
	if len(links) == 0 {
		return nil, info, fmt.Errorf("no valid links in subscription")
	}

	profiles := make([]model.Profile, 0, len(links))
	for _, l := range links {
		p, err := link.Parse(l)
		if err != nil {
			continue // skip unsupported/broken entries
		}
		p.SubID = subID
		profiles = append(profiles, p)
	}
	if len(profiles) == 0 {
		return nil, info, fmt.Errorf("no supported servers in subscription")
	}
	return profiles, info, nil
}

// parseUserinfo reads the "upload=..; download=..; total=..; expire=.." header
// value into a Userinfo. Missing or malformed fields stay zero.
func parseUserinfo(h string) Userinfo {
	var info Userinfo
	if h == "" {
		return info
	}
	for _, part := range strings.Split(h, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "upload":
			info.Upload = n
		case "download":
			info.Download = n
		case "total":
			info.Total = n
		case "expire":
			info.Expire = n
		}
	}
	return info
}

// extractLinks turns a subscription body into a slice of share links. The body
// is either base64 of a link list, or the link list itself.
func extractLinks(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	// If it doesn't already look like a list of links, try base64 decoding it.
	if !looksLikeLinks(body) {
		if dec, ok := decodeBase64(body); ok {
			body = dec
		}
	}
	var out []string
	for _, line := range strings.FieldsFunc(body, func(r rune) bool {
		return r == '\n' || r == '\r'
	}) {
		line = strings.TrimSpace(line)
		if isSupportedLink(line) {
			out = append(out, line)
		}
	}
	return out
}

// looksLikeLinks reports whether the body already contains raw share links.
func looksLikeLinks(body string) bool {
	return strings.Contains(body, "://")
}

func isSupportedLink(s string) bool {
	return strings.HasPrefix(s, "vless://") ||
		strings.HasPrefix(s, "vmess://") ||
		strings.HasPrefix(s, "trojan://") ||
		strings.HasPrefix(s, "ss://") ||
		strings.HasPrefix(s, "hysteria2://") ||
		strings.HasPrefix(s, "hy2://") ||
		strings.HasPrefix(s, "hysteria://") ||
		strings.HasPrefix(s, "hy://") ||
		strings.HasPrefix(s, "tuic://")
}

// decodeBase64 tries the common base64 variants used by subscription providers.
func decodeBase64(s string) (string, bool) {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return string(b), true
		}
	}
	return "", false
}
