//go:build windows

package vpn

import (
	"context"
	"fmt"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	sblog "github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	sbjson "github.com/sagernet/sing/common/json"
)

// startBox runs a sing-box config inside this process. The core is linked into
// the binary rather than shipped as sing-box.exe, so there is no subprocess to
// spawn, no temp config file on disk, and no orphan left behind if the app
// dies. wintun.dll is still required next to the executable: sing-box loads it
// at runtime to create the TUN adapter.
func startBox(cfg []byte, logs *LogSink) (runner, error) {
	instance, cancel, err := newInstance(cfg, logBridge{sink: logs})
	if err != nil {
		return nil, err
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		cancel()
		return nil, fmt.Errorf("start sing-box: %w", err)
	}
	return runnerFunc(func() {
		_ = instance.Close()
		cancel()
	}), nil
}

// newInstance parses a sing-box config and builds the core from it, without
// starting it — nothing touches the network adapter yet. Split out so the
// config generator can be verified against the real parser in a test.
func newInstance(cfg []byte, logs sblog.PlatformWriter) (*box.Box, context.CancelFunc, error) {
	// sing-box resolves inbound/outbound/DNS/service types through registries
	// carried on the context, so the context has to exist before the config can
	// even be parsed.
	ctx := box.Context(context.Background(),
		include.InboundRegistry(),
		include.OutboundRegistry(),
		include.EndpointRegistry(),
		include.DNSTransportRegistry(),
		include.ServiceRegistry(),
		include.CertificateProviderRegistry(),
	)

	options, err := sbjson.UnmarshalExtendedContext[option.Options](ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("parse sing-box config: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: options,
		// The platform log writer is how the core's own logs reach the UI, as
		// there is no stdout pipe to read.
		PlatformLogWriter: logs,
	})
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("create sing-box: %w", err)
	}
	return instance, cancel, nil
}

// logBridge adapts the sing-box platform log writer to the app's log sink.
type logBridge struct{ sink *LogSink }

// WriteMessage is called by sing-box for every log line it produces.
func (b logBridge) WriteMessage(level sblog.Level, message string) {
	if b.sink == nil {
		return
	}
	b.sink.append(sblog.FormatLevel(level) + " " + message)
}
