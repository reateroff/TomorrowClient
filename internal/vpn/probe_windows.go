//go:build windows

package vpn

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"TomorrowClient/internal/model"
	"TomorrowClient/internal/singbox"
)

// probeURL is fetched through the server being measured. A 204 endpoint keeps
// the response to headers only, so the timing reflects the round trip and not
// the size of a body. Plain HTTP on purpose: a TLS handshake to the probe target
// would add its own negotiation on top of the proxy's.
const probeURL = "http://cp.cloudflare.com/generate_204"

// probeTimeout bounds one measurement end to end.
const probeTimeout = 8 * time.Second

// maxConcurrentProbes caps how many servers are measured at once. Each probe
// starts its own sing-box instance, so testing a large subscription without a
// cap would try to stand up dozens of cores at the same time.
const maxConcurrentProbes = 6

var probeSlots = make(chan struct{}, maxConcurrentProbes)

// ProbeLatency measures how long a real request through the given server takes.
//
// It stands up a private sing-box instance exposing just that profile on a
// loopback port, sends one request through it and tears the instance down. That
// is slower than a TCP handshake but it is the only measurement that actually
// proves the server works: the request only completes if the transport, TLS and
// credentials are all good.
func ProbeLatency(p model.Profile) (time.Duration, error) {
	probeSlots <- struct{}{}
	defer func() { <-probeSlots }()

	port, err := freeLoopbackPort()
	if err != nil {
		return 0, fmt.Errorf("reserve port: %w", err)
	}

	cfg, err := singbox.BuildProbe(p, port)
	if err != nil {
		return 0, fmt.Errorf("build probe config: %w", err)
	}

	// nil log writer: a platform writer would make sing-box stand up its Clash
	// API, whose controller port is already held by the running instance.
	instance, cancel, err := newInstance(cfg, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = instance.Close()
		cancel()
	}()

	if err := instance.Start(); err != nil {
		return 0, fmt.Errorf("start probe: %w", err)
	}

	proxy, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		return 0, err
	}
	client := &http.Client{
		Timeout: probeTimeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(proxy),
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: probeTimeout,
		},
	}

	ctx, cancelReq := context.WithTimeout(context.Background(), probeTimeout)
	defer cancelReq()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
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

// freeLoopbackPort asks the OS for an unused port. There is a small window
// between closing the listener and sing-box binding it, which is acceptable
// here: a clash only fails this one measurement.
func freeLoopbackPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}
