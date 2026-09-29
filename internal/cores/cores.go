// Package cores knows which proxy engines the app carries, what each of them
// can run, and which one to use for a profile when the user leaves the choice
// to the app.
package cores

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"TomorrowClient/internal/model"

	todayconst "github.com/tumgovic/todaycore/constant"
	xraycore "github.com/xtls/xray-core/core"
)

// Info describes one core for the UI.
type Info struct {
	ID          model.Core `json:"id"`
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Description string     `json:"description"`
}

// All is every concrete core, in the order the UI lists them.
var All = []model.Core{model.CoreSingBox, model.CoreTodayCore, model.CoreXray, model.CoreMihomo}

// List returns the linked cores with the versions actually compiled in.
func List() []Info {
	return []Info{
		{model.CoreSingBox, "sing-box", moduleVersion("github.com/sagernet/sing-box"),
			"Туннель, DNS и прокси в одном экземпляре. Самый широкий набор протоколов."},
		{model.CoreTodayCore, "TodayCore", todayconst.TodayCoreVersion,
			"sing-box с XHTTP, VLESS Encryption и REALITY из Xray. Бета."},
		{model.CoreXray, "Xray", xraycore.Version(),
			"Эталонная реализация VLESS, REALITY и XHTTP."},
		{model.CoreMihomo, "mihomo", moduleVersion("github.com/metacubex/mihomo"),
			"Ядро Clash Meta: XHTTP, VLESS Encryption, AnyTLS и остальное."},
	}
}

// Name returns the display name of a core.
func Name(c model.Core) string {
	for _, i := range List() {
		if i.ID == c {
			return i.Name
		}
	}
	if c == model.CoreAuto {
		return "Авто"
	}
	return string(c)
}

// moduleVersion reads a dependency's version from the build info. Xray is not
// read this way: it tags releases v26.9.9, which Go cannot use as a module
// version, so its module carries a commit pseudo-version and the release number
// comes from the core itself.
func moduleVersion(path string) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, m := range bi.Deps {
		if m.Path == path {
			if m.Replace != nil && m.Replace.Version != "" {
				return strings.TrimPrefix(m.Replace.Version, "v")
			}
			return strings.TrimPrefix(m.Version, "v")
		}
	}
	return ""
}

// Resolve maps the configured core — possibly auto — to the core that runs p.
// A pinned core is returned as is together with the reason it cannot run p,
// if any, so the caller can refuse with a message instead of starting a core
// that will fail on the first packet.
func Resolve(setting model.Core, p model.Profile) (model.Core, error) {
	if setting != model.CoreAuto && setting != "" {
		return setting, Supports(setting, p)
	}
	for _, c := range preference(p) {
		if Supports(c, p) == nil {
			return c, nil
		}
	}
	return model.CoreSingBox, fmt.Errorf("ни одно ядро не поддерживает этот профиль: %w", Supports(model.CoreSingBox, p))
}

// preference orders the cores auto tries for a profile.
//
// Xray leads for what it is the reference implementation of — REALITY, XHTTP
// and VLESS Encryption — because servers built on current Xray are exactly the
// ones a port can fall behind on. Everything else goes to sing-box first: it
// carries the tunnel in a single instance, without a SOCKS hop to a second core.
func preference(p model.Profile) []model.Core {
	if p.Protocol == model.ProtoVLESS && (p.Security == "reality" || network(p) == "xhttp" || hasEncryption(p)) {
		return []model.Core{model.CoreXray, model.CoreTodayCore, model.CoreMihomo, model.CoreSingBox}
	}
	return []model.Core{model.CoreSingBox, model.CoreMihomo, model.CoreXray, model.CoreTodayCore}
}

// Supports reports why core c cannot run profile p, or nil if it can.
func Supports(c model.Core, p model.Profile) error {
	if p.Address == "" || (p.Port == 0 && p.Protocol != model.ProtoWireGuard) {
		return fmt.Errorf("в профиле нет адреса сервера")
	}
	switch c {
	case model.CoreSingBox:
		return singBoxSupports(p, false)
	case model.CoreTodayCore:
		return singBoxSupports(p, true)
	case model.CoreXray:
		return xraySupports(p)
	case model.CoreMihomo:
		return mihomoSupports(p)
	}
	return fmt.Errorf("неизвестное ядро %q", c)
}

