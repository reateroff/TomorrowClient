//go:build windows && with_gvisor && with_quic && with_utls && with_clash_api && with_wireguard

package vpn

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	madapter "github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/hub/executor"
	sblog "github.com/sagernet/sing-box/log"
	tclog "github.com/tumgovic/todaycore/log"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/link"
	"TomorrowClient/internal/mihomo"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
	"TomorrowClient/internal/xray"
)

type discardLogs struct{}

func (discardLogs) WriteMessage(sblog.Level, string) {}

type discardTodayLogs struct{}

func (discardTodayLogs) WriteMessage(tclog.Level, string) {}

// Valid x25519 keys: REALITY and WireGuard decode them while the outbound is
// created, so a placeholder would fail for the wrong reason.
const (
	realityKey = "jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0"
	wgPrivate  = "eCtXsJZ27+4PbhDkHnB923tkUn2Gj59wZw5wFA75MnU="
	wgPublic   = "Cr8hWlKvtDt7nrvf2qBc4t4mU10BZX5oD/v7/3r9gGU="
	testUUID   = "b831381d-6324-4d53-ad4f-8cda48b30811"
	// A VLESS Encryption client string from Xray's own `xray vlessenc`.
	vlessEnc = "mlkem768x25519plus.native.0rtt.dL9i9FGxjZ3SHxWIMR5Ygz-2uVkLppKoSq13f0CmSzc"
)

var testSettings = model.AppSettings{
	Core:        model.CoreAuto,
	DNS:         "https://1.1.1.1/dns-query",
	DNSFallback: "8.8.8.8",
	TunName:     "TomorrowTunTest",
	Stack:       "gvisor",
	Rules: []model.RoutingRule{
		{Type: "domain", Value: "ads.example", Action: "block"},
		{Type: "domain", Value: "corp.example", Action: "direct"},
		{Type: "ip", Value: "8.8.8.8", Action: "direct"},
		{Type: "ip", Value: "10.8.0.0/16", Action: "proxy"},
		{Type: "process", Value: "chrome.exe", Action: "proxy"},
	},
}

