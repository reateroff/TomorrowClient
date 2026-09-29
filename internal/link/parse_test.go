package link

import (
	"encoding/base64"
	"testing"

	"TomorrowClient/internal/model"
)

func mustParse(t *testing.T, raw string) model.Profile {
	t.Helper()
	p, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%q): %v", raw, err)
	}
	if p.ID == "" || p.Raw == "" {
		t.Errorf("profile missing ID or Raw: %+v", p)
	}
	return p
}

func eq[T comparable](t *testing.T, field string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

func TestVLESSRealityXHTTP(t *testing.T) {
	p := mustParse(t, "vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443"+
		"?type=xhttp&security=reality&pbk=KEY&sid=ab12&spx=%2F&fp=chrome&sni=www.example.com"+
		"&path=%2Fapi&host=cdn.example.com&mode=stream-up"+
		"&extra=%7B%22xPaddingBytes%22%3A%22100-1000%22%7D&encryption=none#%F0%9F%87%B3%F0%9F%87%B1%20NL")
	eq(t, "protocol", p.Protocol, model.ProtoVLESS)
	eq(t, "network", p.Network, "xhttp")
	eq(t, "security", p.Security, "reality")
	eq(t, "publicKey", p.PublicKey, "KEY")
	eq(t, "shortId", p.ShortID, "ab12")
	eq(t, "spiderX", p.SpiderX, "/")
	eq(t, "mode", p.Mode, "stream-up")
	eq(t, "extra", p.Extra, `{"xPaddingBytes":"100-1000"}`)
	eq(t, "encryption", p.Encryption, "") // "none" is plain VLESS
	eq(t, "name", p.Name, "🇳🇱 NL")
}

func TestVLESSEncryptionAndAliases(t *testing.T) {
	p := mustParse(t, "vless://id@example.com:8443?type=splithttp&security=tls&allowInsecure=1"+
		"&encryption=mlkem768x25519plus.native.0rtt.KEY&flow=xtls-rprx-vision")
	eq(t, "network", p.Network, "xhttp")
	eq(t, "encryption", p.Encryption, "mlkem768x25519plus.native.0rtt.KEY")
	eq(t, "allowInsecure", p.AllowInsecure, true)
	eq(t, "flow", p.Flow, "xtls-rprx-vision")
	eq(t, "name", p.Name, "example.com")
}

func TestVLESSGRPCAndKCP(t *testing.T) {
	p := mustParse(t, "vless://id@h:443?type=grpc&serviceName=gun&mode=multi&security=tls")
	eq(t, "serviceName", p.ServiceName, "gun")
	eq(t, "mode", p.Mode, "multi")

	p = mustParse(t, "vless://id@h:443?type=kcp&headerType=wechat-video&seed=s3cr3t")
	eq(t, "network", p.Network, "kcp")
	eq(t, "headerType", p.HeaderType, "wechat-video")
	eq(t, "seed", p.Seed, "s3cr3t")

	p = mustParse(t, "vless://id@h:443?type=raw&headerType=http&host=a.com&path=%2F")
	eq(t, "network", p.Network, "tcp")
	eq(t, "headerType", p.HeaderType, "http")
}

func TestVMessV2rayN(t *testing.T) {
	js := `{"v":"2","ps":"vm","add":"v.example.com","port":"443","id":"id","aid":"0","scy":"aes-128-gcm",` +
		`"net":"ws","type":"none","host":"cdn.example.com","path":"/ws?ed=2048","tls":"tls","sni":"sni.example.com",` +
		`"alpn":"h2,http/1.1","fp":"firefox","allowInsecure":1}`
	p := mustParse(t, "vmess://"+base64.StdEncoding.EncodeToString([]byte(js)))
	eq(t, "address", p.Address, "v.example.com")
	eq(t, "port", p.Port, 443)
	eq(t, "method", p.Method, "aes-128-gcm")
	eq(t, "network", p.Network, "ws")
	eq(t, "headerType", p.HeaderType, "") // "none" over ws means nothing
	eq(t, "path", p.Path, "/ws?ed=2048")
	eq(t, "fingerprint", p.Fingerprint, "firefox")
	eq(t, "allowInsecure", p.AllowInsecure, true)

	grpc := `{"ps":"g","add":"g.example.com","port":443,"id":"id","aid":0,"net":"grpc","type":"multi","path":"svc","tls":"tls"}`
	p = mustParse(t, "vmess://"+base64.RawURLEncoding.EncodeToString([]byte(grpc)))
	eq(t, "serviceName", p.ServiceName, "svc")
	eq(t, "mode", p.Mode, "multi")
}

func TestTrojan(t *testing.T) {
	p := mustParse(t, "trojan://pa%40ss@t.example.com:443?sni=s.example.com&type=ws&path=%2Ft&insecure=1#t")
	eq(t, "password", p.Password, "pa@ss")
	eq(t, "security", p.Security, "tls")
	eq(t, "network", p.Network, "ws")
	eq(t, "allowInsecure", p.AllowInsecure, true)
}

func TestShadowsocks(t *testing.T) {
	userinfo := base64.RawURLEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:secret"))
	p := mustParse(t, "ss://"+userinfo+"@s.example.com:8388/?plugin=obfs-local%3Bobfs%3Dhttp%3Bobfs-host%3Dbing.com#SS")
	eq(t, "method", p.Method, "chacha20-ietf-poly1305")
	eq(t, "password", p.Password, "secret")
	eq(t, "port", p.Port, 8388)
	eq(t, "plugin", p.Plugin, "obfs-local")
	eq(t, "pluginOpts", p.PluginOpts, "obfs=http;obfs-host=bing.com")
	eq(t, "name", p.Name, "SS")

	// Shadowsocks 2022 keys are base64 themselves, so the userinfo is
	// percent-encoded plain text rather than base64 of it.
	p = mustParse(t, "ss://2022-blake3-aes-128-gcm:Y8PxFOE7smXq9UUTkPWfrg%3D%3D@[2001:db8::1]:443#2022")
	eq(t, "method", p.Method, "2022-blake3-aes-128-gcm")
	eq(t, "password", p.Password, "Y8PxFOE7smXq9UUTkPWfrg==")
	eq(t, "address", p.Address, "2001:db8::1")

	legacy := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:pw@1.2.3.4:1234"))
	p = mustParse(t, "ss://"+legacy+"#old")
	eq(t, "address", p.Address, "1.2.3.4")
	eq(t, "port", p.Port, 1234)
}

func TestHysteria2Hopping(t *testing.T) {
	p := mustParse(t, "hysteria2://auth@h.example.com:443,20000-30000/?sni=s.example.com&obfs=salamander&obfs-password=o&insecure=1#hy2")
	eq(t, "address", p.Address, "h.example.com")
	eq(t, "port", p.Port, 443)
	eq(t, "ports", p.Ports, "443,20000-30000")
	eq(t, "password", p.Password, "auth")
	eq(t, "obfs", p.Obfs, "salamander")
	eq(t, "allowInsecure", p.AllowInsecure, true)
	eq(t, "name", p.Name, "hy2")

	p = mustParse(t, "hy2://user:pass@h.example.com:8443?mport=30000-40000&up=50%20Mbps")
	eq(t, "password", p.Password, "user:pass")
	eq(t, "port", p.Port, 8443)
	eq(t, "ports", p.Ports, "30000-40000")
	eq(t, "up", p.UpMbps, 50)

	p = mustParse(t, "hysteria2://auth@h.example.com:443")
	eq(t, "ports", p.Ports, "") // a single port is no hopping
}

func TestTUICAndHysteria(t *testing.T) {
	p := mustParse(t, "tuic://uuid:pw@t.example.com:443?congestion_control=bbr&udp_relay_mode=quic&alpn=h3&allow_insecure=1")
	eq(t, "uuid", p.UUID, "uuid")
	eq(t, "password", p.Password, "pw")
	eq(t, "congestion", p.Congestion, "bbr")
	eq(t, "allowInsecure", p.AllowInsecure, true)

	p = mustParse(t, "hysteria://h.example.com:443?auth=a&upmbps=10&downmbps=50&obfsParam=o")
	eq(t, "password", p.Password, "a")
	eq(t, "obfs", p.Obfs, "o")
	eq(t, "down", p.DownMbps, 50)
}

func TestWireGuard(t *testing.T) {
	p := mustParse(t, "wireguard://cHJpdmF0ZQ%3D%3D@wg.example.com:51820?publickey=cHVibGlj%3D&address=10.0.0.2%2F32,fd00::2%2F128&reserved=1,2,3&mtu=1280#wg")
	eq(t, "privateKey", p.PrivateKey, "cHJpdmF0ZQ==")
	eq(t, "publicKey", p.PublicKey, "cHVibGlj=")
	eq(t, "localAddress", p.LocalAddress, "10.0.0.2/32,fd00::2/128")
	eq(t, "reserved", p.Reserved, "1,2,3")
	eq(t, "mtu", p.MTU, 1280)

	if _, err := Parse("wg://@wg.example.com:51820"); err == nil {
		t.Error("wireguard link without keys was accepted")
	}
}

func TestAnyTLSAndSOCKS(t *testing.T) {
	p := mustParse(t, "anytls://pw@a.example.com?sni=s.example.com#a")
	eq(t, "protocol", p.Protocol, model.ProtoAnyTLS)
	eq(t, "port", p.Port, 443)
	eq(t, "sni", p.SNI, "s.example.com")

	p = mustParse(t, "socks://"+base64.StdEncoding.EncodeToString([]byte("user:pass"))+"@127.0.0.1:1080#local")
	eq(t, "username", p.Username, "user")
	eq(t, "password", p.Password, "pass")

	p = mustParse(t, "socks5://u:p@10.0.0.1:1081")
	eq(t, "username", p.Username, "u")
	eq(t, "port", p.Port, 1081)
}

func TestRejects(t *testing.T) {
	for _, raw := range []string{"", "https://example.com/sub", "ssr://abc", "vless://"} {
		if _, err := Parse(raw); err == nil {
			t.Errorf("Parse(%q) succeeded", raw)
		}
	}
	if !IsLink("VLESS://x") || IsLink("https://x") {
		t.Error("IsLink scheme matching is off")
	}
}
