package sub

import (
	"encoding/base64"
	"testing"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/model"
)

const clashDoc = `
mixed-port: 7890
proxies:
  - name: "🇩🇪 DE vless"
    type: vless
    server: de.example.com
    port: 443
    uuid: b831381d-6324-4d53-ad4f-8cda48b30811
    network: xhttp
    tls: true
    servername: www.example.com
    client-fingerprint: chrome
    reality-opts: {public-key: KEY, short-id: ab}
    xhttp-opts: {path: /x, mode: stream-up}
  - {name: ws, type: vmess, server: v.example.com, port: 443, uuid: id, alterId: 0, cipher: auto, tls: true, network: ws, ws-opts: {path: /ws, headers: {Host: cdn.example.com}, v2ray-http-upgrade: true}}
  - {name: ss, type: ss, server: s.example.com, port: 8388, cipher: aes-128-gcm, password: pw, plugin: obfs, plugin-opts: {mode: tls, host: bing.com}}
  - {name: hy2, type: hysteria2, server: h.example.com, port: 443, password: pw, ports: 20000-30000, up: "50 Mbps", obfs: salamander, obfs-password: o}
  - {name: wg, type: wireguard, server: w.example.com, port: 51820, private-key: PRIV, public-key: PUB, ip: 172.16.0.2, ipv6: fd01::2, reserved: [1, 2, 3], mtu: 1280}
  - {name: ssr, type: ssr, server: r.example.com, port: 1, cipher: none, password: x, protocol: origin, obfs: plain}
  - {name: group, type: select, proxies: [ws]}
`

func TestParseClash(t *testing.T) {
	ps, err := Parse(clashDoc)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 5 {
		t.Fatalf("got %d profiles, want 5 (ssr and the group are skipped)", len(ps))
	}
	vless, ws, ss, hy2, wg := ps[0], ps[1], ps[2], ps[3], ps[4]

	if vless.Security != "reality" || vless.Network != "xhttp" || vless.Mode != "stream-up" || vless.PublicKey != "KEY" {
		t.Errorf("vless: %+v", vless)
	}
	if vless.Name != "🇩🇪 DE vless" || vless.ID == "" {
		t.Errorf("vless name/id: %q %q", vless.Name, vless.ID)
	}
	if ws.Network != "httpupgrade" || ws.Host != "cdn.example.com" || ws.Security != "tls" {
		t.Errorf("vmess: %+v", ws)
	}
	if ss.Plugin != "obfs-local" || ss.PluginOpts != "obfs=tls;obfs-host=bing.com" {
		t.Errorf("ss plugin: %q %q", ss.Plugin, ss.PluginOpts)
	}
	if hy2.Ports != "20000-30000" || hy2.UpMbps != 50 || hy2.Obfs != "salamander" {
		t.Errorf("hy2: %+v", hy2)
	}
	if wg.LocalAddress != "172.16.0.2/32,fd01::2/128" || wg.Reserved != "1,2,3" || wg.MTU != 1280 {
		t.Errorf("wireguard: %+v", wg)
	}
	for _, p := range ps {
		if _, err := cores.Resolve(model.CoreAuto, p); err != nil {
			t.Errorf("%s runs on no core: %v", p.Name, err)
		}
	}
}

const singBoxDoc = `{
  "outbounds": [
    {"type": "selector", "tag": "select", "outbounds": ["a"]},
    {"type": "vless", "tag": "a", "server": "a.example.com", "server_port": 443,
     "uuid": "id", "flow": "xtls-rprx-vision",
     "tls": {"enabled": true, "server_name": "www.example.com",
             "utls": {"enabled": true, "fingerprint": "chrome"},
             "reality": {"enabled": true, "public_key": "KEY", "short_id": "ab"}}},
    {"type": "trojan", "tag": "t", "server": "t.example.com", "server_port": 443, "password": "pw",
     "tls": {"enabled": true, "insecure": true},
     "transport": {"type": "grpc", "service_name": "gun"}},
    {"type": "hysteria2", "tag": "h", "server": "h.example.com", "server_port": 443,
     "server_ports": ["20000:30000"], "password": "pw", "tls": {"enabled": true}},
    {"type": "direct", "tag": "direct"}
  ],
  "endpoints": [
    {"type": "wireguard", "tag": "wg", "address": ["172.16.0.2/32"], "private_key": "PRIV",
     "peers": [{"address": "w.example.com", "port": 51820, "public_key": "PUB", "reserved": [1, 2, 3]}]}
  ]
}`

func TestParseSingBox(t *testing.T) {
	ps, err := Parse(singBoxDoc)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 4 {
		t.Fatalf("got %d profiles, want 4", len(ps))
	}
	if ps[0].Security != "reality" || ps[0].Fingerprint != "chrome" || ps[0].Name != "a" {
		t.Errorf("vless: %+v", ps[0])
	}
	if ps[1].Network != "grpc" || ps[1].ServiceName != "gun" || !ps[1].AllowInsecure {
		t.Errorf("trojan: %+v", ps[1])
	}
	if ps[2].Ports != "20000-30000" {
		t.Errorf("hy2 ports: %q", ps[2].Ports)
	}
	if ps[3].Address != "w.example.com" || ps[3].Reserved != "1,2,3" {
		t.Errorf("wireguard: %+v", ps[3])
	}
}

func TestParseLinksStillFirst(t *testing.T) {
	list := "vless://id@a.example.com:443?security=tls#a\ntrojan://pw@b.example.com:443#b\nnot a link"
	ps, err := Parse(base64.StdEncoding.EncodeToString([]byte(list)))
	if err != nil || len(ps) != 2 {
		t.Fatalf("base64 list: %d profiles, err %v", len(ps), err)
	}
	if _, err := Parse("hello world"); err == nil {
		t.Error("garbage parsed")
	}
}

// A JSON-imported server keeps its entry, and an edited entry parses back.
func TestRawJSONAndParseEntry(t *testing.T) {
	ps, err := Parse(singBoxDoc)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseEntry(ps[1].Raw) // trojan+grpc, as imported
	if err != nil {
		t.Fatalf("sing-box entry: %v\n%s", err, ps[1].Raw)
	}
	if p.Protocol != model.ProtoTrojan || p.ServiceName != "gun" {
		t.Errorf("sing-box entry parsed as %+v", p)
	}
	if p, err := ParseEntry(ps[3].Raw); err != nil || p.Protocol != model.ProtoWireGuard {
		t.Errorf("sing-box wireguard endpoint: %+v %v", p, err)
	}

	cs, err := Parse(clashDoc)
	if err != nil {
		t.Fatal(err)
	}
	p, err = ParseEntry(cs[3].Raw) // hysteria2 from Clash
	if err != nil || p.Protocol != model.ProtoHysteria2 || p.Ports != "20000-30000" {
		t.Errorf("clash entry: %+v %v\n%s", p, err, cs[3].Raw)
	}
	if _, err := ParseEntry(`{"type":"select"}`); err == nil {
		t.Error("a group parsed as a server")
	}
}
