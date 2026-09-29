package singbox

import (
	"encoding/json"

	"TomorrowClient/internal/model"
)

// BuildProbe renders a throwaway config that exposes a single profile through a
// local mixed inbound: no TUN, no DNS, no routing rules, no logging. It is used
// to measure a server's real latency by making an actual request through it.
//
// A plain TCP handshake to the server cannot do that job. It only proves a port
// answered — a server with an expired key or a wrong password looks perfectly
// healthy — and on machines where a security product or transparent proxy
// accepts every connection, it reports every address as alive and instant.
//
// auto_detect_interface makes sing-box bind this probe's own connection to the
// physical adapter, so the measurement stays out of the app's own tunnel when
// one is running. Without it a probe would be carried through the active proxy
// and time the wrong path entirely.
func BuildProbe(p model.Profile, listenPort int, f Flavor) ([]byte, error) {
	proxy, err := outbound(p, f)
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{
		"log": map[string]any{"disabled": true},
		"inbounds": []any{
			map[string]any{
				"type":        "mixed",
				"tag":         "probe-in",
				"listen":      "127.0.0.1",
				"listen_port": listenPort,
			},
		},
		// The server name goes to the system resolver, as any app's would.
		"dns": map[string]any{
			"servers": []any{map[string]any{"type": "local", "tag": "system"}},
		},
		"route": map[string]any{
			"final":                   "proxy",
			"auto_detect_interface":   true,
			"default_domain_resolver": "system",
		},
	}
	if proxy["type"] == "wireguard" {
		cfg["endpoints"] = []any{proxy}
	} else {
		cfg["outbounds"] = []any{proxy}
	}
	return json.Marshal(cfg)
}
