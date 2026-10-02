package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/energye/systray"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/device"
	"TomorrowClient/internal/link"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
	"TomorrowClient/internal/startup"
	"TomorrowClient/internal/store"
	"TomorrowClient/internal/sub"
	"TomorrowClient/internal/vpn"
)

// Version is the client version shown on the About screen.
const Version = "1.1.0"

// App is the Wails-bound application object. Every exported method here is
// callable from the React frontend.
type App struct {
	ctx       context.Context
	store     *store.Store
	engine    *vpn.Engine
	mToggle   *systray.MenuItem // tray "connect/disconnect" item, relabeled on status
	startedAt time.Time         // process start, reported as uptime in the dev tools
	tools     toolState
	storeErr  error // why the on-disk store could not open, if it could not

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
	// License is stated in the UI because the app links sing-box, TodayCore
	// and mihomo, which are GPL-3.0-or-later: the terms have to reach the
	// person running it, not just whoever reads the repository.
	License string `json:"license"`
}

// GetAppInfo returns static build/about information for the UI.
func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Version:   Version,
		Copyright: fmt.Sprintf("© %d TomorrowClient", time.Now().Year()),
		BuiltWith: "Wails · Go · React · sing-box · TodayCore · Xray · mihomo",
		License:   "AGPL-3.0-or-later",
	}
}

// NewApp creates the application with its store and engine already in place.
//
// They are built here rather than in startup because the frontend can call
// bound methods before startup has run — in dev mode a rebuilt app gets calls
// from the already-open page immediately — and every method relies on them.
func NewApp() *App {
	a := &App{startedAt: time.Now()}
	sub.DefaultUserAgent = "TomorrowClient/" + Version
	st, err := store.New()
	if err != nil {
		a.storeErr = err
		st = store.NewMemory()
	}
	a.store = st
	// Downloaded rule-sets and the core's other caches live with the data.
	singbox.CacheDir = st.Dir()
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		panic("secure controller token: " + err.Error())
	}
	singbox.APISecret = hex.EncodeToString(token)

	// The engine pushes status snapshots to the frontend over the
	// "vpn:status" event, and each core log line over "vpn:log". The status
	// callback also relabels the tray connect/disconnect item. Events need the
	// Wails context, which only exists once startup has run.
	a.engine = vpn.New(
		func(s model.Status) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "vpn:status", s)
			}
			a.updateTrayStatus(s)
		},
		func(line string) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "vpn:log", line)
			}
		},
	)
	return a
}

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.storeErr != nil {
		runtime.LogError(ctx, "store init (settings will not be saved): "+a.storeErr.Error())
	}

	a.setupTray()
	go a.updateLoop(ctx)

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

// ImportLinks imports every server in the pasted text: one share link,
// several one per line, a base64 subscription body, or a whole sing-box /
// Clash config. Entries that fail to parse are skipped; it is an error only
// when nothing could be imported.
func (a *App) ImportLinks(text string) ([]model.Profile, error) {
	profiles, err := sub.Parse(text)
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		if err := a.store.UpsertProfile(p); err != nil {
			return nil, err
		}
	}
	return profiles, nil
}

// SaveProfile inserts or updates a profile edited in the form. A profile that
// came from a link gets its link rebuilt from the edited fields, so copying it
// hands out what the app actually runs.
func (a *App) SaveProfile(p model.Profile) error {
	if !strings.HasPrefix(strings.TrimSpace(p.Raw), "{") {
		if l := link.Encode(p); l != "" {
			p.Raw = l
		}
	}
	return a.store.UpsertProfile(p)
}

// SaveProfileJSON replaces a JSON-imported profile with an edited version of
// its entry, keeping its identity and subscription.
func (a *App) SaveProfileJSON(id, text string) (model.Profile, error) {
	old, ok := a.store.Profile(id)
	if !ok {
		return model.Profile{}, fmt.Errorf("профиль не найден")
	}
	p, err := sub.ParseEntry(text)
	if err != nil {
		return model.Profile{}, err
	}
	p.ID, p.SubID = old.ID, old.SubID
	if err := a.store.UpsertProfile(p); err != nil {
		return model.Profile{}, err
	}
	return p, nil
}

// DeleteProfile removes a profile by id.
func (a *App) DeleteProfile(id string) error {
	return a.store.DeleteProfile(id)
}

// PingResult is one server's latency cell.
type PingResult struct {
	// LatencyMs is the measured delay by the configured method, -1 on failure.
	LatencyMs int `json:"latencyMs"`
	// OK reports whether the check succeeded.
	OK bool `json:"ok"`
}

// GetCores lists the linked cores with their versions.
func (a *App) GetCores() []cores.Info {
	return cores.List()
}

// CoreSupport says whether one core can run a profile, and why not.
type CoreSupport struct {
	Core      model.Core `json:"core"`
	Supported bool       `json:"supported"`
	Reason    string     `json:"reason,omitempty"`
}

// ProfileCores reports, for a profile, the core the current setting picks and
// how every core stands with it.
type ProfileCores struct {
	Selected model.Core    `json:"selected"`
	Error    string        `json:"error,omitempty"`
	Cores    []CoreSupport `json:"cores"`
}

// GetProfileCores tells the UI which core would run a profile and which of the
// others could.
func (a *App) GetProfileCores(id string) ProfileCores {
	p, ok := a.store.Profile(id)
	if !ok {
		return ProfileCores{Error: "профиль не найден"}
	}
	var out ProfileCores
	c, err := cores.Resolve(a.store.Settings().Core, p)
	out.Selected = c
	if err != nil {
		out.Error = err.Error()
	}
	for _, core := range cores.All {
		cs := CoreSupport{Core: core, Supported: true}
		if err := cores.Supports(core, p); err != nil {
			cs.Supported, cs.Reason = false, err.Error()
		}
		out.Cores = append(out.Cores, cs)
	}
	return out
}

