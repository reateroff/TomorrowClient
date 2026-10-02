//go:build windows

// Developer tools. Everything here is reachable only once the user unlocks the
// hidden developer section (ten taps on the client name in About), so these
// methods trade polish for raw insight into the runtime environment.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/startup"
	"TomorrowClient/internal/vpn"
)

// coreBinaries are the runtime files expected next to the executable. Every
// core is linked into the app, so wintun.dll is all that is left: sing-box
// loads it to create the TUN adapter. The second field is the argument that makes a file
// print its version ("" = not an executable).
var coreBinaries = []struct {
	name       string
	versionArg string
}{
	{"wintun.dll", ""},
}

// BinStatus reports whether a bundled dependency is present next to the app.
type BinStatus struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	SizeKB  int64  `json:"sizeKB"`
	Version string `json:"version"`
}

// Diagnostics is a snapshot of the runtime environment, gathered for the
// exported diagnostics report.
type Diagnostics struct {
	Version    string      `json:"version"`
	GoVersion  string      `json:"goVersion"`
	Arch       string      `json:"arch"`
	NumCPU     int         `json:"numCPU"`
	Goroutines int         `json:"goroutines"`
	HeapMB     float64     `json:"heapMB"`
	UptimeSec  int64       `json:"uptimeSec"`
	Elevated   bool        `json:"elevated"`
	PID        int         `json:"pid"`
	ExePath    string      `json:"exePath"`
	BinDir     string      `json:"binDir"`
	ConfigDir  string      `json:"configDir"`
	Binaries   []BinStatus `json:"binaries"`
}

// diagnostics collects environment, process and dependency information. It is
// unexported because the UI no longer shows a diagnostics panel — the data now
// only feeds ExportDiagnostics.
func (a *App) diagnostics() Diagnostics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	exe, _ := os.Executable()
	binDir := vpn.BinDir()

	d := Diagnostics{
		Version:    Version,
		GoVersion:  runtime.Version(),
		Arch:       runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
		HeapMB:     float64(ms.Alloc) / (1024 * 1024),
		Elevated:   windows.GetCurrentProcessToken().IsElevated(),
		PID:        os.Getpid(),
		ExePath:    exe,
		BinDir:     binDir,
		ConfigDir:  a.configDir(),
	}
	if !a.startedAt.IsZero() {
		d.UptimeSec = int64(time.Since(a.startedAt).Seconds())
	}

	for _, b := range coreBinaries {
		path := filepath.Join(binDir, b.name)
		st := BinStatus{Name: b.name}
		if fi, err := os.Stat(path); err == nil {
			st.Present = true
			st.SizeKB = fi.Size() / 1024
			if b.versionArg != "" {
				st.Version = firstLine(runHidden(3*time.Second, path, b.versionArg))
			}
		}
		d.Binaries = append(d.Binaries, st)
	}
	return d
}

// configDir returns the store directory, or "" when the store failed to open.
func (a *App) configDir() string {
	if a.store == nil {
		return ""
	}
	return a.store.Dir()
}

// GetSettingsJSON returns the persisted settings exactly as they are stored, so
// the raw shape (including fields the UI does not surface) can be inspected.
func (a *App) GetSettingsJSON() (string, error) {
	b, err := json.MarshalIndent(a.store.Settings(), "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RunNetDiag returns a compact network picture: the interface table plus the
// IPv4 routing table, which is where TUN problems normally show up.
func (a *App) RunNetDiag() (string, error) {
	var sb strings.Builder

	sb.WriteString("=== netsh interface ipv4 show interfaces ===\n")
	sb.WriteString(runHidden(8*time.Second, "netsh", "interface", "ipv4", "show", "interfaces"))
	sb.WriteString("\n=== route print -4 ===\n")
	sb.WriteString(runHidden(8*time.Second, "route", "print", "-4"))

	out := strings.TrimSpace(sb.String())
	if out == "" {
		return "", fmt.Errorf("не удалось получить сетевую информацию")
	}
	return out, nil
}

// ResetSettings restores the default configuration. Profiles and subscriptions
// are left untouched; DevMode is kept so the caller does not lock itself out.
func (a *App) ResetSettings() error {
	if err := startup.Set(false); err != nil {
		return err
	}
	def := model.DefaultSettings()
	def.DevMode = a.store.Settings().DevMode
	return a.store.SaveSettings(def)
}

// ResetAllData wipes every profile, subscription and setting — a factory reset.
// The tunnel is torn down first so nothing keeps running against deleted state.
// Unlike ResetSettings this also clears DevMode, hiding the developer section
// again, which is the honest outcome of a clean install.
func (a *App) ResetAllData() error {
	a.stopSimulation()
	if a.engine != nil {
		a.engine.Disconnect()
	}
	if err := startup.Set(false); err != nil {
		return err
	}
	if err := a.store.ResetAll(); err != nil {
		return err
	}
	// The frontend caches profiles and subscriptions, so tell it to reload
	// everything rather than leaving deleted servers on screen.
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "app:datareset")
	}
	return nil
}

