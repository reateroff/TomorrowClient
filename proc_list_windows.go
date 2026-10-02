//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ProcessInfo is one running executable surfaced to the routing tab: its file
// name (e.g. "chrome.exe") plus, when we can extract it, the app icon encoded
// as a PNG data URL so the UI can render the real icon instead of a placeholder.
type ProcessInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Application bool   `json:"application"`
	Icon        string `json:"icon"` // "data:image/png;base64,..." or "" when unavailable
}

// Lazy WinAPI handles used for icon extraction.
var (
	modShell32 = windows.NewLazySystemDLL("shell32.dll")
	modUser32  = windows.NewLazySystemDLL("user32.dll")
	modGdi32   = windows.NewLazySystemDLL("gdi32.dll")

	procSHGetFileInfo = modShell32.NewProc("SHGetFileInfoW")
	procGetIconInfo   = modUser32.NewProc("GetIconInfo")
	procDestroyIcon   = modUser32.NewProc("DestroyIcon")
	procGetDC         = modUser32.NewProc("GetDC")
	procReleaseDC     = modUser32.NewProc("ReleaseDC")
	procGetObject     = modGdi32.NewProc("GetObjectW")
	procGetDIBits     = modGdi32.NewProc("GetDIBits")
	procDeleteObject  = modGdi32.NewProc("DeleteObject")
)

const (
	shgfiIcon      = 0x000000100
	shgfiLargeIcon = 0x000000000
)

type shFileInfo struct {
	hIcon         uintptr
	iIcon         int32
	dwAttributes  uint32
	szDisplayName [260]uint16
	szTypeName    [80]uint16
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

type gdiBitmap struct {
	bmType       int32
	bmWidth      int32
	bmHeight     int32
	bmWidthBytes int32
	bmPlanes     uint16
	bmBitsPixel  uint16
	bmBits       uintptr
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

// ListProcesses returns the distinct running executables (by file name), sorted
// alphabetically, each with its app icon when extractable. Used by the routing
// tab so the user can pick a process — and see its real icon — without typing.
func (a *App) ListProcesses() []ProcessInfo {
	appPIDs := visibleApplicationPIDs()
	applications := map[string]bool{}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snapshot)

	// Distinct exe name -> first full path we saw for it.
	paths := map[string]string{}
	order := []string{}

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil
	}
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if name != "" && strings.HasSuffix(strings.ToLower(name), ".exe") {
			key := strings.ToLower(name)
			if appPIDs[entry.ProcessID] {
				applications[key] = true
			}
			if _, ok := paths[key]; !ok {
				paths[key] = processPath(entry.ProcessID)
				order = append(order, name)
			}
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}

	sort.Slice(order, func(i, j int) bool {
		return strings.ToLower(order[i]) < strings.ToLower(order[j])
	})

	iconCache := map[string]string{} // path -> data URL (avoid re-extracting)
	out := make([]ProcessInfo, 0, len(order))
	for _, name := range order {
		p := paths[strings.ToLower(name)]
		icon := ""
		if p != "" {
			if cached, ok := iconCache[p]; ok {
				icon = cached
			} else {
				icon = iconForPath(p)
				iconCache[p] = icon
			}
		}
		out = append(out, ProcessInfo{Name: name, Icon: icon, Path: p, Application: applications[strings.ToLower(name)]})
	}
	return out
}

