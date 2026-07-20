//go:build windows

package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
)

// sbRunner runs sing-box.exe as a subprocess.
type sbRunner struct {
	cmd        *exec.Cmd
	configPath string
}

// start writes the config and launches sing-box run -c <config>.
func (r *sbRunner) start(p model.Profile, s model.AppSettings) error {
	cfg, err := singbox.Build(p, s)
	if err != nil {
		return fmt.Errorf("build sing-box config: %w", err)
	}
	r.configPath = filepath.Join(os.TempDir(), "tomorrow-singbox.json")
	if err := os.WriteFile(r.configPath, cfg, 0o644); err != nil {
		return fmt.Errorf("write sing-box config: %w", err)
	}

	bin := binPath("sing-box.exe")
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("sing-box.exe not found next to the app: %w", err)
	}
	r.cmd = exec.Command(bin, "run", "-c", r.configPath)
	r.cmd.Dir = binDir()
	hidden(r.cmd)
	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("start sing-box: %w", err)
	}
	return nil
}

// stop terminates sing-box and removes the temp config.
func (r *sbRunner) stop() {
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_, _ = r.cmd.Process.Wait()
	}
	if r.configPath != "" {
		_ = os.Remove(r.configPath)
	}
	r.cmd = nil
}
