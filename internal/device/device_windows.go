//go:build windows

// Package device describes this machine to subscription panels. Panels such as
// Remnawave and Marzban limit how many devices a subscription may use, and
// tell devices apart by headers the client sends with every subscription
// request: a hardware ID plus the OS, its version and the device model.
package device

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Info is what the client reports about the device.
type Info struct {
	HWID      string `json:"hwid"`
	OS        string `json:"os"`
	OSVersion string `json:"osVersion"`
	Model     string `json:"model"`
	UserAgent string `json:"userAgent"`
}

// Detect reads the real values from the system. HWID is derived from the
// Windows MachineGuid rather than sent as is: it stays stable across app
// reinstalls and reboots, yet does not hand the panel the machine's own
// identifier.
func Detect(userAgent string) Info {
	return Info{
		HWID:      hwid(),
		OS:        "Windows",
		OSVersion: osVersion(),
		Model:     model(),
		UserAgent: userAgent,
	}
}

func hwid() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer k.Close()
	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil || guid == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("TomorrowClient:" + strings.ToLower(guid)))
	return hex.EncodeToString(sum[:16])
}

func osVersion() string {
	v := windows.RtlGetVersion()
	return fmt.Sprintf("%d.%d.%d", v.MajorVersion, v.MinorVersion, v.BuildNumber)
}

// model is the BIOS product name, prefixed with the maker when the name alone
// says little ("MS-7C56" → "Micro-Star MS-7C56").
func model() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\BIOS`, registry.QUERY_VALUE)
	if err != nil {
		return "PC"
	}
	defer k.Close()
	product, _, _ := k.GetStringValue("SystemProductName")
	maker, _, _ := k.GetStringValue("SystemManufacturer")
	product, maker = clean(product), clean(maker)
	switch {
	case product == "":
		return "PC"
	case maker == "" || strings.Contains(strings.ToLower(product), strings.ToLower(firstWord(maker))):
		return product
	default:
		return firstWord(maker) + " " + product
	}
}

// clean drops the placeholder strings OEMs leave in the BIOS.
func clean(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "to be filled by o.e.m.", "system product name", "system manufacturer", "default string", "o.e.m.":
		return ""
	}
	return s
}

func firstWord(s string) string {
	if i := strings.IndexAny(s, " ,"); i > 0 {
		return s[:i]
	}
	return s
}

// IncyHWID follows INCY's published Windows algorithm. It is independently
// derived; it does not read/claim identity from an installed INCY instance.
func IncyHWID() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer k.Close()
	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil || guid == "" {
		return ""
	}
	host, _ := os.Hostname()
	osName := "Windows 10"
	if windows.RtlGetVersion().BuildNumber >= 22000 {
		osName = "Windows 11"
	}
	raw := strings.Join([]string{guid, host, osName, runtime.GOARCH, os.Getenv("USERNAME")}, "|")
	deviceID := sha256.Sum256([]byte(raw))
	hash := sha256.Sum256([]byte("incy_hwid_" + hex.EncodeToString(deviceID[:])))
	h := strings.ToUpper(hex.EncodeToString(hash[:16]))
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
