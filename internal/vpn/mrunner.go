//go:build windows

package vpn

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/listener"
	mlog "github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel"
	"github.com/metacubex/mihomo/tunnel/statistic"
)

// mihomo keeps its whole runtime in package-level state — one tunnel, one set
// of listeners, one log stream — so only one configuration can be live in the
// process. mihomoMu serializes starting and stopping it.
var mihomoMu sync.Mutex

// mihomoHome points mihomo's working directory (cache.db and the like) into
// the app's data directory instead of the user profile's .config.
func mihomoHome() {
	base, err := os.UserConfigDir()
	if err != nil {
		return
	}
	dir := filepath.Join(base, "TomorrowClient", "mihomo")
	_ = os.MkdirAll(dir, 0o755)
	C.SetHomeDir(dir)
}

// startMihomo applies a mihomo config and returns a runner that tears it down.
func startMihomo(cfg []byte, port int, logs *LogSink) (runner, error) {
	mihomoMu.Lock()
	defer mihomoMu.Unlock()

	mihomoHome()
	parsed, err := executor.ParseWithBytes(cfg)
	if err != nil {
		return nil, fmt.Errorf("parse mihomo config: %w", err)
	}

	sub := mlog.Subscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range sub {
			if logs != nil {
				logs.append(ev.Type() + " " + ev.Payload)
			}
		}
	}()

	executor.ApplyConfig(parsed, true)

	// teardown runs with mihomoMu held.
	teardown := func() {
		// Nothing listens, nothing stays connected, nothing holds a QUIC
		// session open once the tunnel is down.
		listener.PatchInboundListeners(nil, tunnel.Tunnel, true)
		statistic.DefaultManager.Range(func(c statistic.Tracker) bool {
			_ = c.Close()
			return true
		})
		old := tunnel.Proxies()
		tunnel.UpdateProxies(map[string]C.Proxy{}, nil)
		for _, p := range old {
			if closer, ok := p.(io.Closer); ok {
				_ = closer.Close()
			}
		}
		executor.Shutdown()
		mlog.UnSubscribe(sub)
		<-done
	}

	// ApplyConfig reports a failed listener only in the log. Wait for the port
	// instead of discovering it when the front's first connection is refused.
	if err := waitListening(port, 3*time.Second); err != nil {
		teardown()
		return nil, fmt.Errorf("mihomo: %w", err)
	}
	return runnerFunc(func() {
		mihomoMu.Lock()
		defer mihomoMu.Unlock()
		teardown()
	}), nil
}

// waitListening polls a loopback port until it accepts a connection.
func waitListening(port int, timeout time.Duration) error {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(timeout)
	for {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("listener on %s did not come up", addr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
