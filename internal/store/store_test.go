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

// TestMigrateStackFromLegacyFile covers the v1 migration end to end through
// load(): a settings file written before versioning has no settingsVersion key,
// and load() merges onto DefaultSettings, so the marker has to be forced to 0
// or the migration silently believes it already ran.
func TestMigrateStackFromLegacyFile(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"core":"sing-box","stack":"gvisor","dns":"1.1.1.1"}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(legacy), 0o644); err != nil {
		t.Fatalf("seed settings.json: %v", err)
	}

	s := &Store{dir: dir, settings: model.DefaultSettings()}
	s.load()
	if !s.migrate() {
		t.Fatal("migrate reported no change for a pre-v1 file")
	}
	if got := s.Settings().Stack; got != "" {
		t.Errorf("stack = %q, want native empty stack", got)
	}
	if got := s.Settings().SettingsVersion; got != model.SettingsVersion {
		t.Errorf("version = %d, want %d", got, model.SettingsVersion)
	}

	if s.migrate() {
		t.Error("second migration must be a no-op")
	}
	// Even a manually reintroduced legacy override is discarded on next load.
	s.settings.Stack = "gvisor"
	if !s.migrate() || s.Settings().Stack != "" {
		t.Error("legacy stack override survived")
	}

}

// A fresh install carries the current version already, so nothing may migrate.
func TestMigrateSkipsFreshInstall(t *testing.T) {
	s := &Store{dir: t.TempDir(), settings: model.DefaultSettings()}
	s.load()
	if s.migrate() {
		t.Error("migrate ran on a fresh install")
	}
	if got := s.Settings().Stack; got != "" {
		t.Errorf("default stack = %q, want native empty stack", got)
	}
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

// Chained v2/v3 migration replaces the old default with Xray.
// A v1 file is the realistic input: it carries a version and still migrates.
func TestMigrateCoreToXray(t *testing.T) {
	dir := t.TempDir()
	v1 := `{"settingsVersion":1,"core":"sing-box","stack":"gvisor"}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(v1), 0o644); err != nil {
		t.Fatalf("seed settings.json: %v", err)
	}

	s := &Store{dir: dir, settings: model.DefaultSettings()}
	s.load()
	if !s.migrate() {
		t.Fatal("migrate reported no change for a v1 file")
	}
	if got := s.Settings().Core; got != model.CoreXray {
		t.Errorf("core = %q, want %q", got, model.CoreXray)
	}
	if got := s.Settings().Stack; got != "" {
		t.Errorf("legacy stack survived: %q", got)
	}

	// A sing-box picked after the migration is deliberate and must stick.
	s.settings.Core = model.CoreSingBox
	if s.migrate() {
		t.Error("migrate ran a second time")
	}
	if got := s.Settings().Core; got != model.CoreSingBox {
		t.Errorf("migrate overwrote a deliberate choice: core = %q", got)
	}
}

func TestV3CoreMigrationPreservesExplicitChoices(t *testing.T) {
	for _, core := range []model.Core{model.CoreAuto, model.CoreSingBox, model.CoreTodayCore, model.CoreXray, model.CoreMihomo} {
		t.Run(string(core), func(t *testing.T) {
			s := newTestStore(t)
			s.settings.SettingsVersion = 2
			s.settings.Core = core
			s.settings.Stack = "system"
			s.migrate()
			want := core
			if core == model.CoreAuto {
				want = model.CoreXray
			}
			if s.Settings().Core != want || s.Settings().Stack != "" {
				t.Fatalf("unexpected migration: %+v", s.Settings())
			}
			if s.migrate() {
				t.Fatal("migration repeated")
			}
		})
	}
}
func TestStoreRejectsLegacyStackOverrides(t *testing.T) {
	s := newTestStore(t)
	for _, core := range []model.Core{model.CoreAuto, model.CoreSingBox, model.CoreTodayCore, model.CoreXray, model.CoreMihomo} {
		cfg := model.DefaultSettings()
		cfg.Core = core
		cfg.Stack = "gvisor"
		if err := s.SaveSettings(cfg); err != nil {
			t.Fatal(err)
		}
		if s.Settings().Stack != "" || s.Settings().Core != core {
			t.Fatalf("unexpected saved settings: %+v", s.Settings())
		}
	}
	if model.DefaultSettings().Core != model.CoreXray {
		t.Fatal("fresh install must default to Xray")
	}
}
