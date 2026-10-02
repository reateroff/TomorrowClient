package store

import (
	"TomorrowClient/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestAppearanceDefaultsAndSavedChoices(t *testing.T) {
	fresh := newTestStore(t)
	fresh.load()
	cfg := fresh.Settings()
	if cfg.Theme != "smoke" || cfg.Accent != "sunset-mist" || cfg.Font != "rubik" || cfg.Radius != "soft" || cfg.NavPosition != "top" || cfg.Animation != "fade" {
		t.Fatalf("unexpected fresh appearance: %+v", cfg)
	}
	saved := `{"theme":"midnight","accent":"#3f9b85","font":"mono","radius":"custom","customRadius":22,"navPosition":"right","animation":"none"}`
	if err := os.WriteFile(filepath.Join(fresh.dir, "settings.json"), []byte(saved), 0600); err != nil {
		t.Fatal(err)
	}
	restored := &Store{dir: fresh.dir, settings: model.DefaultSettings()}
	restored.load()
	got := restored.Settings()
	if got.Theme != "midnight" || got.Accent != "#3f9b85" || got.Font != "mono" || got.Radius != "custom" || got.CustomRadius != 22 || got.NavPosition != "right" || got.Animation != "none" {
		t.Fatalf("saved appearance overwritten: %+v", got)
	}
}
