//go:build windows

package vpn

import (
	"context"
	"fmt"

	sbjson "github.com/sagernet/sing/common/json"
	todaybox "github.com/tumgovic/todaycore"
	"github.com/tumgovic/todaycore/include"
	tclog "github.com/tumgovic/todaycore/log"
	"github.com/tumgovic/todaycore/option"
)

// TodayCore is sing-box under its own module path (third_party/todaycore), so
// this file mirrors sbrunner.go against that package tree. The config format is
// the same; the types are not, which is why the two cannot share code.

// startTodayBox runs a TodayCore config inside this process.
func startTodayBox(cfg []byte, logs *LogSink) (runner, error) {
	instance, cancel, err := newTodayInstance(cfg, todayLogBridge{sink: logs})
	if err != nil {
		return nil, err
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		cancel()
		return nil, fmt.Errorf("start TodayCore: %w", err)
	}
	return runnerFunc(func() {
		_ = instance.Close()
		cancel()
	}), nil
}

// newTodayInstance parses and builds a TodayCore instance without starting it.
func newTodayInstance(cfg []byte, logs tclog.PlatformWriter) (*todaybox.Box, context.CancelFunc, error) {
	ctx := todaybox.Context(context.Background(),
		include.InboundRegistry(),
		include.OutboundRegistry(),
		include.EndpointRegistry(),
		include.DNSTransportRegistry(),
		include.ServiceRegistry(),
		include.CertificateProviderRegistry(),
	)

	options, err := sbjson.UnmarshalExtendedContext[option.Options](ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("parse TodayCore config: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	instance, err := todaybox.New(todaybox.Options{
		Context:           ctx,
		Options:           options,
		PlatformLogWriter: logs,
	})
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("create TodayCore: %w", err)
	}
	return instance, cancel, nil
}

// todayLogBridge adapts TodayCore's platform log writer to the app's log sink.
type todayLogBridge struct{ sink *LogSink }

func (b todayLogBridge) WriteMessage(level tclog.Level, message string) {
	if b.sink == nil {
		return
	}
	b.sink.append(tclog.FormatLevel(level) + " " + message)
}
