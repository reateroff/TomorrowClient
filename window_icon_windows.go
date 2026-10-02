//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const windowsAppID = "reater.TomorrowClient"

var (
	windowIconUser32               = windows.NewLazySystemDLL("user32.dll")
	windowIconShell32              = windows.NewLazySystemDLL("shell32.dll")
	iconCreate                     = windowIconUser32.NewProc("CreateIconFromResourceEx")
	iconEnumWindows                = windowIconUser32.NewProc("EnumWindows")
	iconWindowPID                  = windowIconUser32.NewProc("GetWindowThreadProcessId")
	iconClassName                  = windowIconUser32.NewProc("GetClassNameW")
	iconSendMessage                = windowIconUser32.NewProc("SendMessageW")
	iconSetClass                   = windowIconUser32.NewProc("SetClassLongPtrW")
	iconMetrics                    = windowIconUser32.NewProc("GetSystemMetrics")
	iconAppID                      = windowIconShell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	windowIconsOnce                sync.Once
	windowSmallIcon, windowBigIcon uintptr
	windowIconsErr                 error
)

// Give the installed app one stable identity across versions; never use a
// versioned identifier, which would split taskbar groups on every upgrade.
func setWindowsAppIdentity() {
	id, err := windows.UTF16PtrFromString(windowsAppID)
	if err == nil {
		iconAppID.Call(uintptr(unsafe.Pointer(id)))
	}
}

// iconImage selects a complete image from the embedded ICO, not from a shell
// shortcut or its cached icon. Prefer a larger source when sizes tie.
func iconImage(data []byte, size int) ([]byte, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, fmt.Errorf("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count == 0 || count > (len(data)-6)/16 {
		return nil, fmt.Errorf("invalid ICO directory")
	}
	best, bestScore := []byte(nil), int(^uint(0)>>1)
	for i := 0; i < count; i++ {
		e := data[6+i*16 : 6+(i+1)*16]
		w, h := int(e[0]), int(e[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		length, offset := uint64(binary.LittleEndian.Uint32(e[8:12])), uint64(binary.LittleEndian.Uint32(e[12:16]))
		if length == 0 || offset < uint64(6+count*16) || offset+length > uint64(len(data)) {
			return nil, fmt.Errorf("invalid ICO image bounds")
		}
		dw, dh := w-size, h-size
		score := dw*dw + dh*dh
		if w < size || h < size {
			score++
		}
		if score < bestScore {
			bestScore = score
			best = data[int(offset):int(offset+length)]
		}
	}
	return best, nil
}

func createWindowIcon(size int) (uintptr, error) {
	data, err := iconImage(trayIcon, size)
	if err != nil {
		return 0, err
	}
	handle, _, callErr := iconCreate.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 1, 0x30000, uintptr(size), uintptr(size), 0)
	runtime.KeepAlive(data)
	if handle == 0 {
		return 0, fmt.Errorf("create window icon: %w", callErr)
	}
	return handle, nil
}

func applyWindowIcons(hwnd, small, big uintptr) {
	// Wails sets ICON_SMALL only. Explicitly set both window and class icons so
	// taskbar/Alt+Tab never fall back to a stale or unrelated class icon.
	iconSendMessage.Call(hwnd, 0x80, 0, small)
	iconSendMessage.Call(hwnd, 0x80, 1, big)
	iconSetClass.Call(hwnd, ^uintptr(13), big)   // GCLP_HICON = -14
	iconSetClass.Call(hwnd, ^uintptr(33), small) // GCLP_HICONSM = -34
}

func syncWindowsWindowIcon() error {
	// Keep these two handles for the process lifetime; Windows reclaims them at
	// exit. Repeated DOM-ready callbacks reuse them rather than leaking icons.
	windowIconsOnce.Do(func() {
		small, _, _ := iconMetrics.Call(49)
		big, _, _ := iconMetrics.Call(11)
		windowSmallIcon, windowIconsErr = createWindowIcon(int(small))
		if windowIconsErr != nil {
			return
		}
		windowBigIcon, windowIconsErr = createWindowIcon(int(big))
	})
	if windowIconsErr != nil {
		return windowIconsErr
	}
	found := false
	callback := windows.NewCallback(func(hwnd, _ uintptr) uintptr {
		var pid uint32
		iconWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid != uint32(os.Getpid()) {
			return 1
		}
		var name [128]uint16
		iconClassName.Call(hwnd, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
		if windows.UTF16ToString(name[:]) != "wailsWindow" {
			return 1
		}
		applyWindowIcons(hwnd, windowSmallIcon, windowBigIcon)
		found = true
		return 1
	})
	iconEnumWindows.Call(callback, 0)
	if !found {
		return fmt.Errorf("main window not found for icon update")
	}
	return nil
}