func singBoxSupports(p model.Profile, today bool) error {
	name := Name(model.CoreSingBox)
	if today {
		name = Name(model.CoreTodayCore)
	}
	switch p.Protocol {
	case model.ProtoVLESS, model.ProtoVMess, model.ProtoTrojan, model.ProtoShadowsocks,
		model.ProtoHysteria, model.ProtoHysteria2, model.ProtoTUIC, model.ProtoWireGuard,
		model.ProtoAnyTLS, model.ProtoSOCKS, model.ProtoHTTP:
	default:
		return unsupported(name, "протокол "+string(p.Protocol))
	}
	switch n := network(p); n {
	case "tcp", "ws", "grpc", "http", "httpupgrade", "quic", "udp":
	case "xhttp":
		if !today {
			return unsupported(name, "транспорт XHTTP")
		}
		if extraHas(p, "downloadSettings") {
			return unsupported(name, "XHTTP downloadSettings")
		}
	default:
		return unsupported(name, "транспорт "+n)
	}
	if network(p) == "tcp" && p.HeaderType == "http" {
		return unsupported(name, "TCP с HTTP-обфускацией")
	}
	if hasEncryption(p) && !today {
		return unsupported(name, "VLESS Encryption")
	}
	switch p.Plugin {
	case "", "obfs-local", "simple-obfs", "v2ray-plugin":
	default:
		return unsupported(name, "плагин "+p.Plugin)
	}
	return nil
}

func xraySupports(p model.Profile) error {
	name := Name(model.CoreXray)
	switch p.Protocol {
	case model.ProtoVLESS, model.ProtoVMess, model.ProtoTrojan, model.ProtoShadowsocks,
		model.ProtoHysteria2, model.ProtoWireGuard, model.ProtoSOCKS, model.ProtoHTTP:
	default:
		return unsupported(name, "протокол "+string(p.Protocol))
	}
	switch n := network(p); n {
	case "tcp", "ws", "grpc", "httpupgrade", "xhttp", "kcp":
	case "http", "quic":
		return unsupported(name, "транспорт "+n+" (удалён из Xray)")
	default:
		if !udpProtocol(p) {
			return unsupported(name, "транспорт "+n)
		}
	}
	if p.AllowInsecure {
		return unsupported(name, "allowInsecure (удалён из Xray)")
	}
	if p.Protocol == model.ProtoVMess && p.AlterID > 0 {
		return unsupported(name, "VMess с alterId (только AEAD)")
	}
	if p.Plugin != "" {
		return unsupported(name, "плагины Shadowsocks")
	}
	if p.Ports != "" {
		return unsupported(name, "смена портов Hysteria2")
	}
	return nil
}

func mihomoSupports(p model.Profile) error {
	name := Name(model.CoreMihomo)
	switch p.Protocol {
	case model.ProtoVLESS, model.ProtoVMess, model.ProtoTrojan, model.ProtoShadowsocks,
		model.ProtoHysteria, model.ProtoHysteria2, model.ProtoTUIC, model.ProtoWireGuard,
		model.ProtoAnyTLS, model.ProtoSOCKS, model.ProtoHTTP:
	default:
		return unsupported(name, "протокол "+string(p.Protocol))
	}
	switch n := network(p); n {
	case "tcp", "ws", "grpc", "http", "httpupgrade":
	case "xhttp":
		if extraHas(p, "downloadSettings") {
			return unsupported(name, "XHTTP downloadSettings")
		}
	default:
		if !udpProtocol(p) {
			return unsupported(name, "транспорт "+n)
		}
	}
	switch p.Plugin {
	case "", "obfs-local", "simple-obfs", "obfs", "v2ray-plugin":
	default:
		return unsupported(name, "плагин "+p.Plugin)
	}
	return nil
}

func unsupported(core, what string) error {
	return fmt.Errorf("%s не поддерживает %s", core, what)
}

// network normalizes the transport name across the spellings share links use.
// QUIC-native protocols carry no transport of their own.
func network(p model.Profile) string {
	if udpProtocol(p) {
		return "udp"
	}
	switch n := strings.ToLower(p.Network); n {
	case "", "tcp", "raw", "none":
		return "tcp"
	case "websocket":
		return "ws"
	case "h2":
		return "http"
	case "splithttp":
		return "xhttp"
	case "mkcp":
		return "kcp"
	default:
		return n
	}
}

// Network is network exported for the config generators, so every core reads a
// profile's transport the same way.
func Network(p model.Profile) string { return network(p) }

// udpProtocol reports protocols that bring their own transport.
func udpProtocol(p model.Profile) bool {
	switch p.Protocol {
	case model.ProtoHysteria, model.ProtoHysteria2, model.ProtoTUIC, model.ProtoWireGuard:
		return true
	}
	return false
}

func hasEncryption(p model.Profile) bool {
	return p.Protocol == model.ProtoVLESS && p.Encryption != "" && p.Encryption != "none"
}

// HasEncryption reports whether a profile uses VLESS Encryption.
func HasEncryption(p model.Profile) bool { return hasEncryption(p) }

// Extra decodes a profile's XHTTP "extra" object, nil when absent or invalid.
func Extra(p model.Profile) map[string]any {
	if p.Extra == "" {
		return nil
	}
	var m map[string]any
	if json.Unmarshal([]byte(p.Extra), &m) != nil {
		return nil
	}
	return m
}

func extraHas(p model.Profile, key string) bool {
	_, ok := Extra(p)[key]
	return ok
}