// GetActiveProfileJSON returns the currently selected profile exactly as it is
// stored, so every parsed field of a share link can be inspected.
func (a *App) GetActiveProfileJSON() (string, error) {
	s := a.store.Settings()
	if s.ActiveProfileID == "" {
		return "", fmt.Errorf("профиль не выбран")
	}
	p, ok := a.store.Profile(s.ActiveProfileID)
	if !ok {
		return "", fmt.Errorf("профиль не найден")
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

/* -------------------------------- Simulation --------------------------------- */

// simScenarios maps a scenario id to the status it pushes to the UI. Only the
// states the engine can genuinely report are offered — the model has no
// "disconnecting" state, so simulating one would test a screen that cannot
// occur in practice.
var simScenarios = map[string]struct {
	state model.ConnState
	err   string
}{
	"connecting":   {model.StateConnecting, ""},
	"connected":    {model.StateConnected, ""},
	"disconnected": {model.StateDisconnected, ""},
	"error":        {model.StateError, "не удалось установить соединение с сервером"},
	"no-internet":  {model.StateError, "нет подключения к сети"},
	"no-wintun":    {model.StateError, "wintun.dll не найден рядом с приложением"},
	"no-admin":     {model.StateError, "нужны права администратора для создания TUN-адаптера"},
	"timeout":      {model.StateError, "таймаут подключения к серверу"},
	"handshake":    {model.StateError, "сервер разорвал соединение при рукопожатии"},
}

// SimulateStatus pushes a fake connection status to the UI so every state can
// be inspected without a working server. It only changes what the app displays:
// no tunnel is created and the engine is untouched. A real status event from
// the engine overrides the simulation, and StopSimulation restores the truth.
func (a *App) SimulateStatus(scenario string) error {
	sc, ok := simScenarios[scenario]
	if !ok {
		return fmt.Errorf("неизвестный сценарий: %s", scenario)
	}
	a.stopSimulation()

	s := a.store.Settings()
	st := model.Status{State: sc.state, Core: s.Core, Error: sc.err}
	if p, found := a.store.Profile(s.ActiveProfileID); found {
		st.ActiveProfile = &p
		st.Core, _ = cores.Resolve(s.Core, p)
	}
	if st.Core == model.CoreAuto {
		st.Core = model.CoreSingBox
	}

	if sc.state == model.StateConnected {
		st.ConnectedAt = time.Now().UnixMilli()
		a.pushStatus(st)
		a.animateTraffic(st)
		return nil
	}
	a.pushStatus(st)
	return nil
}

// StopSimulation cancels any running simulation and re-publishes the engine's
// real status.
func (a *App) StopSimulation() {
	a.stopSimulation()
	if a.engine != nil {
		a.pushStatus(a.engine.Status())
	}
}

// animateTraffic keeps a simulated "connected" status alive with plausible,
// moving counters so the speed and total readouts can be checked.
func (a *App) animateTraffic(base model.Status) {
	stop := make(chan struct{})
	a.simMu.Lock()
	a.simStop = stop
	a.simMu.Unlock()

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		st := base
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				up := 40_000 + rand.Uint64()%90_000
				down := 300_000 + rand.Uint64()%1_600_000
				st.Stats.UploadSpeed = up
				st.Stats.DownloadSpeed = down
				st.Stats.Upload += up
				st.Stats.Download += down
				a.pushStatus(st)
			}
		}
	}()
}

// stopSimulation halts the traffic animation if one is running.
func (a *App) stopSimulation() {
	a.simMu.Lock()
	defer a.simMu.Unlock()
	if a.simStop != nil {
		close(a.simStop)
		a.simStop = nil
	}
}

// pushStatus publishes a status snapshot exactly the way the engine does, so
// the window and the tray stay in agreement during a simulation.
func (a *App) pushStatus(s model.Status) {
	if a.ctx == nil {
		return
	}
	wruntime.EventsEmit(a.ctx, "vpn:status", s)
	a.updateTrayStatus(s)
}

// ExportDiagnostics writes a plain-text report (environment, settings and the
// buffered core log) next to the config files and returns its full path.
func (a *App) ExportDiagnostics() (string, error) {
	dir := a.configDir()
	if dir == "" {
		return "", fmt.Errorf("каталог конфигурации недоступен")
	}

	d := a.diagnostics()
	var sb strings.Builder
	fmt.Fprintf(&sb, "TomorrowClient diagnostics — %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&sb, "version    %s\ngo         %s\narch       %s\ncpus       %d\n",
		d.Version, d.GoVersion, d.Arch, d.NumCPU)
	fmt.Fprintf(&sb, "goroutines %d\nheap       %.1f MB\nuptime     %ds\n",
		d.Goroutines, d.HeapMB, d.UptimeSec)
	fmt.Fprintf(&sb, "elevated   %v\npid        %d\nexe        %s\nconfig     %s\n\n",
		d.Elevated, d.PID, d.ExePath, d.ConfigDir)

	sb.WriteString("--- binaries ---\n")
	for _, b := range d.Binaries {
		if b.Present {
			fmt.Fprintf(&sb, "%-16s ok    %6d KB  %s\n", b.Name, b.SizeKB, b.Version)
		} else {
			fmt.Fprintf(&sb, "%-16s MISSING\n", b.Name)
		}
	}

	sb.WriteString("\n--- settings ---\n")
	if s, err := a.GetSettingsJSON(); err == nil {
		sb.WriteString(s)
	}

	sb.WriteString("\n\n--- core log ---\n")
	if a.engine != nil {
		sb.WriteString(strings.Join(a.engine.Logs(), "\n"))
	}

	path := filepath.Join(dir, fmt.Sprintf("diagnostics-%s.txt", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// runHidden executes a command without flashing a console window and returns
// its combined output. Failures collapse to the output captured so far, since
// several of these tools write useful text to stderr and exit non-zero.
func runHidden(timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}

// firstLine trims a multi-line tool banner down to its first line.
func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
