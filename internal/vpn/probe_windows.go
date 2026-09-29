//go:build windows

package vpn

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	madapter "github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/component/dialer"
	C "github.com/metacubex/mihomo/constant"

	"TomorrowClient/internal/mihomo"
	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
	"TomorrowClient/internal/xray"
)

// probeURL is fetched through the server being measured when the caller names
// no URL (a var so tests can point it at a local target). A 204 endpoint keeps
// the response to headers only, so the timing reflects the round trip and not
// the size of a body. Plain HTTP on purpose: a TLS handshake to the probe target
// would add its own negotiation on top of the proxy's.
var probeURL = model.DefaultPingURL

// probeTimeout bounds one measurement end to end when the caller sets none.
const probeTimeout = 8 * time.Second

// ProbeOptions shape one check through the proxy.
type ProbeOptions struct {
	Method  string        // http.MethodGet (default) or http.MethodHead
	URL     string        // default probeURL
	Timeout time.Duration // default probeTimeout
}

func (o ProbeOptions) withDefaults() ProbeOptions {
	if o.Method == "" {
		o.Method = http.MethodGet
	}
	if o.URL == "" {
		o.URL = probeURL
	}
	if o.Timeout <= 0 {
		o.Timeout = probeTimeout
	}
	return o
}

// maxConcurrentProbes caps how many servers are measured at once. Each probe
// starts its own core instance, so testing a large subscription without a cap
// would try to stand up dozens of cores at the same time.
const maxConcurrentProbes = 6

var probeSlots = make(chan struct{}, maxConcurrentProbes)

// ProbeLatency measures how long a real request through the given server takes
// on core c.
//
// It stands up a private instance of that core exposing just this profile on a
// loopback port, sends one request through it and tears the instance down. That
// is slower than a TCP handshake but it is the only measurement that actually
// proves the server works: the request only completes if the transport, TLS and
// credentials are all good — as that particular core implements them.
func ProbeLatency(p model.Profile, c model.Core, o ProbeOptions) (time.Duration, error) {
	o = o.withDefaults()
	probeSlots <- struct{}{}
	defer func() { <-probeSlots }()

	if c == model.CoreMihomo {
		return probeMihomo(p, o)
	}

	port, err := freeLoopbackPort()
	if err != nil {
		return 0, fmt.Errorf("reserve port: %w", err)
	}

	var proxy *url.URL
	switch c {
	case model.CoreSingBox, model.CoreTodayCore:
		f := singbox.Upstream
		if c == model.CoreTodayCore {
			f = singbox.TodayCore
		}
		cfg, err := singbox.BuildProbe(p, port, f)
		if err != nil {
			return 0, fmt.Errorf("build probe config: %w", err)
		}
		// nil log writer: a platform writer would make sing-box stand up its
		// Clash API, whose controller port is already held by the running
		// instance.
		var closeFn func()
		if c == model.CoreSingBox {
			instance, cancel, err := newInstance(cfg, nil)
			if err != nil {
				return 0, err
			}
			closeFn = func() { _ = instance.Close(); cancel() }
			if err := instance.Start(); err != nil {
				closeFn()
				return 0, fmt.Errorf("start probe: %w", err)
			}
		} else {
			instance, cancel, err := newTodayInstance(cfg, nil)
			if err != nil {
				return 0, err
			}
			closeFn = func() { _ = instance.Close(); cancel() }
			if err := instance.Start(); err != nil {
				closeFn()
				return 0, fmt.Errorf("start probe: %w", err)
			}
		}
		defer closeFn()
		proxy = &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", port)}

	case model.CoreXray:
		user, pass := randomToken(), randomToken()
		cfg, err := xray.Build(p, xray.Local{
			Port: port, Username: user, Password: pass,
			// Bound like the real connection, so the probe stays out of a
			// running tunnel and times the direct path to the server.
			Interface: bindInterface(p),
			LogLevel:  "none",
		})
		if err != nil {
			return 0, fmt.Errorf("build probe config: %w", err)
		}
		instance, err := newXrayInstance(cfg)
		if err != nil {
			return 0, err
		}
		defer instance.Close()
		if err := instance.Start(); err != nil {
			return 0, fmt.Errorf("start probe: %w", err)
		}
		proxy = &url.URL{Scheme: "socks5", User: url.UserPassword(user, pass), Host: fmt.Sprintf("127.0.0.1:%d", port)}

	default:
		return 0, fmt.Errorf("unknown core %q", c)
	}

	return timedRequest(&http.Transport{
		Proxy:               http.ProxyURL(proxy),
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: o.Timeout,
	}, o)
}

// timedRequest sends the check request over the given transport and times it
// from the first byte out to the response headers.
func timedRequest(tr *http.Transport, o ProbeOptions) (time.Duration, error) {
	client := &http.Client{
		Timeout:   o.Timeout,
		Transport: tr,
		// A redirect is still an answer from the far side.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, o.Method, o.URL, nil)
	if err != nil {
		return 0, err
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	elapsed := time.Since(start)
	_ = resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("probe returned HTTP %d", resp.StatusCode)
	}
	return elapsed, nil
}

// probeMihomo tests a profile with a standalone mihomo proxy. mihomo's runtime
// is process-wide and may be carrying the live connection, but a proxy adapter
// can be built and used on its own without touching it.
func probeMihomo(p model.Profile, o ProbeOptions) (time.Duration, error) {
	m, err := mihomo.Proxy(p)
	if err != nil {
		return 0, err
	}
	proxy, err := madapter.ParseProxy(m)
	if err != nil {
		return 0, fmt.Errorf("mihomo: %w", err)
	}
	defer func() {
		if c, ok := any(proxy).(io.Closer); ok {
			_ = c.Close()
		}
	}()

	// mihomo dials through a process-wide bound interface. Keep it on the
	// physical adapter so the probe stays out of a running tunnel; a running
	// mihomo core has it set to the same adapter.
	if iface := defaultInterface(); iface != "" {
		dialer.DefaultInterface.Store(iface)
	}

	// The request goes out through the adapter's own dialer, so GET and HEAD
	// behave exactly as for the other cores.
	return timedRequest(&http.Transport{
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: o.Timeout,
		DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
			var md C.Metadata
			if err := md.SetRemoteAddress(address); err != nil {
				return nil, err
			}
			return proxy.DialContext(ctx, &md)
		},
	}, o)
}

// freeLoopbackPort asks the OS for an unused port. There is a small window
// between closing the listener and the core binding it, which is acceptable
// here: a clash only fails this one attempt.
func freeLoopbackPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}
