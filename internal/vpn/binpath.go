package vpn

import (
	"os"
	"path/filepath"
)

// binDir returns the directory of the running executable, where the bundled
// core binaries (sing-box.exe, xray.exe, tun2socks.exe) and wintun.dll live.
func binDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// binPath resolves a bundled binary name against the executable directory.
func binPath(name string) string {
	return filepath.Join(binDir(), name)
}
