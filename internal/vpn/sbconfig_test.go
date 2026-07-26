//go:build windows && with_gvisor && with_quic && with_utls && with_clash_api

package vpn

import (
	"testing"

	sblog "github.com/sagernet/sing-box/log"

	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
)

// discardLogs satisfies the sing-box platform log writer without storing lines.
type discardLogs struct{}

func (discardLogs) WriteMessage(sblog.Level, string) {}

// A valid x25519 public key; REALITY decodes it while the outbound is created,
// so a placeholder would fail for the wrong reason.
const realityKey = "jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0"

const testUUID = "b831381d-6324-4d53-ad4f-8cda48b30811"

// TestGeneratedConfigIsAcceptedByCore feeds what singbox.Build produces into the
// real sing-box parser and constructor for every protocol the client can
// import. It is the guard against the config generator drifting away from the
// linked core version — exactly the breakage that silently ships otherwise,
// since a bad config only surfaces when a user hits Connect.
func TestGeneratedConfigIsAcceptedByCore(t *testing.T) {
	settings := model.AppSettings{
		Core:    model.CoreSingBox,
		DNS:     "1.1.1.1",
		TunName: "TomorrowTunTest",
		Stack:   "gvisor",
		Rules: []model.RoutingRule{
			{Type: "domain", Value: "ads.example", Action: "block"},
			{Type: "domain", Value: "corp.example", Action: "direct"},
			{Type: "ip", Value: "8.8.8.8", Action: "direct"},
			{Type: "ip", Value: "10.8.0.0/16", Action: "proxy"},
			{Type: "process", Value: "chrome.exe", Action: "proxy"},
		},
	}

	base := model.Profile{
		ID: "p", Name: "test", Address: "example.com", Port: 443,
	}

	cases := []struct {
		name   string
		mutate func(p *model.Profile)
	}{
		{"vless+reality", func(p *model.Profile) {
			p.Protocol = model.ProtoVLESS
			p.UUID = testUUID
			p.Flow = "xtls-rprx-vision"
			p.Security = "reality"
			p.PublicKey = realityKey
			p.ShortID = "0123456789abcdef"
			p.Fingerprint = "chrome"
		}},
		{"vless+ws+tls", func(p *model.Profile) {
			p.Protocol = model.ProtoVLESS
			p.UUID = testUUID
			p.Security = "tls"
			p.Network = "ws"
			p.Path = "/ws"
			p.Host = "example.com"
			p.ALPN = "h2,http/1.1"
		}},
		{"vmess+grpc", func(p *model.Profile) {
			p.Protocol = model.ProtoVMess
			p.UUID = testUUID
			p.Security = "tls"
			p.Network = "grpc"
			p.ServiceName = "gun"
		}},
		{"trojan", func(p *model.Profile) {
			p.Protocol = model.ProtoTrojan
			p.Password = "secret"
			p.Security = "tls"
		}},
		{"shadowsocks", func(p *model.Profile) {
			p.Protocol = model.ProtoShadowsocks
			p.Method = "aes-256-gcm"
			p.Password = "secret"
			p.Port = 8388
		}},
		{"hysteria2", func(p *model.Profile) {
			p.Protocol = model.ProtoHysteria2
			p.Password = "secret"
			p.Security = "tls"
			p.Obfs = "salamander"
			p.ObfsPassword = "obfs"
			p.UpMbps = 50
			p.DownMbps = 200
		}},
		{"tuic", func(p *model.Profile) {
			p.Protocol = model.ProtoTUIC
			p.UUID = testUUID
			p.Password = "secret"
			p.Security = "tls"
			p.Congestion = "bbr"
			p.UDPRelayMode = "native"
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)

			cfg, err := singbox.Build(p, settings)
			if err != nil {
				t.Fatalf("build config: %v", err)
			}

			// Create but never Start: constructing the core validates the whole
			// config without touching the TUN adapter, so no elevation needed.
			instance, cancel, err := newInstance(cfg, discardLogs{})
			if err != nil {
				t.Fatalf("core rejected generated config: %v\n\n%s", err, cfg)
			}
			_ = instance.Close()
			cancel()
		})
	}
}
