//go:build windows

package main

import (
	"TomorrowClient/internal/model"
	"encoding/json"
	"testing"
)

func TestEasyPresetRoutingRoundTrip(t *testing.T) {
	d := routingDocument{Format: "TomorrowClient.routing", Version: 1, Mode: "simple", SimpleFinal: "direct", Rules: []model.RoutingRule{{Type: "geosite", Value: "youtube", Action: "proxy"}, {Type: "geoip", Value: "ru", Action: "direct"}}}
	b, _ := json.Marshal(d)
	got, err := parseRoutingDocument(b)
	if err != nil || got.SimpleFinal != "direct" || len(got.Rules) != 2 {
		t.Fatalf("round-trip: %+v %v", got, err)
	}
	d.SimpleFinal = "invalid"
	b, _ = json.Marshal(d)
	if _, err = parseRoutingDocument(b); err == nil {
		t.Fatal("invalid final accepted")
	}
}