// profileCases covers every protocol and transport the client can import.
// Each is run on every core that claims to support it.
var profileCases = []struct {
	name   string
	mutate func(p *model.Profile)
}{
	{"vless+reality+vision", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Flow = model.ProtoVLESS, testUUID, "xtls-rprx-vision"
		p.Security, p.PublicKey, p.ShortID, p.Fingerprint = "reality", realityKey, "0123456789abcdef", "chrome"
	}},
	{"vless+ws+tls+ed", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Security = model.ProtoVLESS, testUUID, "tls"
		p.Network, p.Path, p.Host, p.ALPN = "ws", "/ws?ed=2048", "example.com", "http/1.1"
	}},
	{"vless+httpupgrade", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Security = model.ProtoVLESS, testUUID, "tls"
		p.Network, p.Path, p.Host = "httpupgrade", "/up", "example.com"
	}},
	{"vless+xhttp+reality", func(p *model.Profile) {
		p.Protocol, p.UUID = model.ProtoVLESS, testUUID
		p.Security, p.PublicKey, p.ShortID, p.Fingerprint = "reality", realityKey, "ab", "chrome"
		p.Network, p.Path, p.Mode = "xhttp", "/x", "stream-up"
	}},
	{"vless+xhttp+extra", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Security = model.ProtoVLESS, testUUID, "tls"
		p.Network, p.Path, p.Host, p.Mode = "xhttp", "/x", "example.com", "packet-up"
		p.Extra = `{"xPaddingBytes":"100-1000","noGRPCHeader":true,"xmux":{"maxConcurrency":"16-32","hKeepAlivePeriod":30}}`
	}},
	{"vless+xhttp+downloadSettings", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Security = model.ProtoVLESS, testUUID, "tls"
		p.Network, p.Path, p.Mode = "xhttp", "/x", "auto"
		p.Extra = `{"downloadSettings":{"address":"dl.example.com","port":443,"network":"xhttp","security":"tls","xhttpSettings":{"path":"/x"}}}`
	}},
	{"vless+encryption", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Encryption = model.ProtoVLESS, testUUID, vlessEnc
	}},
	{"vless+grpc+reality", func(p *model.Profile) {
		p.Protocol, p.UUID = model.ProtoVLESS, testUUID
		p.Security, p.PublicKey, p.ShortID = "reality", realityKey, ""
		p.Network, p.ServiceName = "grpc", "gun"
	}},
	{"vmess+ws", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Network, p.Path = model.ProtoVMess, testUUID, "ws", "/v"
	}},
	{"vmess+alterid", func(p *model.Profile) {
		p.Protocol, p.UUID, p.AlterID = model.ProtoVMess, testUUID, 64
	}},
	{"vmess+grpc+tls", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Security, p.Network, p.ServiceName = model.ProtoVMess, testUUID, "tls", "grpc", "gun"
	}},
	{"vmess+h2", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Security, p.Network, p.Path, p.Host = model.ProtoVMess, testUUID, "tls", "h2", "/h", "example.com"
	}},
	{"vmess+tcp+http-header", func(p *model.Profile) {
		p.Protocol, p.UUID, p.HeaderType, p.Path, p.Host = model.ProtoVMess, testUUID, "http", "/", "example.com"
	}},
	{"vmess+kcp", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Network, p.HeaderType, p.Seed = model.ProtoVMess, testUUID, "kcp", "none", "seed"
	}},
	{"trojan", func(p *model.Profile) {
		p.Protocol, p.Password, p.Security = model.ProtoTrojan, "secret", "tls"
	}},
	{"trojan+insecure", func(p *model.Profile) {
		p.Protocol, p.Password, p.Security, p.AllowInsecure = model.ProtoTrojan, "secret", "tls", true
	}},
	{"shadowsocks", func(p *model.Profile) {
		p.Protocol, p.Method, p.Password, p.Port = model.ProtoShadowsocks, "aes-256-gcm", "secret", 8388
	}},
	{"shadowsocks-2022", func(p *model.Profile) {
		p.Protocol, p.Method, p.Password = model.ProtoShadowsocks, "2022-blake3-aes-128-gcm", "Y8PxFOE7smXq9UUTkPWfrg=="
	}},
	{"shadowsocks+obfs", func(p *model.Profile) {
		p.Protocol, p.Method, p.Password = model.ProtoShadowsocks, "chacha20-ietf-poly1305", "secret"
		p.Plugin, p.PluginOpts = "obfs-local", "obfs=http;obfs-host=example.com"
	}},
	{"hysteria2", func(p *model.Profile) {
		p.Protocol, p.Password, p.SNI = model.ProtoHysteria2, "secret", "example.com"
		p.Obfs, p.ObfsPassword, p.UpMbps, p.DownMbps = "salamander", "obfs", 50, 200
	}},
	{"hysteria2+hopping", func(p *model.Profile) {
		p.Protocol, p.Password, p.Ports = model.ProtoHysteria2, "secret", "20000-30000"
	}},
	{"hysteria", func(p *model.Profile) {
		p.Protocol, p.Password, p.ALPN = model.ProtoHysteria, "secret", "h3"
	}},
	{"tuic", func(p *model.Profile) {
		p.Protocol, p.UUID, p.Password = model.ProtoTUIC, testUUID, "secret"
		p.Congestion, p.UDPRelayMode, p.ALPN = "bbr", "native", "h3"
	}},
	{"wireguard", func(p *model.Profile) {
		p.Protocol, p.Port = model.ProtoWireGuard, 51820
		p.PrivateKey, p.PublicKey = wgPrivate, wgPublic
		p.LocalAddress, p.Reserved, p.MTU = "172.16.0.2/32,fd01::2/128", "1,2,3", 1280
	}},
	{"anytls", func(p *model.Profile) {
		p.Protocol, p.Password = model.ProtoAnyTLS, "secret"
	}},
	{"socks", func(p *model.Profile) {
		p.Protocol, p.Port, p.Username, p.Password = model.ProtoSOCKS, 1080, "u", "p"
	}},
	{"http", func(p *model.Profile) {
		p.Protocol, p.Port, p.Username, p.Password, p.Security = model.ProtoHTTP, 8443, "u", "p", "tls"
	}},
}

func profileFor(mutate func(*model.Profile)) model.Profile {
	p := model.Profile{ID: "p", Name: "test", Address: "example.com", Port: 443}
	mutate(&p)
	return p
}

