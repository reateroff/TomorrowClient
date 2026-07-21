//go:build windows

package main

import (
	"os/exec"
	"sort"
	"strings"
	"syscall"
)

// ListProcesses returns the distinct executable names of currently running
// processes (e.g. "chrome.exe"), sorted alphabetically. Used by the routing
// tab to pick a process without typing its name by hand.
func (a *App) ListProcesses() []string {
	cmd := exec.Command("tasklist", "/fo", "csv", "/nh")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "\"") {
			continue
		}
		// CSV: "name.exe","PID","Session","Session#","Mem"
		name := line[1:]
		if i := strings.Index(name, "\""); i >= 0 {
			name = name[:i]
		}
		if name == "" || !strings.HasSuffix(strings.ToLower(name), ".exe") {
			continue
		}
		seen[name] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	return names
}
