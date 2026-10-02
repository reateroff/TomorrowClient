//go:build windows

package main

import (
	"TomorrowClient/internal/model"
	"encoding/json"
	"strings"
	"testing"
)

func TestRoutingNotesRoundTrip(t *testing.T) {
	d := routingDocument{Format: "TomorrowClient.routing", Version: 1, Mode: "pro", Graph: model.RouteGraph{Final: "", FinalHidden: true, Notes: []model.RouteNote{{ID: "note-1", Text: "Игры — напрямую\nРабота — VPN", X: 40, Y: 60}}, Nodes: []model.RouteNode{{ID: "icmp", Type: "network", Values: []string{"icmp"}, Action: "direct"}}}}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseRoutingDocument(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Graph.Notes) != 1 || parsed.Graph.Notes[0].Text != d.Graph.Notes[0].Text || !parsed.Graph.FinalHidden {
		t.Fatal("lost notes or catch-all visibility")
	}
	for _, kind := range []string{"duplicate", "reserved", "too-long"} {
		bad := d
		bad.Graph.Notes = append([]model.RouteNote{}, d.Graph.Notes...)
		switch kind {
		case "duplicate":
			bad.Graph.Notes[0].ID = "icmp"
		case "reserved":
			bad.Graph.Notes[0].ID = "final"
		case "too-long":
			bad.Graph.Notes[0].Text = strings.Repeat("x", 16001)
		}
		raw, _ := json.Marshal(bad)
		if _, err := parseRoutingDocument(raw); err == nil {
			t.Fatalf("accepted %s note", kind)
		}
	}
}
