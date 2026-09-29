package singbox

import (
	"regexp"
	"sort"
	"strings"

	"TomorrowClient/internal/model"
)

// Rule-set sources for the geosite: and geoip: matchers — SagerNet's compiled
// sets, through the jsDelivr CDN, which stays reachable where GitHub's raw
// host is not.
const (
	geositeURL = "https://cdn.jsdelivr.net/gh/SagerNet/sing-geosite@rule-set/geosite-%s.srs"
	geoipURL   = "https://cdn.jsdelivr.net/gh/SagerNet/sing-geoip@rule-set/geoip-%s.srs"
)

// routeParts is the route block plus the DNS rules that go with it.
type routeParts struct {
	block    map[string]any
	dnsRules []any
}

// buildRoute renders the route block: the fixed plumbing (the fronted core's
// own traffic, sniffing, DNS hijack), then the user's rules — the flat list or
// the Pro graph — then LAN direct, then the catch-all.
func buildRoute(s model.AppSettings, front *SocksUpstream) routeParts {
	rules := []any{}
	if front != nil && front.Self != "" {
		// Ahead of the DNS hijack: the core's own lookups must not be answered
		// through the proxy they are trying to reach.
		rules = append(rules, map[string]any{"process_path": []any{front.Self}, "outbound": "direct"})
	}
	if s.Sniff {
		rules = append(rules, map[string]any{"action": "sniff"})
	}
	rules = append(rules, map[string]any{"protocol": "dns", "action": "hijack-dns"})

	final := model.ActionProxy
	var (
		ruleSets []any
		dnsRules []any
	)
	if s.RoutingMode == model.RoutingPro {
		g := graphRules(s.Graph)
		rules = append(rules, g.rules...)
		ruleSets, dnsRules = g.ruleSets, g.dnsRules
		if s.Graph.Final != "" {
			final = s.Graph.Final
		}
	} else {
		for _, r := range s.Rules {
			if rule := userRule(r); rule != nil {
				rules = append(rules, rule)
			}
		}
	}

	// Local/LAN traffic (loopback, private ranges) always goes direct.
	rules = append(rules, map[string]any{"ip_is_private": true, "outbound": "direct"})

	block := map[string]any{
		"auto_detect_interface":   true,
		"default_domain_resolver": "local",
		"final":                   "proxy",
	}
	switch final {
	case model.ActionDirect:
		block["final"] = "direct"
	case model.ActionBlock:
		// final must name an outbound; blocking is a rule that matches all.
		rules = append(rules, map[string]any{"network": []any{"tcp", "udp"}, "action": "reject"})
	}
	block["rules"] = rules
	if len(ruleSets) > 0 {
		block["rule_set"] = ruleSets
	}
	return routeParts{block: block, dnsRules: dnsRules}
}

// userRule converts a simple-mode RoutingRule into a sing-box route rule.
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
	return withAction(rule, r.Action)
}

// withAction sets a rule's action: reject for block, else an outbound.
func withAction(rule map[string]any, action string) map[string]any {
	switch action {
	case model.ActionBlock:
		rule["action"] = "reject"
	case model.ActionDirect:
		rule["outbound"] = "direct"
	default:
		rule["outbound"] = "proxy"
	}
	return rule
}

type graphParts struct {
	rules    []any
	ruleSets []any
	dnsRules []any
}

// graphRules renders the Pro graph's wired nodes, in their order.
func graphRules(g model.RouteGraph) graphParts {
	var out graphParts
	seen := map[string]bool{}
	for _, n := range g.Nodes {
		if n.Action == "" {
			continue // unwired: kept on the canvas, off the route
		}
		match, tags := nodeMatch(n)
		if match == nil {
			continue
		}
		for _, t := range tags {
			if !seen[t.tag] {
				seen[t.tag] = true
				out.ruleSets = append(out.ruleSets, t.set)
			}
		}
		rule := map[string]any{}
		for k, v := range match {
			rule[k] = v
		}
		out.rules = append(out.rules, withAction(rule, n.Action))

		// Split DNS: names that leave directly are resolved directly, and
		// blocked names are refused before any connection is tried.
		if dns := dnsMatch(match); dns != nil {
			switch n.Action {
			case model.ActionDirect:
				dns["server"] = "local"
				out.dnsRules = append(out.dnsRules, dns)
			case model.ActionBlock:
				dns["action"] = "reject"
				out.dnsRules = append(out.dnsRules, dns)
			}
		}
	}
	// Stable order keeps the generated config diffable.
	sort.SliceStable(out.ruleSets, func(i, j int) bool {
		return out.ruleSets[i].(map[string]any)["tag"].(string) < out.ruleSets[j].(map[string]any)["tag"].(string)
	})
	return out
}