// processPath resolves a PID to its full executable path, or "" if inaccessible
// (system/protected processes commonly deny access).
func processPath(pid uint32) string {
	const queryLimited = 0x1000 // PROCESS_QUERY_LIMITED_INFORMATION
	h, err := windows.OpenProcess(queryLimited, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

// iconForPath extracts the executable's icon and returns it as a PNG data URL,
// or "" on any failure. All GDI/USER handles are released before returning.
func iconForPath(path string) string {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}

	var shfi shFileInfo
	r, _, _ := procSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(p)),
		0,
		uintptr(unsafe.Pointer(&shfi)),
		unsafe.Sizeof(shfi),
		uintptr(shgfiIcon|shgfiLargeIcon),
	)
	if r == 0 || shfi.hIcon == 0 {
		return ""
	}
	defer procDestroyIcon.Call(shfi.hIcon)

	var ii iconInfo
	if r, _, _ := procGetIconInfo.Call(shfi.hIcon, uintptr(unsafe.Pointer(&ii))); r == 0 {
		return ""
	}
	if ii.hbmMask != 0 {
		defer procDeleteObject.Call(ii.hbmMask)
	}
	if ii.hbmColor == 0 {
		return ""
	}
	defer procDeleteObject.Call(ii.hbmColor)

	var bmp gdiBitmap
	if r, _, _ := procGetObject.Call(ii.hbmColor, unsafe.Sizeof(bmp), uintptr(unsafe.Pointer(&bmp))); r == 0 {
		return ""
	}
	w, h := int(bmp.bmWidth), int(bmp.bmHeight)
	if w <= 0 || h <= 0 || w > 512 || h > 512 {
		return ""
	}

	bi := bitmapInfoHeader{
		biSize:        40,
		biWidth:       int32(w),
		biHeight:      -int32(h), // negative => top-down rows
		biPlanes:      1,
		biBitCount:    32,
		biCompression: 0, // BI_RGB
	}

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return ""
	}
	defer procReleaseDC.Call(0, hdc)

	buf := make([]byte, w*h*4)
	r, _, _ = procGetDIBits.Call(
		hdc,
		ii.hbmColor,
		0,
		uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
	)
	if r == 0 {
		return ""
	}

	// Buffer is BGRA, top-down. Some icons ship without an alpha channel (all
	// zero) — treat those as fully opaque so they don't render invisible.
	hasAlpha := false
	for i := 3; i < len(buf); i += 4 {
		if buf[i] != 0 {
			hasAlpha = true
			break
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		b := buf[i*4]
		g := buf[i*4+1]
		rr := buf[i*4+2]
		aa := buf[i*4+3]
		if !hasAlpha {
			aa = 255
		}
		img.Pix[i*4] = rr
		img.Pix[i*4+1] = g
		img.Pix[i*4+2] = b
		img.Pix[i*4+3] = aa
	}

	// Legacy icons carry transparency in a separate 1bpp AND mask rather than an
	// alpha channel; without it they would render as opaque squares.
	if !hasAlpha && ii.hbmMask != 0 {
		applyIconMask(hdc, ii.hbmMask, img, w, h)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
}

// applyIconMask punches transparency into img using the icon's 1bpp AND mask,
// where a set bit means "transparent". Rows are padded to 4-byte boundaries.
func applyIconMask(hdc, hbmMask uintptr, img *image.RGBA, w, h int) {
	stride := ((w + 31) / 32) * 4
	buf := make([]byte, stride*h)

	// BITMAPINFO for a 1bpp DIB: header plus a two-entry colour table.
	var bi struct {
		hdr    bitmapInfoHeader
		colors [2]uint32
	}
	bi.hdr = bitmapInfoHeader{
		biSize:     40,
		biWidth:    int32(w),
		biHeight:   -int32(h), // top-down
		biPlanes:   1,
		biBitCount: 1,
	}

	r, _, _ := procGetDIBits.Call(
		hdc,
		hbmMask,
		0,
		uintptr(h),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
	)
	if r == 0 {
		return
	}

	for y := 0; y < h; y++ {
		row := y * stride
		for x := 0; x < w; x++ {
			if buf[row+x/8]&(0x80>>(x%8)) != 0 {
				img.Pix[(y*w+x)*4+3] = 0
			}
		}
	}
}

// Enumerate visible unowned top-level windows: the same distinction users expect
// from Task Manager's "Apps" rather than system/service executable heuristics.
func visibleApplicationPIDs() map[uint32]bool {
	result := map[uint32]bool{}
	enum := modUser32.NewProc("EnumWindows")
	visible := modUser32.NewProc("IsWindowVisible")
	owner := modUser32.NewProc("GetWindow")
	pidProc := modUser32.NewProc("GetWindowThreadProcessId")
	callback := windows.NewCallback(func(hwnd, _ uintptr) uintptr {
		v, _, _ := visible.Call(hwnd)
		o, _, _ := owner.Call(hwnd, 4)
		if v != 0 && o == 0 {
			var pid uint32
			pidProc.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
			result[pid] = true
		}
		return 1
	})
	enum.Call(callback, 0)
	return result
}
