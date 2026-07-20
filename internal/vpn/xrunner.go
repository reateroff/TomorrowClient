//go:build windows

package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"TomorrowClient/internal/model"
	"TomorrowClient/internal/xray"
)

// xRunner runs xray.exe plus tun2socks.exe. xray exposes a local SOCKS inbound;
// tun2socks creates the WinTun adapter and forwards it into that SOCKS port.
type xRunner struct {
	xrayCmd    *exec.Cmd
	t2sCmd     *exec.Cmd
	configPath string
	serverIP   string
}

func (r *xRunner) start(p model.Profile, s model.AppSettings) error {
	cfg, err := xray.Build(p, s)
	if err != nil {
		return fmt.Errorf("build xray config: %w", err)
	}
	r.configPath = filepath.Join(os.TempDir(), "tomorrow-xray.json")
	if err := os.WriteFile(r.configPath, cfg, 0o644); err != nil {
		return fmt.Errorf("write xray config: %w", err)
	}

	xrayBin := binPath("xray.exe")
	if _, err := os.Stat(xrayBin); err != nil {
		return fmt.Errorf("xray.exe not found next to the app: %w", err)
	}
	t2sBin := binPath("tun2socks.exe")
	if _, err := os.Stat(t2sBin); err != nil {
		return fmt.Errorf("tun2socks.exe not found next to the app: %w", err)
	}

	// 1. Start xray.
	r.xrayCmd = exec.Command(xrayBin, "run", "-c", r.configPath)
	r.xrayCmd.Dir = binDir()
	hidden(r.xrayCmd)
	if err := r.xrayCmd.Start(); err != nil {
		return fmt.Errorf("start xray: %w", err)
	}

	// 2. Start tun2socks pointed at the xray SOCKS inbound.
	proxy := fmt.Sprintf("socks5://127.0.0.1:%d", xray.SocksPort)
	r.t2sCmd = exec.Command(t2sBin,
		"-device", "tun://"+tunAdapterName,
		"-proxy", proxy,
		"-loglevel", "warning",
	)
	r.t2sCmd.Dir = binDir()
	hidden(r.t2sCmd)
	if err := r.t2sCmd.Start(); err != nil {
		r.stop()
		return fmt.Errorf("start tun2socks: %w", err)
	}

	// 3. Configure the adapter address, routes and DNS.
	r.serverIP = resolveServerIP(p.Address)
	if err := configureTun(); err != nil {
		r.stop()
		return fmt.Errorf("configure tun: %w", err)
	}
	if err := addRoutes(r.serverIP); err != nil {
		r.stop()
		return fmt.Errorf("add routes: %w", err)
	}
	_ = setDNSOnTun(s.DNS)
	return nil
}

func (r *xRunner) stop() {
	delRoutes(r.serverIP)
	if r.t2sCmd != nil && r.t2sCmd.Process != nil {
		_ = r.t2sCmd.Process.Kill()
		_, _ = r.t2sCmd.Process.Wait()
	}
	if r.xrayCmd != nil && r.xrayCmd.Process != nil {
		_ = r.xrayCmd.Process.Kill()
		_, _ = r.xrayCmd.Process.Wait()
	}
	if r.configPath != "" {
		_ = os.Remove(r.configPath)
	}
	r.xrayCmd, r.t2sCmd = nil, nil
}
