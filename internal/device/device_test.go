//go:build windows

package device

import "testing"

func TestDetect(t *testing.T) {
	a, b := Detect("ua"), Detect("ua")
	t.Logf("%+v", a)
	if len(a.HWID) != 32 || a.HWID != b.HWID {
		t.Errorf("HWID should be a stable 32-char hex: %q vs %q", a.HWID, b.HWID)
	}
	if a.OSVersion == "" || a.OS != "Windows" || a.Model == "" {
		t.Errorf("incomplete: %+v", a)
	}
}
