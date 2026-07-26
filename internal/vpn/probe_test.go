//go:build windows && with_gvisor && with_quic && with_utls && with_clash_api

package vpn

import (
	"testing"
	"time"

	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
)

// TestProbeConfigIsAcceptedByCore checks that the throwaway probe config parses
// and builds for every protocol, the same guard sbconfig_test gives the real
// config.
func TestProbeConfigIsAcceptedByCore(t *testing.T) {
	base := model.Profile{ID: "p", Name: "t", Address: "example.com", Port: 443}

	cases := []struct {
		name   string
		mutate func(p *model.Profile)
	}{
		{"vless+reality", func(p *model.Profile) {
			p.Protocol = model.ProtoVLESS
			p.UUID = testUUID
			p.Security = "reality"
			p.PublicKey = realityKey
			p.ShortID = "0123456789abcdef"
			p.Fingerprint = "chrome"
		}},
		{"trojan", func(p *model.Profile) {
			p.Protocol = model.ProtoTrojan
			p.Password = "secret"
			p.Security = "tls"
		}},
		{"hysteria2", func(p *model.Profile) {
			p.Protocol = model.ProtoHysteria2
			p.Password = "secret"
			p.Security = "tls"
		}},
		{"shadowsocks", func(p *model.Profile) {
			p.Protocol = model.ProtoShadowsocks
			p.Method = "aes-256-gcm"
			p.Password = "secret"
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)

			cfg, err := singbox.BuildProbe(p, 34567)
			if err != nil {
				t.Fatalf("BuildProbe: %v", err)
			}
			instance, cancel, err := newInstance(cfg, nil)
			if err != nil {
				t.Fatalf("core rejected probe config: %v\n\n%s", err, cfg)
			}
			_ = instance.Close()
			cancel()
		})
	}
}

// TestProbeRejectsDeadServer is the point of the whole rework. A TCP handshake
// cannot tell a dead server apart on a machine where something accepts every
// connection, so the probe has to fail by not completing a request through it.
func TestProbeRejectsDeadServer(t *testing.T) {
	dead := model.Profile{
		ID: "dead", Name: "dead",
		Protocol: model.ProtoTrojan,
		Address:  "192.0.2.1", // TEST-NET-1
		Port:     443,
		Password: "nope",
		Security: "tls",
	}

	start := time.Now()
	d, err := ProbeLatency(dead)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("probe reported a dead server as reachable in %v", d)
	}
	t.Logf("dead server correctly failed after %v: %v", elapsed.Round(time.Millisecond), err)

	if elapsed > probeTimeout+2*time.Second {
		t.Errorf("probe took %v, well past the %v timeout", elapsed, probeTimeout)
	}
}
