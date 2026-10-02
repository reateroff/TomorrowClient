//go:build windows

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"TomorrowClient/internal/cores"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
	"TomorrowClient/internal/vpn"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const ClientRepository = "https://github.com/reateroff/TomorrowClient"

type toolState struct {
	mu                                  sync.Mutex
	update                              UpdateInfo
	checking, downloading, speedRunning bool
	epoch                               int64
	seen                                map[string][2]int64
	apps                                map[string]*ApplicationTraffic
}

type RuntimeStats struct {
	GoVersion     string       `json:"goVersion"`
	Goroutines    int          `json:"goroutines"`
	CPUs          int          `json:"cpus"`
	HeapBytes     uint64       `json:"heapBytes"`
	HeapObjects   uint64       `json:"heapObjects"`
	SystemBytes   uint64       `json:"systemBytes"`
	GCCount       uint32       `json:"gcCount"`
	GCPauseMs     float64      `json:"gcPauseMs"`
	UptimeSeconds int64        `json:"uptimeSeconds"`
	PID           int          `json:"pid"`
	State         model.Status `json:"status"`
}

func (a *App) GetRuntimeStats() RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return RuntimeStats{runtime.Version(), runtime.NumGoroutine(), runtime.NumCPU(), m.HeapAlloc, m.HeapObjects, m.Sys, m.NumGC, float64(m.PauseTotalNs) / 1e6, int64(time.Since(a.startedAt).Seconds()), os.Getpid(), a.engine.Status()}
}
func (a *App) GetGoroutineDump() (string, error) {
	if !a.store.Settings().DevMode {
		return "", fmt.Errorf("включите DEV режим")
	}
	var b bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&b, 1); err != nil {
		return "", err
	}
	if b.Len() > 2<<20 {
		return string(b.Bytes()[:2<<20]) + "\n[truncated]", nil
	}
	return b.String(), nil
}
func (a *App) CollectGarbage() error {
	if !a.store.Settings().DevMode {
		return fmt.Errorf("включите DEV режим")
	}
	runtime.GC()
	return nil
}
func (a *App) ExportHeapProfile() (string, error) {
	if !a.store.Settings().DevMode {
		return "", fmt.Errorf("включите DEV режим")
	}
	p, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Heap profile", DefaultFilename: "TomorrowClient-heap.pprof"})
	if err != nil || p == "" {
		return "", err
	}
	f, err := os.Create(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err = pprof.WriteHeapProfile(f); err != nil {
		return "", err
	}
	return p, nil
}

type ConnectionMetadata struct {
	Network         string `json:"network"`
	Host            string `json:"host"`
	DestinationIP   string `json:"destinationIP"`
	DestinationPort string `json:"destinationPort"`
	Process         string `json:"process"`
	ProcessPath     string `json:"processPath"`
}

type LiveConnection struct {
	ID       string             `json:"id"`
	Metadata ConnectionMetadata `json:"metadata"`
	Upload   int64              `json:"upload"`
	Download int64              `json:"download"`
	Start    string             `json:"start"`
	Chains   []string           `json:"chains"`
	Rule     string             `json:"rule"`
}
type ApplicationTraffic struct {
	Process     string `json:"process"`
	Upload      int64  `json:"upload"`
	Download    int64  `json:"download"`
	Connections int    `json:"connections"`
}
type ConnectionSnapshot struct {
	Connections   []LiveConnection     `json:"connections"`
	Applications  []ApplicationTraffic `json:"applications"`
	UploadTotal   int64                `json:"uploadTotal"`
	DownloadTotal int64                `json:"downloadTotal"`
	Memory        int64                `json:"memory"`
}

func controllerRequest(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://"+singbox.ClashAPIAddr+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+singbox.APISecret)
	return (&http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}).Do(req)
}
func (a *App) GetConnections() (ConnectionSnapshot, error) {
	out := ConnectionSnapshot{Connections: []LiveConnection{}, Applications: []ApplicationTraffic{}}
	st := a.engine.Status()
	if st.State != model.StateConnected {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := controllerRequest(ctx, "GET", "/connections")
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return out, fmt.Errorf("controller HTTP %d", resp.StatusCode)
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&out); err != nil {
		return out, err
	}
	a.tools.mu.Lock()
	defer a.tools.mu.Unlock()
	if a.tools.epoch != st.ConnectedAt {
		a.tools.epoch = st.ConnectedAt
		a.tools.seen = map[string][2]int64{}
		a.tools.apps = map[string]*ApplicationTraffic{}
	}
	for _, p := range a.tools.apps {
		p.Connections = 0
	}
	live := map[string]bool{}
	for _, c := range out.Connections {
		live[c.ID] = true
		name := c.Metadata.Process
		if name == "" {
			name = filepath.Base(c.Metadata.ProcessPath)
		}
		if name == "" || name == "." {
			name = "Не определено"
		}
		if _, ok := a.tools.apps[name]; !ok {
			a.tools.apps[name] = &ApplicationTraffic{Process: name}
		}
		p := a.tools.apps[name]
		last := a.tools.seen[c.ID]
		if c.Upload > last[0] {
			p.Upload += c.Upload - last[0]
		}
		if c.Download > last[1] {
			p.Download += c.Download - last[1]
		}
		p.Connections++
		a.tools.seen[c.ID] = [2]int64{c.Upload, c.Download}
	}
	for id := range a.tools.seen {
		if !live[id] {
			delete(a.tools.seen, id)
		}
	}
	for _, p := range a.tools.apps {
		out.Applications = append(out.Applications, *p)
	}
	sort.Slice(out.Applications, func(i, j int) bool {
		return out.Applications[i].Download+out.Applications[i].Upload > out.Applications[j].Download+out.Applications[j].Upload
	})
	if out.Connections == nil {
		out.Connections = []LiveConnection{}
	}
	return out, nil
}
func (a *App) CloseConnection(id string) error {
	if id == "" {
		return fmt.Errorf("connection ID is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := controllerRequest(ctx, "DELETE", "/connections/"+url.PathEscape(id))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("controller HTTP %d", resp.StatusCode)
	}
	return nil
}
func (a *App) TestProfileSpeed(id string) (vpn.SpeedResult, error) {
	a.tools.mu.Lock()
	if a.tools.speedRunning {
		a.tools.mu.Unlock()
		return vpn.SpeedResult{}, fmt.Errorf("тест скорости уже выполняется")
	}
	a.tools.speedRunning = true
	a.tools.mu.Unlock()
	defer func() { a.tools.mu.Lock(); a.tools.speedRunning = false; a.tools.mu.Unlock() }()
	p, ok := a.store.Profile(id)
	if !ok {
		return vpn.SpeedResult{}, fmt.Errorf("профиль не найден")
	}
	c, err := cores.Resolve(a.store.Settings().Core, p)
	if err != nil {
		return vpn.SpeedResult{}, err
	}
	var revision int64
	if p.SubID != "" {
		if sub, found := a.store.Subscription(p.SubID); found {
			revision = sub.UpdatedAt
		}
	}
	result, err := vpn.DownloadSpeed(p, c)
	if err != nil {
		return result, err
	}
	_, err = a.store.SaveProfileSpeed(p, revision, model.ProfileSpeed{DownloadMbps: result.DownloadMbps, Bytes: result.Bytes, DurationMs: result.DurationMs, Core: string(result.Core), MeasuredAt: time.Now().UnixMilli()})
	return result, err
}

type routingDocument struct {
	Format      string              `json:"format"`
	Version     int                 `json:"version"`
	Mode        string              `json:"mode"`
	Rules       []model.RoutingRule `json:"rules"`
	Graph       model.RouteGraph    `json:"graph"`
	SimpleFinal string              `json:"simpleFinal,omitempty"`
}

func (a *App) ExportRouting() (string, error) {
	s := a.store.Settings()
	doc := routingDocument{Format: "TomorrowClient.routing", Version: 1, Mode: s.RoutingMode, Rules: s.Rules, Graph: s.Graph, SimpleFinal: s.SimpleFinal}
	// Icons are UI cache, not routing configuration. Keep exports small/portable.
	doc.Rules = append([]model.RoutingRule{}, s.Rules...)
	for i := range doc.Rules {
		doc.Rules[i].Icon = ""
	}
	doc.Graph.Nodes = append([]model.RouteNode{}, s.Graph.Nodes...)
	for i := range doc.Graph.Nodes {
		doc.Graph.Nodes[i].Icons = nil
	}
	p, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Экспорт маршрутизации", DefaultFilename: "TomorrowClient-routing.json", Filters: []wruntime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}}})
	if err != nil || p == "" {
		return "", err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return p, os.WriteFile(p, b, 0600)
}
func (a *App) ImportRouting() (*model.AppSettings, error) {
	if a.engine.Status().State != model.StateDisconnected && a.engine.Status().State != model.StateError {
		return nil, fmt.Errorf("сначала отключите туннель")
	}
	p, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Импорт маршрутизации", Filters: []wruntime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}}})
	if err != nil || p == "" {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 1<<20 {
		return nil, fmt.Errorf("файл больше 1 МБ")
	}
	doc, err := parseRoutingDocument(b)
	if err != nil {
		return nil, err
	}
	s := a.store.Settings()
	s.RoutingMode = doc.Mode
	s.Rules = doc.Rules
	s.Graph = doc.Graph
	s.SimpleFinal = doc.SimpleFinal
	if err = a.store.SaveSettings(s); err != nil {
		return nil, err
	}
	return &s, nil
}
func parseRoutingDocument(b []byte) (routingDocument, error) {
	var d routingDocument
	if err := json.Unmarshal(b, &d); err != nil {
		return d, err
	}
	if d.Format != "TomorrowClient.routing" || d.Version != 1 {
		return d, fmt.Errorf("неподдерживаемый формат маршрутизации")
	}
	if d.Mode != "simple" && d.Mode != "pro" {
		return d, fmt.Errorf("неизвестный режим маршрутизации")
	}
	validAction := func(s string) bool { return s == "" || s == "proxy" || s == "direct" || s == "block" }
	if !validAction(d.SimpleFinal) {
		return d, fmt.Errorf("неизвестное действие Остальное Easy")
	}
	if !validAction(d.Graph.Final) {
		return d, fmt.Errorf("неизвестное действие Остальное")
	}
	types := map[string]bool{"domain": true, "domain_full": true, "domain_keyword": true, "domain_regex": true, "ip": true, "port": true, "process": true, "process_path": true, "network": true, "protocol": true, "geosite": true, "geoip": true}
	ids := map[string]bool{}
	if len(d.Graph.Nodes) > 1000 || len(d.Rules) > 4096 {
		return d, fmt.Errorf("слишком много правил")
	}
	for _, point := range d.Graph.Layout {
		if math.IsNaN(point.X) || math.IsNaN(point.Y) || math.IsInf(point.X, 0) || math.IsInf(point.Y, 0) {
			return d, fmt.Errorf("некорректные координаты")
		}
	}
	if d.Graph.FinalHidden && d.Graph.Final != "" {
		return d, fmt.Errorf("скрытое Остальное должно быть отключено")
	}
	if len(d.Graph.Notes) > 200 {
		return d, fmt.Errorf("слишком много заметок")
	}
	for _, n := range d.Graph.Notes {
		if n.ID == "" || n.ID == "final" || n.ID == "proxy" || n.ID == "direct" || n.ID == "block" || ids[n.ID] || len(n.Text) > 16000 || math.IsNaN(n.X) || math.IsNaN(n.Y) || math.IsInf(n.X, 0) || math.IsInf(n.Y, 0) {
			return d, fmt.Errorf("некорректная заметка")
		}
		ids[n.ID] = true
	}
	for _, n := range d.Graph.Nodes {
		if n.ID == "" || n.ID == "final" || n.ID == "proxy" || n.ID == "direct" || n.ID == "block" || ids[n.ID] || !types[n.Type] || !validAction(n.Action) {
			return d, fmt.Errorf("некорректная нода %q", n.ID)
		}
		if len(n.Values) > 4096 || math.IsNaN(n.X) || math.IsNaN(n.Y) || math.IsInf(n.X, 0) || math.IsInf(n.Y, 0) {
			return d, fmt.Errorf("некорректные координаты/размер ноды %q", n.ID)
		}
		for _, value := range n.Values {
			if err := validateRouteValue(n.Type, value); err != nil {
				return d, fmt.Errorf("нода %q: %w", n.ID, err)
			}
		}
		ids[n.ID] = true
	}
	for _, r := range d.Rules {
		if (r.Type != "domain" && r.Type != "ip" && r.Type != "process" && r.Type != "geosite" && r.Type != "geoip") || !validAction(r.Action) || strings.TrimSpace(r.Value) == "" {
			return d, fmt.Errorf("некорректное правило")
		}
		if err := validateRouteValue(r.Type, r.Value); err != nil {
			return d, err
		}
	}
	return d, nil
}

func validateRouteValue(kind, value string) error {
	v := strings.TrimSpace(value)
	if v == "" || len(v) > 2048 {
		return fmt.Errorf("пустое или слишком длинное значение")
	}
	switch kind {
	case "ip":
		if strings.Contains(v, "/") {
			if _, _, err := net.ParseCIDR(v); err != nil {
				return fmt.Errorf("некорректный CIDR %q", v)
			}
		} else if net.ParseIP(v) == nil {
			return fmt.Errorf("некорректный IP %q", v)
		}
	case "port":
		parts := regexp.MustCompile(`^([0-9]{1,5})(?:\s*[-:]\s*([0-9]{1,5}))?$`).FindStringSubmatch(v)
		if parts == nil {
			return fmt.Errorf("некорректный порт %q", v)
		}
		first, _ := strconv.Atoi(parts[1])
		last := first
		if parts[2] != "" {
			last, _ = strconv.Atoi(parts[2])
		}
		if first < 1 || last > 65535 || first > last {
			return fmt.Errorf("порт вне диапазона 1–65535")
		}
	case "domain_regex":
		if _, err := regexp.Compile(v); err != nil {
			return fmt.Errorf("некорректный regex: %w", err)
		}
	case "domain", "domain_full":
		if strings.ContainsAny(v, " /\\\t\r\n") || net.ParseIP(v) != nil {
			return fmt.Errorf("некорректный домен %q", v)
		}
	case "network":
		if v = strings.ToLower(v); v != "tcp" && v != "udp" && v != "icmp" {
			return fmt.Errorf("сеть должна быть tcp/udp/icmp")
		}
	case "process":
		if !strings.HasSuffix(strings.ToLower(v), ".exe") || strings.ContainsAny(v, "/\\") {
			return fmt.Errorf("ожидается имя процесса .exe")
		}
	case "process_path":
		if !filepath.IsAbs(v) || !strings.HasSuffix(strings.ToLower(v), ".exe") {
			return fmt.Errorf("ожидается абсолютный путь .exe")
		}
	case "geosite", "geoip":
		v = strings.TrimPrefix(v, kind+":")
		if !regexp.MustCompile(`^[a-zA-Z0-9!@._-]+$`).MatchString(v) {
			return fmt.Errorf("некорректное имя набора правил")
		}
	}
	return nil
}

// Updates come ONLY from this repository's published stable releases. Downloaded
// code is never executed silently, and never replaces the running/dev binary.
type UpdateInfo struct {
	Version      string `json:"version"`
	Available    bool   `json:"available"`
	URL          string `json:"url"`
	AssetURL     string `json:"assetUrl"`
	AssetName    string `json:"assetName"`
	Digest       string `json:"digest"`
	DownloadPath string `json:"downloadPath"`
	Downloading  bool   `json:"downloading"`
	Progress     int    `json:"progress"`
	Error        string `json:"error"`
	CheckedAt    int64  `json:"checkedAt"`
}

func newerVersion(remote, local string) bool {
	parse := func(s string) []int {
		p := strings.Split(strings.TrimPrefix(s, "v"), ".")
		out := make([]int, 3)
		if len(p) != 3 {
			return nil
		}
		for i := range out {
			n, e := strconv.Atoi(p[i])
			if e != nil || n < 0 {
				return nil
			}
			out[i] = n
		}
		return out
	}
	r, l := parse(remote), parse(local)
	if r == nil || l == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if r[i] != l[i] {
			return r[i] > l[i]
		}
	}
	return false
}
func (a *App) GetUpdateInfo() UpdateInfo {
	a.tools.mu.Lock()
	defer a.tools.mu.Unlock()
	return a.tools.update
}
func (a *App) CheckUpdates() (UpdateInfo, error) {
	a.tools.mu.Lock()
	if a.tools.checking {
		u := a.tools.update
		a.tools.mu.Unlock()
		return u, nil
	}
	a.tools.checking = true
	a.tools.mu.Unlock()
	defer func() { a.tools.mu.Lock(); a.tools.checking = false; a.tools.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/reateroff/TomorrowClient/releases/latest", nil)
	req.Header.Set("User-Agent", "TomorrowClient/"+Version)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return a.updateError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		a.tools.mu.Lock()
		a.tools.update = UpdateInfo{CheckedAt: time.Now().UnixMilli()}
		u := a.tools.update
		a.tools.mu.Unlock()
		a.emitUpdate()
		return u, nil
	}
	if resp.StatusCode != 200 {
		return a.updateError(fmt.Errorf("GitHub HTTP %d", resp.StatusCode))
	}
	var release struct {
		Tag        string `json:"tag_name"`
		URL        string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return a.updateError(err)
	}
	u := UpdateInfo{Version: release.Tag, URL: release.URL, Available: !release.Draft && !release.Prerelease && newerVersion(release.Tag, Version), CheckedAt: time.Now().UnixMilli()}
	best := 0
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		score := 0
		if (strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".zip")) && !strings.Contains(name, "arm64") && !strings.Contains(name, "linux") && !strings.Contains(name, "darwin") && (strings.Contains(name, "windows") || strings.Contains(name, "amd64") || strings.HasSuffix(name, ".exe")) {
			score = 1
			if strings.Contains(name, "installer") {
				score = 2
			}
		}
		if score > best {
			best = score
			u.AssetName = asset.Name
			u.AssetURL = asset.URL
			u.Digest = asset.Digest
		}
	}
	a.tools.mu.Lock()
	old := a.tools.update
	if old.Version == u.Version {
		u.DownloadPath = old.DownloadPath
		u.Progress = old.Progress
		u.Downloading = old.Downloading
	}
	a.tools.update = u
	a.tools.mu.Unlock()
	a.emitUpdate()
	return u, nil
}
func (a *App) updateError(err error) (UpdateInfo, error) {
	a.tools.mu.Lock()
	a.tools.update.Error = err.Error()
	a.tools.update.CheckedAt = time.Now().UnixMilli()
	u := a.tools.update
	a.tools.mu.Unlock()
	a.emitUpdate()
	return u, err
}
func (a *App) emitUpdate() {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "app:update", a.GetUpdateInfo())
	}
}
func githubAssetURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Host == "github.com" && strings.HasPrefix(u.Path, "/reateroff/TomorrowClient/releases/download/")
}
func (a *App) DownloadUpdate() (string, error) {
	a.tools.mu.Lock()
	u := a.tools.update
	if a.tools.downloading {
		a.tools.mu.Unlock()
		return "", fmt.Errorf("загрузка уже выполняется")
	}
	if !u.Available || !githubAssetURL(u.AssetURL) || u.AssetName != filepath.Base(u.AssetName) {
		a.tools.mu.Unlock()
		return "", fmt.Errorf("подходящий Windows asset не найден")
	}
	a.tools.downloading = true
	a.tools.update.Downloading = true
	a.tools.update.Error = ""
	a.tools.mu.Unlock()
	a.emitUpdate()
	defer func() {
		a.tools.mu.Lock()
		a.tools.downloading = false
		a.tools.update.Downloading = false
		a.tools.mu.Unlock()
		a.emitUpdate()
	}()
	dir := filepath.Join(a.store.Dir(), "updates")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, u.AssetName)
	tmp := path + ".partial"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", u.AssetURL, nil)
	client := &http.Client{Timeout: 15 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		h := r.URL.Hostname()
		if len(via) > 5 || r.URL.Scheme != "https" || !(h == "github.com" || strings.HasSuffix(h, ".githubusercontent.com")) {
			return fmt.Errorf("небезопасный redirect обновления")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		a.updateError(err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("download HTTP %d", resp.StatusCode)
		a.updateError(err)
		return "", err
	}
	const limit int64 = 512 << 20
	if resp.ContentLength > limit {
		return "", fmt.Errorf("обновление больше 512 МБ")
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp)
	hash := sha256.New()
	reader := io.LimitReader(resp.Body, limit+1)
	buf := make([]byte, 128<<10)
	var total int64
	last := time.Now()
	for {
		n, e := reader.Read(buf)
		if n > 0 {
			if _, err = f.Write(buf[:n]); err != nil {
				break
			}
			hash.Write(buf[:n])
			total += int64(n)
		}
		if time.Since(last) > 500*time.Millisecond {
			a.tools.mu.Lock()
			if resp.ContentLength > 0 {
				a.tools.update.Progress = int(total * 100 / resp.ContentLength)
			}
			a.tools.mu.Unlock()
			a.emitUpdate()
			last = time.Now()
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if total > limit {
		err = fmt.Errorf("обновление превышает лимит")
	}
	if resp.ContentLength > 0 && total != resp.ContentLength && err == nil {
		err = io.ErrUnexpectedEOF
	}
	if u.Digest != "" {
		if u.Digest != "sha256:"+hex.EncodeToString(hash.Sum(nil)) {
			err = fmt.Errorf("SHA256 обновления не совпадает")
		}
	}
	if err != nil {
		a.updateError(err)
		return "", err
	}
	if err = os.Rename(tmp, path); err != nil {
		return "", err
	}
	a.tools.mu.Lock()
	a.tools.update.DownloadPath = path
	a.tools.update.Progress = 100
	a.tools.mu.Unlock()
	return path, nil
}
func (a *App) ShowUpdateFolder() error {
	u := a.GetUpdateInfo()
	if u.DownloadPath == "" {
		return fmt.Errorf("обновление ещё не скачано")
	}
	return exec.Command("explorer.exe", "/select,", u.DownloadPath).Start()
}
func (a *App) updateLoop(ctx context.Context) {
	check := func() {
		u, err := a.CheckUpdates()
		if err == nil && u.Available && u.AssetURL != "" && u.DownloadPath == "" && a.store.Settings().AutoUpdate {
			_, _ = a.DownloadUpdate()
		}
	}
	check()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
