package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"TomorrowClient/internal/link"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/startup"
	"TomorrowClient/internal/store"
	"TomorrowClient/internal/sub"
	"TomorrowClient/internal/vpn"
)

// Version is the client version shown on the About screen.
const Version = "1.0.0"

// App is the Wails-bound application object. Every exported method here is
// callable from the React frontend.
type App struct {
	ctx    context.Context
	store  *store.Store
	engine *vpn.Engine
}

// AppInfo is the metadata shown on the "About" screen.
type AppInfo struct {
	Version   string `json:"version"`
	Copyright string `json:"copyright"`
	BuiltWith string `json:"builtWith"`
}

// GetAppInfo returns static build/about information for the UI.
func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Version:   Version,
		Copyright: fmt.Sprintf("© %d TomorrowClient", time.Now().Year()),
		BuiltWith: "Wails · Go · React · sing-box · xray-core",
	}
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	st, err := store.New()
	if err != nil {
		runtime.LogError(ctx, "store init: "+err.Error())
	}
	a.store = st

	// The engine pushes status snapshots to the frontend over the
	// "vpn:status" event, and each core log line over "vpn:log".
	a.engine = vpn.New(
		func(s model.Status) { runtime.EventsEmit(ctx, "vpn:status", s) },
		func(line string) { runtime.EventsEmit(ctx, "vpn:log", line) },
	)

	// Auto-connect to the last active profile if the user enabled it.
	if s := a.store.Settings(); s.AutoConnect && s.ActiveProfileID != "" {
		if p, ok := a.store.Profile(s.ActiveProfileID); ok {
			go func() { _ = a.engine.Connect(p, s) }()
		}
	}
}

// shutdown makes sure the tunnel is torn down when the window closes.
func (a *App) shutdown(ctx context.Context) {
	if a.engine != nil {
		a.engine.Disconnect()
	}
}

// --- Profiles ---

// GetProfiles returns all saved profiles.
func (a *App) GetProfiles() []model.Profile {
	return a.store.Profiles()
}

// ImportLink parses a share link, saves it as a new profile, and returns it.
func (a *App) ImportLink(raw string) (model.Profile, error) {
	p, err := link.Parse(raw)
	if err != nil {
		return model.Profile{}, err
	}
	if err := a.store.UpsertProfile(p); err != nil {
		return model.Profile{}, err
	}
	return p, nil
}

// SaveProfile inserts or updates a profile.
func (a *App) SaveProfile(p model.Profile) error {
	return a.store.UpsertProfile(p)
}

// DeleteProfile removes a profile by id.
func (a *App) DeleteProfile(id string) error {
	return a.store.DeleteProfile(id)
}

// --- Subscriptions ---

// GetSubscriptions returns all saved subscriptions.
func (a *App) GetSubscriptions() []model.Subscription {
	return a.store.Subscriptions()
}

// AddSubscription registers a new subscription URL, fetches it, and imports the
// servers it contains. The name is optional; a fallback is derived from the URL
// host when it is empty. Returns the stored subscription with its fresh count.
func (a *App) AddSubscription(name, url string) (model.Subscription, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return model.Subscription{}, fmt.Errorf("empty subscription url")
	}
	s := model.Subscription{
		ID:   uuid.NewString(),
		Name: strings.TrimSpace(name),
		URL:  url,
	}
	if s.Name == "" {
		s.Name = subNameFromURL(url)
	}

	profiles, err := sub.Fetch(url, s.ID)
	if err != nil {
		return model.Subscription{}, err
	}
	if err := a.store.ReplaceSubProfiles(s.ID, profiles); err != nil {
		return model.Subscription{}, err
	}
	s.Count = len(profiles)
	s.UpdatedAt = time.Now().UnixMilli()
	if err := a.store.UpsertSubscription(s); err != nil {
		return model.Subscription{}, err
	}
	return s, nil
}

// UpdateSubscription re-fetches an existing subscription and replaces its
// servers with the fresh set.
func (a *App) UpdateSubscription(id string) (model.Subscription, error) {
	s, ok := a.store.Subscription(id)
	if !ok {
		return model.Subscription{}, fmt.Errorf("subscription not found")
	}
	profiles, err := sub.Fetch(s.URL, s.ID)
	if err != nil {
		return model.Subscription{}, err
	}
	if err := a.store.ReplaceSubProfiles(s.ID, profiles); err != nil {
		return model.Subscription{}, err
	}
	s.Count = len(profiles)
	s.UpdatedAt = time.Now().UnixMilli()
	if err := a.store.UpsertSubscription(s); err != nil {
		return model.Subscription{}, err
	}
	return s, nil
}

// DeleteSubscription removes a subscription and all servers it imported.
func (a *App) DeleteSubscription(id string) error {
	return a.store.DeleteSubscription(id)
}

// subNameFromURL derives a friendly name from a subscription URL host.
func subNameFromURL(raw string) string {
	s := raw
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return "Подписка"
	}
	return s
}

// --- Settings ---

// GetSettings returns the persisted settings.
func (a *App) GetSettings() model.AppSettings {
	return a.store.Settings()
}

// SaveSettings persists settings and applies the Windows autostart task to
// match the LaunchAtStartup flag.
func (a *App) SaveSettings(s model.AppSettings) error {
	if err := startup.Set(s.LaunchAtStartup); err != nil {
		runtime.LogError(a.ctx, "autostart: "+err.Error())
	}
	return a.store.SaveSettings(s)
}

// GetLogs returns the buffered core log lines (used on initial load of the
// Logs tab).
func (a *App) GetLogs() []string {
	return a.engine.Logs()
}

// ClearLogs empties the in-memory core log buffer.
func (a *App) ClearLogs() {
	a.engine.ClearLogs()
}

// --- Connection ---

// Connect brings the tunnel up using the active profile from settings (or the
// explicitly given id when non-empty).
func (a *App) Connect(profileID string) error {
	s := a.store.Settings()
	if profileID == "" {
		profileID = s.ActiveProfileID
	}
	if profileID == "" {
		return fmt.Errorf("no profile selected")
	}
	p, ok := a.store.Profile(profileID)
	if !ok {
		return fmt.Errorf("profile not found")
	}
	// Remember the chosen profile as active.
	s.ActiveProfileID = profileID
	_ = a.store.SaveSettings(s)
	return a.engine.Connect(p, s)
}

// Disconnect tears the tunnel down.
func (a *App) Disconnect() {
	a.engine.Disconnect()
}

// GetStatus returns the current connection snapshot (used on initial load).
func (a *App) GetStatus() model.Status {
	return a.engine.Status()
}
