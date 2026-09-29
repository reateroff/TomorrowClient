//go:build windows

package vpn

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"TomorrowClient/internal/mihomo"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
	"TomorrowClient/internal/xray"
)

// runner is a running tunnel, whatever cores it takes.
type runner interface{ stop() }

type runnerFunc func()

func (f runnerFunc) stop() { f() }

// startCore brings the tunnel up for p on core c, which must be a concrete core
// (see cores.Resolve).
func startCore(c model.Core, p model.Profile, s model.AppSettings, logs *LogSink) (runner, error) {
	switch c {
	case model.CoreSingBox:
		cfg, err := singbox.Build(p, s, singbox.Upstream)
		if err != nil {
			return nil, err
		}
		return startBox(cfg, logs)
	case model.CoreTodayCore:
		cfg, err := singbox.Build(p, s, singbox.TodayCore)
		if err != nil {
			return nil, err
		}
		return startTodayBox(cfg, logs)
	case model.CoreXray, model.CoreMihomo:
		return startFronted(c, p, s, logs)
	}
	return nil, fmt.Errorf("unknown core %q", c)
}

// startFronted runs Xray or mihomo behind a sing-box front. The front owns the
// TUN adapter, DNS and routing — one implementation of the user's rules for
// every core — and passes proxied traffic to the core's loopback SOCKS
// listener. The listener takes a per-connection random password, so no other
// program on the machine can use it as an open proxy.
func startFronted(c model.Core, p model.Profile, s model.AppSettings, logs *LogSink) (runner, error) {
	up, iface, err := newUpstream(p)
	if err != nil {
		return nil, err
	}

	var core runner
	switch c {
	case model.CoreXray:
		cfg, err := xray.Build(p, xray.Local{Port: up.Port, Username: up.Username, Password: up.Password, Interface: iface})
		if err != nil {
			return nil, err
		}
		core, err = startXray(cfg, logs)
		if err != nil {
			return nil, err
		}
	case model.CoreMihomo:
		cfg, err := mihomo.Build(p, mihomo.Local{Port: up.Port, Username: up.Username, Password: up.Password, Interface: iface})
		if err != nil {
			return nil, err
		}
		core, err = startMihomo(cfg, up.Port, logs)
		if err != nil {
			return nil, err
		}
	}

	frontCfg, err := singbox.BuildFront(s, up)
	if err != nil {
		core.stop()
		return nil, err
	}
	front, err := startBox(frontCfg, logs)
	if err != nil {
		core.stop()
		return nil, err
	}
	// Front first: nothing should reach the core while it is shutting down.
	return runnerFunc(func() {
		front.stop()
		core.stop()
	}), nil
}

// newUpstream picks the loopback port and credentials linking the front to a
// core, and the physical adapter the core should bind to.
func newUpstream(p model.Profile) (singbox.SocksUpstream, string, error) {
	port, err := freeLoopbackPort()
	if err != nil {
		return singbox.SocksUpstream{}, "", fmt.Errorf("reserve port: %w", err)
	}
	self, _ := os.Executable()
	return singbox.SocksUpstream{
		Port:     port,
		Username: randomToken(),
		Password: randomToken(),
		Server:   p.Address,
		Self:     self,
	}, bindInterface(p), nil
}

func randomToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Preview renders the configs core c would run for p, for the developer tools.
// A fronted core shows both halves. The loopback credentials are placeholders.
func Preview(c model.Core, p model.Profile, s model.AppSettings) (string, error) {
	switch c {
	case model.CoreSingBox, model.CoreTodayCore:
		f := singbox.Upstream
		if c == model.CoreTodayCore {
			f = singbox.TodayCore
		}
		b, err := singbox.Build(p, s, f)
		return string(b), err
	case model.CoreXray, model.CoreMihomo:
		self, _ := os.Executable()
		up := singbox.SocksUpstream{Port: 10808, Username: "user", Password: "pass", Server: p.Address, Self: self}
		iface := bindInterface(p)
		var core []byte
		var err error
		if c == model.CoreXray {
			core, err = xray.Build(p, xray.Local{Port: up.Port, Username: up.Username, Password: up.Password, Interface: iface})
		} else {
			core, err = mihomo.Build(p, mihomo.Local{Port: up.Port, Username: up.Username, Password: up.Password, Interface: iface})
		}
		if err != nil {
			return "", err
		}
		front, err := singbox.BuildFront(s, up)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("// %s\n%s\n\n// sing-box (TUN, DNS, маршрутизация)\n%s", c, core, front), nil
	}
	return "", fmt.Errorf("unknown core %q", c)
}