// GetCoreIssues maps each profile the configured core cannot run to the
// reason. Empty under auto unless no core at all fits a profile.
func (a *App) GetCoreIssues() map[string]string {
	setting := a.store.Settings().Core
	out := map[string]string{}
	for _, p := range a.store.Profiles() {
		if _, err := cores.Resolve(setting, p); err != nil {
			out[p.ID] = err.Error()
		}
	}
	return out
}

// PingLatency is the live figure on the main screen, refreshed on a timer:
// the latency alone, -1 when the check fails.
func (a *App) PingLatency(id string) int {
	return a.PingProfile(id).LatencyMs
}

// PingProfile checks a profile the way the settings say:
//
//   - icmp: echo to the server — the bare network round trip;
//   - tcp: a handshake with the server's port — proves something listens;
//   - get / head: a real request through the proxy on the core that would
//     carry it — slower, but the only check that proves the transport, TLS and
//     credentials all work.
//
// ICMP and TCP leave through the physical adapter, so a running tunnel neither
// answers for the server nor carries the check.
func (a *App) PingProfile(id string) PingResult {
	p, ok := a.store.Profile(id)
	if !ok || p.Address == "" {
		return PingResult{LatencyMs: -1}
	}
	s := a.store.Settings()
	timeout := time.Duration(s.PingTimeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	var (
		d   time.Duration
		err error
	)
	switch s.PingMethod {
	case model.PingICMP:
		d, err = vpn.PingICMP(p.Address)
	case model.PingTCP:
		if p.Port == 0 {
			return PingResult{LatencyMs: -1}
		}
		d, err = vpn.PingTCP(p.Address, p.Port, timeout)
	default:
		core, rerr := cores.Resolve(s.Core, p)
		if rerr != nil {
			return PingResult{LatencyMs: -1}
		}
		method := "GET"
		if s.PingMethod == model.PingHEAD {
			method = "HEAD"
		}
		d, err = vpn.ProbeLatency(p, core, vpn.ProbeOptions{Method: method, URL: s.PingURL, Timeout: timeout})
	}
	if err != nil {
		return PingResult{LatencyMs: -1}
	}
	ms := int(d.Milliseconds())
	if ms < 1 {
		ms = 1
	}
	return PingResult{LatencyMs: ms, OK: true}
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

	profiles, info, err := sub.Fetch(url, s.ID, a.SubscriptionHeaders())
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
	profiles, info, err := sub.Fetch(s.URL, s.ID, a.SubscriptionHeaders())
	if err != nil {
		return model.Subscription{}, err
	}
	// Preserve identity across refresh for equivalent servers (including active selection).
	previous := a.store.Profiles()
	used := map[string]bool{}
	for i := range profiles {
		for _, old := range previous {
			if old.SubID == id && !used[old.ID] && old.Protocol == profiles[i].Protocol && old.Address == profiles[i].Address && old.Port == profiles[i].Port && old.Name == profiles[i].Name {
				profiles[i].ID = old.ID
				used[old.ID] = true
				break
			}
		}
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

// --- Device (HWID) ---

// GetDeviceInfo returns what the machine really is — the values used wherever
// the settings leave a field empty.
func (a *App) GetDeviceInfo() device.Info {
	return device.Detect(sub.DefaultUserAgent)
}

// SubscriptionHeaders is what goes out with every subscription request under
// the current settings. Exposed so the settings page can show it.
func (a *App) SubscriptionHeaders() map[string]string {
	s := a.store.Settings()
	real := device.Detect(sub.DefaultUserAgent)
	pick := func(set, auto string) string {
		if strings.TrimSpace(set) != "" {
			return strings.TrimSpace(set)
		}
		return auto
	}
	h := map[string]string{"User-Agent": pick(s.UserAgent, real.UserAgent)}
	if s.ClientPreset == "incy" {
		h["x-client"] = "INCY"
		h["x-app-version"] = s.ClientVersion
		h["Accept"] = "*/*"
		h["Accept-Language"] = "ru-RU"
		h["x-device-locale"] = "ru-RU"
		if s.HWIDEnabled && s.HWID == "" {
			real.HWID = device.IncyHWID()
		}
	}
	if s.HWIDEnabled {
		h["x-hwid"] = pick(s.HWID, real.HWID)
		h["x-device-os"] = pick(s.DeviceOS, real.OS)
		h["x-ver-os"] = pick(s.OSVersion, real.OSVersion)
		h["x-device-model"] = pick(s.DeviceModel, real.Model)
	}
	return h
}

// --- Settings ---

// GetSettings returns the persisted settings.
func (a *App) GetSettings() model.AppSettings {
	return a.store.Settings()
}

// SaveSettings persists settings and applies the Windows autostart task to
// match the LaunchAtStartup flag.
func (a *App) SaveSettings(s model.AppSettings) error {
	if s.LaunchAtStartup != a.store.Settings().LaunchAtStartup {
		if err := startup.Set(s.LaunchAtStartup); err != nil {
			return fmt.Errorf("autostart: %w", err)
		}
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

// PreviewConfig renders the config the active profile would run with under the
// current settings, for the developer tools. For Xray and mihomo that is the
// core's config followed by the sing-box front's.
func (a *App) PreviewConfig() (string, error) {
	s := a.store.Settings()
	if s.ActiveProfileID == "" {
		return "", fmt.Errorf("профиль не выбран")
	}
	p, ok := a.store.Profile(s.ActiveProfileID)
	if !ok {
		return "", fmt.Errorf("профиль не найден")
	}
	c, err := cores.Resolve(s.Core, p)
	if err != nil {
		return "", err
	}
	return vpn.Preview(c, p, s)
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