// TestCoresAcceptGeneratedConfigs feeds every generated config into the real
// parser and constructor of the core that would run it — sing-box, TodayCore,
// Xray, mihomo, and the sing-box front in front of the last two. It is the
// guard against a generator drifting away from a linked core version: a bad
// config otherwise only surfaces when a user hits Connect.
//
// Nothing is started: constructing a core validates the whole config without
// touching the TUN adapter, so no elevation is needed.
func TestCoresAcceptGeneratedConfigs(t *testing.T) {
	for _, tc := range profileCases {
		p := profileFor(tc.mutate)
		supported := 0
		for _, c := range cores.All {
			if err := cores.Supports(c, p); err != nil {
				t.Logf("%s on %s: skipped (%v)", tc.name, c, err)
				continue
			}
			supported++
			t.Run(tc.name+"/"+string(c), func(t *testing.T) { checkCore(t, c, p) })
		}
		if supported == 0 {
			t.Errorf("%s: no core supports it", tc.name)
		}
		if c, err := cores.Resolve(model.CoreAuto, p); err != nil {
			t.Errorf("%s: auto found no core: %v", tc.name, err)
		} else {
			t.Logf("%s: auto → %s", tc.name, c)
		}
	}
}

func checkCore(t *testing.T, c model.Core, p model.Profile) {
	switch c {
	case model.CoreSingBox:
		cfg, err := singbox.Build(p, testSettings, singbox.Upstream)
		mustBuild(t, cfg, err)
		instance, cancel, err := newInstance(cfg, discardLogs{})
		if err != nil {
			t.Fatalf("sing-box rejected config: %v\n\n%s", err, cfg)
		}
		_ = instance.Close()
		cancel()

		probe, err := singbox.BuildProbe(p, 10999, singbox.Upstream)
		mustBuild(t, probe, err)
		instance, cancel, err = newInstance(probe, nil)
		if err != nil {
			t.Fatalf("sing-box rejected probe config: %v\n\n%s", err, probe)
		}
		_ = instance.Close()
		cancel()

	case model.CoreTodayCore:
		cfg, err := singbox.Build(p, testSettings, singbox.TodayCore)
		mustBuild(t, cfg, err)
		instance, cancel, err := newTodayInstance(cfg, discardTodayLogs{})
		if err != nil {
			t.Fatalf("TodayCore rejected config: %v\n\n%s", err, cfg)
		}
		_ = instance.Close()
		cancel()

		probe, err := singbox.BuildProbe(p, 10999, singbox.TodayCore)
		mustBuild(t, probe, err)
		instance, cancel, err = newTodayInstance(probe, nil)
		if err != nil {
			t.Fatalf("TodayCore rejected probe config: %v\n\n%s", err, probe)
		}
		_ = instance.Close()
		cancel()

	case model.CoreXray:
		cfg, err := xray.Build(p, xray.Local{Port: 10999, Username: "u", Password: "p", Interface: "Ethernet"})
		mustBuild(t, cfg, err)
		instance, err := newXrayInstance(cfg)
		if err != nil {
			t.Fatalf("Xray rejected config: %v\n\n%s", err, cfg)
		}
		_ = instance.Close()
		checkFront(t, p)

	case model.CoreMihomo:
		cfg, err := mihomo.Build(p, mihomo.Local{Port: 10999, Username: "u", Password: "p", Interface: "Ethernet"})
		mustBuild(t, cfg, err)
		parsed, err := executor.ParseWithBytes(cfg)
		if err != nil {
			t.Fatalf("mihomo rejected config: %v\n\n%s", err, cfg)
		}
		if _, ok := parsed.Proxies[mihomo.ProxyName]; !ok {
			t.Fatalf("mihomo dropped the proxy\n\n%s", cfg)
		}
		m, _ := mihomo.Proxy(p)
		if _, err := madapter.ParseProxy(m); err != nil {
			t.Fatalf("mihomo rejected proxy for probing: %v", err)
		}
		checkFront(t, p)
	}
}

// checkFront validates the sing-box front a fronted core runs behind.
func checkFront(t *testing.T, p model.Profile) {
	cfg, err := singbox.BuildFront(testSettings, singbox.SocksUpstream{
		Port: 10999, Username: "u", Password: "p", Server: p.Address, Self: `C:\TomorrowClient.exe`,
	})
	mustBuild(t, cfg, err)
	instance, cancel, err := newInstance(cfg, discardLogs{})
	if err != nil {
		t.Fatalf("sing-box rejected front config: %v\n\n%s", err, cfg)
	}
	_ = instance.Close()
	cancel()
}

func mustBuild(t *testing.T, cfg []byte, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("build config: %v", err)
	}
}

