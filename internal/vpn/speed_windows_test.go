//go:build windows

package vpn

import (
	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/model"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDownloadSpeedThroughAllCores(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write(make([]byte, 256<<10))
	}))
	defer target.Close()
	port, err := freeLoopbackPort()
	if err != nil {
		t.Fatal(err)
	}
	server := fmt.Sprintf(`{"log":{"disabled":true},"inbounds":[{"type":"shadowsocks","listen":"127.0.0.1","listen_port":%d,"method":"aes-128-gcm","password":"speed-test"}],"outbounds":[{"type":"direct"}]}`, port)
	instance, cancel, err := newInstance([]byte(server), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if err = instance.Start(); err != nil {
		_ = instance.Close()
		t.Fatal(err)
	}
	defer instance.Close()
	p := model.Profile{Protocol: model.ProtoShadowsocks, Address: "127.0.0.1", Port: port, Method: "aes-128-gcm", Password: "speed-test"}
	for _, c := range cores.All {
		t.Run(string(c), func(t *testing.T) {
			result, err := downloadSpeed(p, c, target.URL, 3*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if result.Bytes != 256<<10 || result.DownloadMbps <= 0 || result.Core != c {
				t.Fatalf("bad result %+v", result)
			}
			if _, err = downloadSpeed(p, c, target.URL+"/fail", 3*time.Second); err == nil {
				t.Fatal("accepted failed HTTP response")
			}
		})
	}
}
