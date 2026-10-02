// Package store persists profiles and settings to JSON files under the user's
// config directory (%APPDATA%\TomorrowClient on Windows).
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"TomorrowClient/internal/model"
)

const appDir = "TomorrowClient"

// Store is a small thread-safe JSON-backed store.
type Store struct {
	mu       sync.Mutex
	dir      string
	settings model.AppSettings
	profiles []model.Profile
	subs     []model.Subscription
}

// New opens (or creates) the store in the user config directory and loads any
// existing data from disk.
func New() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, appDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, settings: model.DefaultSettings()}
	s.load()
	if s.migrate() {
		_ = writeJSON(s.settingsPath(), s.settings)
	}
	return s, nil
}

// NewMemory returns a store that keeps everything in memory, for when the
// config directory cannot be used: the app still runs, it just forgets on
// exit. Writes go to a throwaway temp directory.
func NewMemory() *Store {
	dir, err := os.MkdirTemp("", "TomorrowClient-")
	if err != nil {
		dir = os.TempDir()
	}
	return &Store{dir: dir, settings: model.DefaultSettings()}
}

// migrate applies one-time rewrites to settings loaded from an older file and
// reports whether anything changed. Changing a default only affects fresh
// installs — an existing settings.json keeps the old value forever — so a value
// that must actually move needs a migration here.
func (s *Store) migrate() bool {
	legacyStack := s.settings.Stack != ""
	s.settings.Stack = ""
	if s.settings.SettingsVersion >= model.SettingsVersion {
		return legacyStack
	}

	// v2: several cores, picked per profile by default. Before v2 sing-box was
	// the only core, so a stored "sing-box" is the old default, not a choice.
	if s.settings.SettingsVersion < 2 && (s.settings.Core == model.CoreSingBox || s.settings.Core == "") {
		s.settings.Core = model.CoreAuto
	}

	// v3: Xray replaces the previous automatic default. Explicit core choices
	// are preserved; Auto chosen after this migration remains a valid opt-in.
	if s.settings.SettingsVersion < 3 && (s.settings.Core == model.CoreAuto || s.settings.Core == "") {
		s.settings.Core = model.CoreXray
	}
	s.settings.SettingsVersion = model.SettingsVersion
	return true
}

// Dir returns the directory holding the store's JSON files.
func (s *Store) Dir() string { return s.dir }

func (s *Store) settingsPath() string { return filepath.Join(s.dir, "settings.json") }
func (s *Store) profilesPath() string { return filepath.Join(s.dir, "profiles.json") }
func (s *Store) subsPath() string     { return filepath.Join(s.dir, "subscriptions.json") }

// load reads the data files if present; missing files fall back to defaults.
func (s *Store) load() {
	if b, err := os.ReadFile(s.settingsPath()); err == nil {
		// Unmarshalling merges onto DefaultSettings, so a key the file omits
		// keeps the default. That is wrong for the version marker: a file
		// written before migrations existed has no settingsVersion key and must
		// read as 0, not as the current version. Zero it first.
		s.settings.SettingsVersion = 0
		_ = json.Unmarshal(b, &s.settings)
	}
	if b, err := os.ReadFile(s.profilesPath()); err == nil {
		_ = json.Unmarshal(b, &s.profiles)
	}
	if b, err := os.ReadFile(s.subsPath()); err == nil {
		_ = json.Unmarshal(b, &s.subs)
	}
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	// Write to a temp file then rename for an atomic replace.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Settings returns a copy of the current settings.
func (s *Store) Settings() model.AppSettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

// SaveSettings persists the given settings.
func (s *Store) SaveSettings(cfg model.AppSettings) error {
	// Enforce this in the store too: JSON editing/imports cannot override TUN.
	cfg.Stack = ""
	cfg.SettingsVersion = model.SettingsVersion
	if cfg.Core == "" {
		cfg.Core = model.CoreXray
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = cfg
	return writeJSON(s.settingsPath(), s.settings)
}

// ResetAll wipes every stored profile, subscription and setting, returning the
// app to a clean-install state. All three files are rewritten so nothing from
// the previous installation survives a restart.
func (s *Store) ResetAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.profiles = nil
	s.subs = nil
	s.settings = model.DefaultSettings()

	if err := writeJSON(s.profilesPath(), []model.Profile{}); err != nil {
		return err
	}
	if err := writeJSON(s.subsPath(), []model.Subscription{}); err != nil {
		return err
	}
	return writeJSON(s.settingsPath(), s.settings)
}

// Profiles returns a copy of the stored profiles slice.
func (s *Store) Profiles() []model.Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Profile, len(s.profiles))
	copy(out, s.profiles)
	return out
}

// Profile looks up a single profile by id.
func (s *Store) Profile(id string) (model.Profile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.profiles {
		if p.ID == id {
			return p, true
		}
	}
	return model.Profile{}, false
}

// UpsertProfile inserts or replaces a profile (matched by ID) and persists.
func (s *Store) UpsertProfile(p model.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.profiles {
		if s.profiles[i].ID == p.ID {
			if !sameSpeedProfile(s.profiles[i], p) {
				p.Speed = nil
			} else {
				p.Speed = s.profiles[i].Speed
			}
			s.profiles[i] = p
			return writeJSON(s.profilesPath(), s.profiles)
		}
	}
	s.profiles = append(s.profiles, p)
	return writeJSON(s.profilesPath(), s.profiles)
}

// DeleteProfile removes a profile by id and persists.
func (s *Store) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.profiles[:0]
	for _, p := range s.profiles {
		if p.ID != id {
			out = append(out, p)
		}
	}
	s.profiles = out
	return writeJSON(s.profilesPath(), s.profiles)
}

// --- Subscriptions ---

// Subscriptions returns a copy of the stored subscriptions.
func (s *Store) Subscriptions() []model.Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Subscription, len(s.subs))
	copy(out, s.subs)
	return out
}

// Subscription looks up a single subscription by id.
func (s *Store) Subscription(id string) (model.Subscription, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.subs {
		if sub.ID == id {
			return sub, true
		}
	}
	return model.Subscription{}, false
}

// UpsertSubscription inserts or replaces a subscription (matched by ID) and
// persists.
func (s *Store) UpsertSubscription(sub model.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.subs {
		if s.subs[i].ID == sub.ID {
			s.subs[i] = sub
			return writeJSON(s.subsPath(), s.subs)
		}
	}
	s.subs = append(s.subs, sub)
	return writeJSON(s.subsPath(), s.subs)
}

// ReplaceSubProfiles swaps every profile belonging to subID for the given set,
// persisting once. Used when a subscription is imported or refreshed.
func (s *Store) ReplaceSubProfiles(subID string, profiles []model.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]model.Profile, 0, len(s.profiles))
	for _, p := range s.profiles {
		if p.SubID != subID {
			kept = append(kept, p)
		}
	}
	for i := range profiles {
		profiles[i].Speed = nil
	}
	kept = append(kept, profiles...)
	s.profiles = kept
	return writeJSON(s.profilesPath(), s.profiles)
}

// DeleteSubscription removes a subscription and all of its profiles, persisting
// both files.
func (s *Store) DeleteSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subs[:0]
	for _, sub := range s.subs {
		if sub.ID != id {
			subs = append(subs, sub)
		}
	}
	s.subs = subs
	if err := writeJSON(s.subsPath(), s.subs); err != nil {
		return err
	}

	kept := make([]model.Profile, 0, len(s.profiles))
	for _, p := range s.profiles {
		if p.SubID != id {
			kept = append(kept, p)
		}
	}
	s.profiles = kept
	return writeJSON(s.profilesPath(), s.profiles)
}
