// Package sub fetches subscription URLs and turns them into profiles. The body
// is usually the "Base64 list" format — base64 of a newline separated list of
// share links (see link.Schemes), sometimes the plain list — but panels choose
// the format by User-Agent and may send a sing-box or Clash config instead;
// Parse reads all of them.
package sub

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"TomorrowClient/internal/link"
	"TomorrowClient/internal/model"
)

// DefaultUserAgent is sent unless the user sets another: "TomorrowClient/"
// plus the app version, set by the app at start. Panels choose the answer
// format by it; every format they send is understood (see Parse).
var DefaultUserAgent = "TomorrowClient"

// fetchTimeout bounds the whole request.
const fetchTimeout = 20 * time.Second

// Meta is what a provider tells us about a subscription besides the servers
// themselves. Zero values mean "not reported".
type Meta struct {
	// Traffic and expiry, from the Subscription-Userinfo header.
	Upload   int64
	Download int64
	Total    int64
	Expire   int64
	// Title is the subscription's own display name, from profile-title or the
	// Content-Disposition filename. Without it the UI has nothing to show but
	// the URL host.
	Title string
}

// maxTitleLen keeps a provider from pushing an essay into the sidebar.
const maxTitleLen = 64

// Fetch downloads the subscription at url and parses it into profiles. Each
// returned profile has its SubID set to subID so it can be replaced on update.
// Individual entries that fail to parse are skipped rather than failing the
// whole import. The provider's Subscription-Userinfo header (if any) is
// returned alongside as traffic/expiry metadata.
//
// headers go out with the request: the User-Agent and, when enabled, the
// device (HWID) headers.
func Fetch(url, subID string, headers map[string]string) ([]model.Profile, Meta, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, Meta{}, fmt.Errorf("bad subscription url: %w", err)
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, Meta{}, fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, Meta{}, fmt.Errorf("subscription returned HTTP %d", resp.StatusCode)
	}

	info := parseUserinfo(resp.Header.Get("Subscription-Userinfo"))
	info.Title = parseTitle(resp.Header)

	// Cap the body to a sane size (4 MiB) to avoid runaway responses.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, info, fmt.Errorf("read subscription: %w", err)
	}

	profiles, err := Parse(string(body))
	if err != nil {
		return nil, info, fmt.Errorf("no supported servers in subscription: %w", err)
	}
	for i := range profiles {
		profiles[i].SubID = subID
	}
	return profiles, info, nil
}

// parseUserinfo reads the "upload=..; download=..; total=..; expire=.." header
// value into a Userinfo. Missing or malformed fields stay zero.
func parseUserinfo(h string) Meta {
	var info Meta
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

// parseTitle reads the subscription's display name from the headers providers
// actually use for it: "profile-title" (plain, or wrapped as "base64:...") and,
// failing that, the filename in Content-Disposition. Without this the client
// has nothing to name a subscription but the URL host.
func parseTitle(h http.Header) string {
	if t := strings.TrimSpace(h.Get("profile-title")); t != "" {
		if enc, ok := strings.CutPrefix(t, "base64:"); ok {
			b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(enc))
			if err != nil {
				return ""
			}
			return clampTitle(string(b))
		}
		return clampTitle(t)
	}

	if cd := h.Get("Content-Disposition"); cd != "" {
		// ParseMediaType also decodes the RFC 5987 filename*=UTF-8''… form.
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			return clampTitle(params["filename"])
		}
	}
	return ""
}

// clampTitle trims a provider-supplied name down to something displayable.
func clampTitle(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxTitleLen {
		s = strings.TrimSpace(s[:maxTitleLen])
	}
	return s
}

// ExtractLinks turns a subscription body into a slice of share links. The body
// is either base64 of a link list, or the link list itself.
func ExtractLinks(body string) []string {
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

func isSupportedLink(s string) bool { return link.IsLink(s) }

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
