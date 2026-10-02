package store

import (
	"TomorrowClient/internal/model"
	"testing"
)

func TestProfileSpeedPersistenceAndInvalidation(t *testing.T) {
	s := newTestStore(t)
	p := model.Profile{ID: "server", SubID: "sub", Name: "Server", Address: "example.invalid", Port: 443}
	s.profiles = []model.Profile{p}
	s.subs = []model.Subscription{{ID: "sub", UpdatedAt: 10}}
	result := model.ProfileSpeed{DownloadMbps: 123.45, Bytes: 25000000, DurationMs: 1500, Core: "xray", MeasuredAt: 20}
	if ok, err := s.SaveProfileSpeed(p, 10, result); err != nil || !ok {
		t.Fatalf("save: %v %v", ok, err)
	}
	reloaded := &Store{dir: s.dir, settings: model.DefaultSettings()}
	reloaded.load()
	got, _ := reloaded.Profile(p.ID)
	if got.Speed == nil || got.Speed.DownloadMbps != result.DownloadMbps {
		t.Fatal("measurement did not survive reload")
	}
	if err := s.UpsertProfile(p); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Profile(p.ID)
	if got.Speed == nil {
		t.Fatal("unchanged profile lost result")
	}
	if err := s.ReplaceSubProfiles("sub", []model.Profile{got}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Profile(p.ID)
	if got.Speed != nil {
		t.Fatal("subscription refresh kept stale result")
	}
	s.subs[0].UpdatedAt = 11
	if ok, err := s.SaveProfileSpeed(p, 10, result); err != nil || ok {
		t.Fatal("old subscription revision accepted")
	}
	s.subs[0].UpdatedAt = 10
	changed := p
	changed.Port = 8443
	s.profiles = []model.Profile{changed}
	if ok, err := s.SaveProfileSpeed(p, 10, result); err != nil || ok {
		t.Fatal("changed profile accepted stale result")
	}
	s.profiles = nil
	if ok, err := s.SaveProfileSpeed(p, 10, result); err != nil || ok {
		t.Fatal("deleted profile resurrected")
	}
}
