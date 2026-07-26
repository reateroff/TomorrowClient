package vpn

import (
	"os"
	"path/filepath"
)

// binDir returns the directory of the running executable, where wintun.dll —
// the one runtime file the embedded sing-box core still needs — must live.
func binDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// BinDir exposes that directory to the developer tools.
func BinDir() string { return binDir() }
