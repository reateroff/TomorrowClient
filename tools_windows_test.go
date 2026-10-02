//go:build windows

package main

import "testing"

func TestNewerVersion(t *testing.T) {
	for _, c := range []struct {
		r, l string
		want bool
	}{{"v1.2.0", "1.1.0", true}, {"1.1.0", "1.1.0", false}, {"v1.0.9", "1.1.0", false}, {"bad", "1.1.0", false}, {"v1.2.0-beta", "1.1.0", false}} {
		if got := newerVersion(c.r, c.l); got != c.want {
			t.Errorf("%v: %v", c, got)
		}
	}
}
func TestRoutingDocument(t *testing.T) {
	valid := []byte(`{"format":"TomorrowClient.routing","version":1,"mode":"pro","graph":{"final":"proxy","nodes":[{"id":"ru","type":"geosite","values":["ru"],"action":"direct"}]}}`)
	if _, err := parseRoutingDocument(valid); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{}`, `{"format":"TomorrowClient.routing","version":9,"mode":"pro"}`, `{"format":"TomorrowClient.routing","version":1,"mode":"pro","graph":{"final":"unknown"}}`} {
		if _, err := parseRoutingDocument([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}
func TestGithubAssetURL(t *testing.T) {
	if !githubAssetURL("https://github.com/reateroff/TomorrowClient/releases/download/v1.2.0/app.exe") {
		t.Fatal("rejected official asset")
	}
	for _, s := range []string{"http://github.com/reateroff/TomorrowClient/releases/download/v1/a.exe", "https://evil.com/a.exe", "https://github.com/other/repo/releases/download/v1/a.exe"} {
		if githubAssetURL(s) {
			t.Errorf("accepted %s", s)
		}
	}
}

func TestRoutingRejectsInvalidValues(t *testing.T) {
	for _, v := range []struct{ kind, value string }{{"ip", "999.2.3.4"}, {"ip", "10.0.0.0/99"}, {"port", "0"}, {"port", "70000"}, {"port", "2000-1000"}, {"domain_regex", "["}, {"network", "sctp"}, {"process", "foo"}, {"geosite", "../foo"}} {
		if err := validateRouteValue(v.kind, v.value); err == nil {
			t.Errorf("accepted %+v", v)
		}
	}
	for _, v := range []struct{ kind, value string }{{"ip", "10.0.0.0/8"}, {"ip", "::1"}, {"port", "443"}, {"port", "1000-2000"}, {"domain_regex", "^example\\.com$"}, {"network", "tcp"}, {"network", "icmp"}, {"network", "ICMP"}, {"process", "chrome.exe"}, {"geosite", "category-ru"}} {
		if err := validateRouteValue(v.kind, v.value); err != nil {
			t.Errorf("rejected %+v: %v", v, err)
		}
	}
}
