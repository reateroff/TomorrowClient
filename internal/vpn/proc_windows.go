//go:build windows

package vpn

import (
	"os/exec"
	"syscall"
)

// hidden configures a command so its console window is not shown when spawned
// from the GUI app.
func hidden(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
