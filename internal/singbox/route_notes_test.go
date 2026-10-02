package singbox

import (
	"TomorrowClient/internal/model"
	"encoding/json"
	"reflect"
	"testing"
)

func TestNetworkICMPMatcher(t *testing.T) {
	match, _ := nodeMatch(model.RouteNode{Type: "network", Values: []string{"TCP", "UDP", "ICMP", "sctp"}})
	if !reflect.DeepEqual(match["network"], []any{"tcp", "udp", "icmp"}) {
		t.Fatalf("unexpected networks: %#v", match)
	}
}
func TestNotesNeverChangeRoutes(t *testing.T) {
	s := model.AppSettings{RoutingMode: model.RoutingPro, Graph: model.RouteGraph{Final: model.ActionDirect}}
	a, _ := json.Marshal(buildRoute(s, nil).block)
	s.Graph.Notes = []model.RouteNote{{ID: "memo", Text: "all traffic via VPN", X: 100, Y: 20}}
	b, _ := json.Marshal(buildRoute(s, nil).block)
	if string(a) != string(b) {
		t.Fatalf("annotation changed routing: %s -> %s", a, b)
	}
}

func TestEasyGeoPresetsMatchPro(t *testing.T) {
	for _, final := range []string{model.ActionProxy, model.ActionDirect, model.ActionBlock} {
		s := model.DefaultSettings()
		s.Sniff = false
		s.RoutingMode = model.RoutingSimple
		s.SimpleFinal = final
		s.Rules = []model.RoutingRule{{Type: "geosite", Value: "category-ru", Action: "direct"}, {Type: "geoip", Value: "ru", Action: "direct"}, {Type: "geosite", Value: "category-ads-all", Action: "block"}, {Type: "geosite", Value: "youtube", Action: "proxy"}}
		easy := buildRoute(s, nil)
		s.RoutingMode = model.RoutingPro
		s.Graph = model.RouteGraph{Final: final}
		for _, r := range s.Rules {
			s.Graph.Nodes = append(s.Graph.Nodes, model.RouteNode{Type: r.Type, Values: []string{r.Value}, Action: r.Action})
		}
		pro := buildRoute(s, nil)
		if !reflect.DeepEqual(easy, pro) {
			t.Fatalf("Easy/Pro mismatch: %#v / %#v", easy, pro)
		}
		if len(easy.block["rule_set"].([]any)) != 4 || len(easy.dnsRules) != 2 {
			t.Fatalf("missing geo sets/split DNS: %#v", easy)
		}
	}
	s := model.DefaultSettings()
	s.SimpleFinal = ""
	if buildRoute(s, nil).block["final"] != "proxy" {
		t.Fatal("old Easy settings changed catch-all")
	}
}