// TestProbeRejectsDeadServer: a TCP handshake cannot tell a dead server apart
// on a machine where something accepts every connection, so every core's probe
// has to fail by not completing a request through it.
func TestProbeRejectsDeadServer(t *testing.T) {
	dead := model.Profile{
		ID: "dead", Name: "dead",
		Protocol: model.ProtoTrojan,
		Address:  "192.0.2.1", // TEST-NET-1
		Port:     443,
		Password: "nope",
		Security: "tls",
	}
	for _, c := range cores.All {
		t.Run(string(c), func(t *testing.T) {
			start := time.Now()
			d, err := ProbeLatency(dead, c, ProbeOptions{})
			elapsed := time.Since(start)
			if err == nil {
				t.Fatalf("probe reported a dead server as reachable in %v", d)
			}
			t.Logf("failed after %v: %v", elapsed.Round(time.Millisecond), err)
			if elapsed > probeTimeout+2*time.Second {
				t.Errorf("probe took %v, well past the %v timeout", elapsed, probeTimeout)
			}
		})
	}
}

// TestProbeThroughLocalServer pushes a real request through every core: a
// Shadowsocks server and an HTTP target both run on loopback, and each core's
// probe has to carry the request from one to the other. The dead-server test
// proves failures fail; this one proves the data path works at all.
func TestProbeThroughLocalServer(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	saved := probeURL
	probeURL = target.URL + "/generate_204"
	defer func() { probeURL = saved }()

	port, err := freeLoopbackPort()
	if err != nil {
		t.Fatal(err)
	}
	const method, password = "aes-128-gcm", "local-test-secret"
	server := fmt.Sprintf(`{
		"log": {"disabled": true},
		"inbounds": [{"type": "shadowsocks", "listen": "127.0.0.1", "listen_port": %d,
			"method": %q, "password": %q}],
		"outbounds": [{"type": "direct"}]
	}`, port, method, password)
	instance, cancel, err := newInstance([]byte(server), nil)
	if err != nil {
		t.Fatalf("test server: %v", err)
	}
	if err := instance.Start(); err != nil {
		t.Fatalf("start test server: %v", err)
	}
	defer func() { _ = instance.Close(); cancel() }()

	p := model.Profile{
		ID: "local", Name: "local", Protocol: model.ProtoShadowsocks,
		Address: "127.0.0.1", Port: port, Method: method, Password: password,
	}
	for _, c := range cores.All {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(string(c)+"/"+method, func(t *testing.T) {
				d, err := ProbeLatency(p, c, ProbeOptions{Method: method, Timeout: 5 * time.Second})
				if err != nil {
					t.Fatalf("%s through %s failed: %v", method, c, err)
				}
				t.Logf("%s carried the %s in %v", c, method, d.Round(time.Microsecond))
			})
		}
	}

	// TCP ping reaches the server's port itself.
	if d, err := PingTCP("127.0.0.1", port, 2*time.Second); err != nil {
		t.Errorf("TCP ping to the local server failed: %v", err)
	} else {
		t.Logf("TCP handshake in %v", d.Round(time.Microsecond))
	}
	if _, err := PingTCP("127.0.0.1", 1, time.Second); err == nil {
		t.Error("TCP ping reported a closed port as open")
	}
}

// TestParsedLinksRunOnCores takes share links as providers publish them through
// the parser and every core that accepts the result, so parser and generators
// are checked as the chain they form in the app.
func TestParsedLinksRunOnCores(t *testing.T) {
	links := []string{
		"vless://" + testUUID + "@example.com:443?type=xhttp&security=reality&pbk=" + realityKey +
			"&sid=ab12&fp=chrome&sni=www.example.com&path=%2Fapi&mode=auto" +
			"&extra=%7B%22xPaddingBytes%22%3A%22100-1000%22%2C%22xmux%22%3A%7B%22maxConcurrency%22%3A%2216-32%22%7D%7D#x",
		"vless://" + testUUID + "@example.com:443?type=tcp&security=reality&pbk=" + realityKey +
			"&sid=ab12&fp=chrome&sni=www.example.com&flow=xtls-rprx-vision#vision",
		"vless://" + testUUID + "@example.com:443?type=ws&security=tls&path=%2Fws%3Fed%3D2048&host=cdn.example.com#ws",
		"vless://" + testUUID + "@example.com:443?encryption=" + vlessEnc + "#enc",
		"trojan://secret@example.com:443?type=grpc&serviceName=gun&sni=example.com#grpc",
		"ss://" + "Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpzZWNyZXQ" + "@example.com:8388/?plugin=obfs-local%3Bobfs%3Dhttp%3Bobfs-host%3Dbing.com#obfs",
		"hysteria2://secret@example.com:443,20000-30000/?sni=example.com&obfs=salamander&obfs-password=o#hop",
		"tuic://" + testUUID + ":secret@example.com:443?congestion_control=bbr&alpn=h3#tuic",
		"wireguard://" + url.QueryEscape(wgPrivate) + "@example.com:51820?publickey=" + url.QueryEscape(wgPublic) +
			"&address=172.16.0.2%2F32&reserved=1,2,3#wg",
		"anytls://secret@example.com:443?sni=example.com#anytls",
	}
	for _, raw := range links {
		p, err := link.Parse(raw)
		if err != nil {
			t.Errorf("parse %s: %v", raw, err)
			continue
		}
		ran := 0
		for _, c := range cores.All {
			if cores.Supports(c, p) != nil {
				continue
			}
			ran++
			t.Run(p.Name+"/"+string(c), func(t *testing.T) { checkCore(t, c, p) })
		}
		if ran == 0 {
			t.Errorf("%s: parsed profile runs on no core", p.Name)
		}
	}
}

