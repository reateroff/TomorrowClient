//go:build windows

package main

import (
	"testing"
)

func TestEmbeddedWindowIconsCreateAtAllShellSizes(t *testing.T) {
	destroy := windowIconUser32.NewProc("DestroyIcon")
	for _, size := range []int{16, 24, 32, 48, 64, 128, 256} {
		h, err := createWindowIcon(size)
		if err != nil || h == 0 {
			t.Fatalf("icon %d: %v", size, err)
		}
		destroy.Call(h)
	}
}

func TestIconImageRejectsInvalidData(t *testing.T) {
	for _, data := range [][]byte{nil, {}, {0, 0, 1, 0, 0, 0}, {0, 0, 1, 0, 1, 0}} {
		if _, err := iconImage(data, 32); err == nil {
			t.Fatal("invalid ICO accepted")
		}
	}
	bad := append([]byte(nil), trayIcon...)
	for i := 18; i < 22; i++ {
		bad[i] = 0xff
	}
	if _, err := iconImage(bad, 32); err == nil {
		t.Fatal("out-of-bounds image accepted")
	}
}
