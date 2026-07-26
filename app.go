package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/energye/systray"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"TomorrowClient/internal/link"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
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
	ctx       context.Context
	store     *store.Store
	engine    *vpn.Engine
	mToggle   *systray.MenuItem // tray "connect/disconnect" item, relabeled on status
	startedAt time.Time         // process start, reported as uptime in the dev tools

	// Developer-tools status simulation. simStop cancels the goroutine that
	// animates fake traffic counters; nil when no simulation is running.
	simMu   sync.Mutex
	simStop chan struct{}
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
		BuiltWith: "Wails · Go · React · sing-box",
	}
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startedAt = time.Now()

	st, err := store.New()
	if err != nil {
		runtime.LogError(ctx, "store init: "+err.Error())
	}
	a.store = st

	// The engine pushes status snapshots to the frontend over the
	// "vpn:status" event, and each core log line over "vpn:log". The status
	// callback also relabels the tray connect/disconnect item.
	a.engine = vpn.New(
		func(s model.Status) {
			runtime.EventsEmit(ctx, "vpn:status", s)
			a.updateTrayStatus(s)
		},
		func(line string) { runtime.EventsEmit(ctx, "vpn:log", line) },
	)

	a.setupTray()

	// Auto-connect to the last active profile if the user enabled it.
	if s := a.store.Settings(); s.AutoConnect && s.ActiveProfileID != "" {
		if p, ok := a.store.Profile(s.ActiveProfileID); ok {
			go func() { _ = a.engine.Connect(p, s) }()
		}
	}
}

// shutdown makes sure the tunnel is torn down when the window closes, and the
// tray icon is removed.
func (a *App) shutdown(ctx context.Context) {
	if a.engine != nil {
		a.engine.Disconnect()
	}
	systray.Quit()
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

// PingResult is one server's latency cell.
type PingResult struct {
	// LatencyMs is the ICMP round trip, or -1 when the host does not answer
	// echo requests. Plenty of servers filter ICMP while working perfectly, so
	// -1 on its own is not a failure — read it together with OK.
	LatencyMs int `json:"latencyMs"`
	// OK reports whether a real request actually made it through the server.
	OK bool `json:"ok"`
}

// PingLatency measures only the ICMP round trip to a profile's server.
//
// The main screen refreshes this on a short timer, so it has to stay cheap: a
// few 32-byte echo packets and nothing else. PingProfile additionally stands up
// a private core and pushes a real request through the server, which is the
// right thing when judging servers you are not connected to, and pure waste for
// the one already carrying your traffic — if it were broken there would be no
// connection to report on.
func (a *App) PingLatency(id string) int {
	p, ok := a.store.Profile(id)
	if !ok || p.Address == "" {
		return -1
	}
	d, err := vpn.PingICMP(p.Address)
	if err != nil {
		return -1
	}
	return int(d.Milliseconds())
}

// PingProfile reports a profile's network latency and whether it works at all.
//
// The two are measured separately on purpose. ICMP gives the honest round trip,
// which is what a latency figure should say; timing a request through the proxy
// instead would fold in TCP setup, the TLS handshake and the far-side fetch, and
// read several hundred milliseconds on a link that is really a few tens. But
// ICMP says nothing about whether the profile is usable, so a real request
// through the server decides that separately — it only completes when the
// transport, TLS and credentials are all good.
func (a *App) PingProfile(id string) PingResult {
	p, ok := a.store.Profile(id)
	if !ok || p.Address == "" || p.Port == 0 {
		return PingResult{LatencyMs: -1}
	}

	var (
		wg      sync.WaitGroup
		latency = -1
		works   bool
	)
	wg.Add(2)

	// Run both together: a server that filters ICMP would otherwise hold up the
	// verdict for the full echo timeout before the real check even started.
	go func() {
		defer wg.Done()
		if d, err := vpn.PingICMP(p.Address); err == nil {
			latency = int(d.Milliseconds())
		}
	}()
	go func() {
		defer wg.Done()
		_, err := vpn.ProbeLatency(p)
		works = err == nil
	}()

	wg.Wait()
	return PingResult{LatencyMs: latency, OK: works}
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

	profiles, info, err := sub.Fetch(url, s.ID)
	if err != nil {
		return model.Subscription{}, err
	}

	// What the user typed wins; otherwise use the name the provider reports,
	// and only fall back to the URL host when there is nothing better.
	if s.Name == "" {
		s.Name = info.Title
	}
	if s.Name == "" {
		s.Name = subNameFromURL(url)
	}
	if err := a.store.ReplaceSubProfiles(s.ID, profiles); err != nil {
		return model.Subscription{}, err
	}
	s.Count = len(profiles)
	s.UpdatedAt = time.Now().UnixMilli()
	applyUserinfo(&s, info)
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
	profiles, info, err := sub.Fetch(s.URL, s.ID)
	if err != nil {
		return model.Subscription{}, err
	}
	// Adopt the provider's name only if the current one was our own fallback to
	// the URL host — a name the user chose must survive a refresh.
	if info.Title != "" && s.Name == subNameFromURL(s.URL) {
		s.Name = info.Title
	}
	if err := a.store.ReplaceSubProfiles(s.ID, profiles); err != nil {
		return model.Subscription{}, err
	}
	s.Count = len(profiles)
	s.UpdatedAt = time.Now().UnixMilli()
	applyUserinfo(&s, info)
	if err := a.store.UpsertSubscription(s); err != nil {
		return model.Subscription{}, err
	}
	return s, nil
}

// applyUserinfo copies traffic/expiry metadata from a fetch into a subscription.
func applyUserinfo(s *model.Subscription, info sub.Meta) {
	s.Upload = info.Upload
	s.Download = info.Download
	s.Total = info.Total
	s.Expire = info.Expire
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
	// The migration marker is backend bookkeeping. Stamping it here means the
	// frontend can never send it back as 0 and make a migration run twice.
	s.SettingsVersion = model.SettingsVersion
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

// PreviewConfig builds and returns the core config JSON for the active profile
// and current settings. Used by the developer tools to inspect what the app
// actually hands to the running core.
func (a *App) PreviewConfig() (string, error) {
	s := a.store.Settings()
	if s.ActiveProfileID == "" {
		return "", fmt.Errorf("профиль не выбран")
	}
	p, ok := a.store.Profile(s.ActiveProfileID)
	if !ok {
		return "", fmt.Errorf("профиль не найден")
	}
	b, err := singbox.Build(p, s)
	if err != nil {
		return "", err
	}
	return string(b), nil
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