// TestProRoutingAccepted runs the Pro graph — every node kind, split DNS,
// rule-sets, a blocking catch-all, IPv6 — through each tunnel-owning core.
func TestProRoutingAccepted(t *testing.T) {
	s := testSettings
	s.RoutingMode = model.RoutingPro
	s.IPv6 = true
	s.StrictRoute = false
	s.Sniff = true
	s.Graph = model.RouteGraph{
		Final: model.ActionBlock,
		Nodes: []model.RouteNode{
			{ID: "1", Type: "domain", Values: []string{"example.com", "*.corp.example"}, Action: model.ActionDirect},
			{ID: "2", Type: "domain_full", Values: []string{"exact.example.com"}, Action: model.ActionProxy},
			{ID: "3", Type: "domain_keyword", Values: []string{"ads"}, Action: model.ActionBlock},
			{ID: "4", Type: "domain_regex", Values: []string{`^cdn\d+\.example\.com$`, "(broken"}, Action: model.ActionDirect},
			{ID: "5", Type: "ip", Values: []string{"2.56.24.0/22", "1.1.1.1", "2001:db8::/32"}, Action: model.ActionDirect},
			{ID: "6", Type: "port", Values: []string{"443", "27000-27100"}, Action: model.ActionProxy},
			{ID: "7", Type: "port", Values: []string{"6881:6889"}, Action: model.ActionDirect},
			{ID: "8", Type: "process", Values: []string{"javaw.exe", "cs2.exe"}, Action: model.ActionProxy},
			{ID: "9", Type: "process_path", Values: []string{`C:\Games\steam.exe`}, Action: model.ActionDirect},
			{ID: "10", Type: "network", Values: []string{"udp", "icmp"}, Action: model.ActionProxy},
			{ID: "11", Type: "protocol", Values: []string{"bittorrent"}, Action: model.ActionBlock},
			{ID: "12", Type: "geosite", Values: []string{"category-ads-all"}, Action: model.ActionBlock},
			{ID: "13", Type: "geoip", Values: []string{"ru"}, Action: model.ActionDirect},
			{ID: "14", Type: "geosite", Values: []string{"ru", "geosite:category-ru"}, Action: model.ActionDirect},
			{ID: "15", Type: "domain", Values: []string{"unwired.example"}, Action: ""},
			{ID: "16", Type: "ip", Values: []string{}, Action: model.ActionBlock},
		},
	}
	p := profileFor(profileCases[0].mutate) // vless+reality

	for _, f := range []singbox.Flavor{singbox.Upstream, singbox.TodayCore} {
		cfg, err := singbox.Build(p, s, f)
		mustBuild(t, cfg, err)
		if f == singbox.Upstream {
			instance, cancel, err := newInstance(cfg, discardLogs{})
			if err != nil {
				t.Fatalf("sing-box rejected Pro routing: %v\n\n%s", err, cfg)
			}
			_ = instance.Close()
			cancel()
		} else {
			instance, cancel, err := newTodayInstance(cfg, discardTodayLogs{})
			if err != nil {
				t.Fatalf("TodayCore rejected Pro routing: %v\n\n%s", err, cfg)
			}
			_ = instance.Close()
			cancel()
		}
	}

	front, err := singbox.BuildFront(s, singbox.SocksUpstream{Port: 10999, Username: "u", Password: "p", Server: p.Address, Self: `C:\x.exe`})
	mustBuild(t, front, err)
	instance, cancel, err := newInstance(front, discardLogs{})
	if err != nil {
		t.Fatalf("sing-box rejected Pro routing in the front: %v\n\n%s", err, front)
	}
	_ = instance.Close()
	cancel()
}
