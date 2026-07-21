//go:build windows

// Package startup manages the "launch at Windows startup" behaviour by creating
// or removing a Task Scheduler task that runs the app with highest privileges.
// A scheduled task is used instead of a Run registry key because the app needs
// administrator rights and a task with highest privileges starts without a UAC
// prompt.
package startup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// taskName is the Task Scheduler entry name.
const taskName = "TomorrowClientAutostart"

// hidden runs a command without flashing a console window.
func hidden(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// Enabled reports whether the autostart task currently exists.
func Enabled() bool {
	cmd := exec.Command("schtasks", "/query", "/tn", taskName)
	hidden(cmd)
	return cmd.Run() == nil
}

// Set creates or removes the autostart task to match the desired state.
func Set(enable bool) error {
	if enable {
		return create()
	}
	return remove()
}

// create registers a logon-triggered task that launches the current executable
// with the highest available privileges.
func create() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	cmd := exec.Command("schtasks",
		"/create", "/f",
		"/tn", taskName,
		"/sc", "onlogon",
		"/rl", "highest",
		"/tr", "\""+exe+"\"",
	)
	hidden(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// remove deletes the autostart task, ignoring "not found".
func remove() error {
	cmd := exec.Command("schtasks", "/delete", "/f", "/tn", taskName)
	hidden(cmd)
	_ = cmd.Run()
	return nil
}
