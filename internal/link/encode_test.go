package link

import (
	"reflect"
	"testing"

	"TomorrowClient/internal/model"
)

// TestEncodeRoundTrip checks that an edited profile survives being written back
// to a link: Parse(Encode(p)) must give p again, field for field. Profiles are
// written in the normalized form Parse produces (tcp, none, udp+tls for QUIC).
func TestEncodeRoundTrip(t *testing.T) {
	cases := []model.Profile{
		{Name: "🇩🇪 Германия", Protocol: model.ProtoVLESS, Address: "de.example.com", Port: 443,
			UUID: "3bcb526a-0e21-4a5c-91ba-e6fbca5d4956", Flow: "xtls-rprx-vision", Network: "tcp",
			Security: "reality", SNI: "dl.google.com", Fingerprint: "qq",
			PublicKey: "Uvw3P07ogOAqRuCaskfnLjvzbYNmGMwvtzLbRB_WkzM", ShortID: "e385f8060d1dd675", SpiderX: "/"},
		{Name: "xhttp", Protocol: model.ProtoVLESS, Address: "1.2.3.4", Port: 8443, UUID: "id",
			Network: "xhttp", Security: "tls", SNI: "a.example.com", ALPN: "h2,http/1.1", AllowInsecure: true,
			Path: "/x", Host: "cdn.example.com", Mode: "stream-up", Extra: `{"xPaddingBytes":"100-1000"}`,
			Encryption: "mlkem768x25519plus.native.0rtt.KEY"},
		{Name: "grpc", Protocol: model.ProtoTrojan, Address: "t.example.com", Port: 443, Password: "p@ss:word",
			Network: "grpc", Security: "tls", ServiceName: "gun", Mode: "multi"},
		{Name: "kcp", Protocol: model.ProtoVLESS, Address: "k.example.com", Port: 443, UUID: "id",
			Network: "kcp", Security: "none", HeaderType: "wechat-video", Seed: "s"},
		{Name: "vmess ws", Protocol: model.ProtoVMess, Address: "v.example.com", Port: 443, UUID: "id",
			AlterID: 0, Method: "auto", Network: "ws", Security: "tls", SNI: "s.example.com",
			Host: "h.example.com", Path: "/ws?ed=2048", Fingerprint: "chrome", ALPN: "h2"},
		{Name: "ss obfs", Protocol: model.ProtoShadowsocks, Address: "s.example.com", Port: 8388,
			Method: "chacha20-ietf-poly1305", Password: "secret", Network: "tcp", Security: "none",
			Plugin: "obfs-local", PluginOpts: "obfs=http;obfs-host=bing.com"},
		{Name: "ss2022", Protocol: model.ProtoShadowsocks, Address: "2001:db8::1", Port: 443,
			Method: "2022-blake3-aes-128-gcm", Password: "Y8PxFOE7smXq9UUTkPWfrg==", Network: "tcp", Security: "none"},
		{Name: "hy2", Protocol: model.ProtoHysteria2, Address: "h.example.com", Port: 443, Password: "pw",
			Network: "udp", Security: "tls", SNI: "s.example.com", Obfs: "salamander", ObfsPassword: "o",
			Ports: "20000-30000", UpMbps: 50, DownMbps: 200, AllowInsecure: true},
		{Name: "hy", Protocol: model.ProtoHysteria, Address: "h.example.com", Port: 443, Password: "a",
			Network: "udp", Security: "tls", SNI: "s.example.com", Obfs: "o", UpMbps: 10, DownMbps: 50},
		{Name: "tuic", Protocol: model.ProtoTUIC, Address: "t.example.com", Port: 443, UUID: "uuid", Password: "pw",
			Network: "udp", Security: "tls", SNI: "s", ALPN: "h3", Congestion: "bbr", UDPRelayMode: "native"},
		{Name: "wg", Protocol: model.ProtoWireGuard, Address: "w.example.com", Port: 51820, Network: "udp",
			PrivateKey: "cHJpdmF0ZQ==", PublicKey: "cHVibGlj=", PreSharedKey: "cHNr",
			LocalAddress: "10.0.0.2/32,fd00::2/128", Reserved: "1,2,3", MTU: 1280},
		{Name: "anytls", Protocol: model.ProtoAnyTLS, Address: "a.example.com", Port: 443, Password: "pw",
			Security: "tls", SNI: "s.example.com", Fingerprint: "chrome"},
		{Name: "socks", Protocol: model.ProtoSOCKS, Address: "127.0.0.1", Port: 1080, Username: "u", Password: "p",
			Security: "none"},
	}
	for _, want := range cases {
		t.Run(want.Name, func(t *testing.T) {
			raw := Encode(want)
			got, err := Parse(raw)
			if err != nil {
				t.Fatalf("Parse(Encode(p)) failed: %v\n%s", err, raw)
			}
			got.ID, got.Raw = "", ""
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round trip changed the profile\nlink: %s\n got: %+v\nwant: %+v", raw, got, want)
			}
		})
	}
}
