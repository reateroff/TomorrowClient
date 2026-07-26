package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"TomorrowClient/internal/model"
)

// newTestStore builds a store over a throwaway directory. The exported New()
// always targets the real user config dir, which a destructive test must not
// touch.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{dir: t.TempDir(), settings: model.DefaultSettings()}
}

func TestResetAllClearsEverything(t *testing.T) {
	s := newTestStore(t)

	if err := s.UpsertProfile(model.Profile{ID: "p1", Name: "server"}); err != nil {
		t.Fatalf("UpsertProfile: %v", err)
	}
	if err := s.UpsertSubscription(model.Subscription{ID: "s1", Name: "sub"}); err != nil {
		t.Fatalf("UpsertSubscription: %v", err)
	}
	cfg := s.Settings()
	cfg.DevMode = true
	cfg.DNS = "8.8.8.8"
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	if err := s.ResetAll(); err != nil {
		t.Fatalf("ResetAll: %v", err)
	}

	if got := len(s.Profiles()); got != 0 {
		t.Errorf("profiles = %d, want 0", got)
	}
	if got := len(s.Subscriptions()); got != 0 {
		t.Errorf("subscriptions = %d, want 0", got)
	}
	if !reflect.DeepEqual(s.Settings(), model.DefaultSettings()) {
		t.Errorf("settings not restored to defaults: %+v", s.Settings())
	}
	if s.Settings().DevMode {
		t.Error("DevMode survived a full reset")
	}

	// A restart reloads from disk, so the files themselves must be empty too.
	reloaded := &Store{dir: s.dir, settings: model.DefaultSettings()}
	reloaded.load()
	if len(reloaded.Profiles()) != 0 || len(reloaded.Subscriptions()) != 0 {
		t.Error("deleted data came back after reload")
	}
	if reloaded.Settings().DNS != model.DefaultSettings().DNS {
		t.Errorf("settings on disk not reset: DNS = %q", reloaded.Settings().DNS)
	}

	for _, name := range []string{"profiles.json", "subscriptions.json", "settings.json"} {
		if _, err := os.Stat(filepath.Join(s.dir, name)); err != nil {
			t.Errorf("%s missing after reset: %v", name, err)
		}
	}

	// The wiped lists must serialise as [] rather than null so the frontend
	// never receives a null where it expects an array.
	b, err := os.ReadFile(filepath.Join(s.dir, "profiles.json"))
	if err != nil {
		t.Fatalf("read profiles.json: %v", err)
	}
	var raw json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("profiles.json is not valid JSON: %v", err)
	}
	if string(b) == "null" {
		t.Error("profiles.json serialised as null")
	}
}