type ruleSetRef struct {
	tag string
	set map[string]any
}

var (
	portRange = regexp.MustCompile(`^(\d{1,5})\s*[-:]\s*(\d{1,5})$`)
	portOne   = regexp.MustCompile(`^\d{1,5}$`)
	setName   = regexp.MustCompile(`^[a-z0-9!@._-]+$`)
)

// nodeMatch turns a node's values into sing-box rule match fields. Values that
// do not fit the node's kind are dropped rather than failing the whole config;
// nil means nothing valid is left.
func nodeMatch(n model.RouteNode) (map[string]any, []ruleSetRef) {
	var vals, ranges []any
	var sets []ruleSetRef
	field := ""
	for _, raw := range n.Values {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		switch n.Type {
		case "domain":
			field = "domain_suffix"
			vals = append(vals, strings.TrimPrefix(strings.TrimPrefix(v, "*."), "."))
		case "domain_full":
			field = "domain"
			vals = append(vals, v)
		case "domain_keyword":
			field = "domain_keyword"
			vals = append(vals, v)
		case "domain_regex":
			if _, err := regexp.Compile(v); err != nil {
				continue
			}
			field = "domain_regex"
			vals = append(vals, v)
		case "ip":
			field = "ip_cidr"
			vals = append(vals, normalizeCIDR(v))
		case "process":
			field = "process_name"
			vals = append(vals, v)
		case "process_path":
			field = "process_path"
			vals = append(vals, v)
		case "network":
			if v = strings.ToLower(v); v != "tcp" && v != "udp" {
				continue
			}
			field = "network"
			vals = append(vals, v)
		case "protocol":
			field = "protocol"
			vals = append(vals, strings.ToLower(v))
		case "port":
			if m := portRange.FindStringSubmatch(v); m != nil {
				ranges = append(ranges, m[1]+":"+m[2])
			} else if portOne.MatchString(v) {
				field = "port"
				vals = append(vals, atoi(v))
			}
		case "geosite", "geoip":
			name := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(v, "geosite:"), "geoip:"))
			if !setName.MatchString(name) {
				continue
			}
			tag := n.Type + "-" + name
			url := geositeURL
			if n.Type == "geoip" {
				url = geoipURL
			}
			sets = append(sets, ruleSetRef{tag: tag, set: map[string]any{
				"type":            "remote",
				"tag":             tag,
				"format":          "binary",
				"url":             strings.Replace(url, "%s", name, 1),
				"update_interval": "72h",
				// Fetched through the proxy: the CDN is what tends to be
				// filtered, the proxy is what is known to work.
				"http_client": map[string]any{"detour": "proxy"},
			}})
		}
	}

	match := map[string]any{}
	if len(vals) > 0 {
		match[field] = vals
	}
	if len(ranges) > 0 {
		if _, single := match["port"]; single {
			// port and port_range in one rule would both have to match; the
			// node means either, so the ranges go into a logical "or".
			return map[string]any{"type": "logical", "mode": "or", "rules": []any{
				map[string]any{"port": match["port"]},
				map[string]any{"port_range": ranges},
			}}, nil
		}
		match["port_range"] = ranges
	}
	if len(sets) > 0 {
		var names []any
		for _, t := range sets {
			names = append(names, t.tag)
		}
		match["rule_set"] = names
	}
	if len(match) == 0 {
		return nil, nil
	}
	return match, sets
}

// dnsMatch keeps the parts of a route match that also apply to a DNS query:
// the domain matchers and domain rule-sets.
func dnsMatch(match map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"domain", "domain_suffix", "domain_keyword", "domain_regex"} {
		if v, ok := match[k]; ok {
			out[k] = v
		}
	}
	if sets, ok := match["rule_set"].([]any); ok {
		var geosite []any
		for _, t := range sets {
			if strings.HasPrefix(t.(string), "geosite-") {
				geosite = append(geosite, t)
			}
		}
		if len(geosite) > 0 {
			out["rule_set"] = geosite
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
