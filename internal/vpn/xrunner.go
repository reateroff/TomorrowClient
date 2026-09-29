//go:build windows

package vpn

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"

	applog "github.com/xtls/xray-core/app/log"
	xlog "github.com/xtls/xray-core/common/log"
	xcore "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all" // every Xray protocol, transport and app
)

// xrayLogs is where Xray's console log goes. Xray picks its log handler from a
// process-wide registry, so the sink is process-wide too; probes run their
// instances with logging off and never write here.
var xrayLogs atomic.Pointer[LogSink]

func init() {
	_ = applog.RegisterHandlerCreator(applog.LogType_Console,
		func(applog.LogType, applog.HandlerCreatorOptions) (xlog.Handler, error) {
			return xrayLogHandler{}, nil
		})
}

type xrayLogHandler struct{}

func (xrayLogHandler) Handle(msg xlog.Message) {
	if sink := xrayLogs.Load(); sink != nil {
		sink.append(strings.TrimSpace(msg.String()))
	}
}

// newXrayInstance parses an Xray JSON config and builds the core without
// starting it.
func newXrayInstance(cfg []byte) (*xcore.Instance, error) {
	config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
	if err != nil {
		return nil, fmt.Errorf("parse Xray config: %w", err)
	}
	instance, err := xcore.New(config)
	if err != nil {
		return nil, fmt.Errorf("create Xray: %w", err)
	}
	return instance, nil
}

// startXray starts an Xray config and returns a runner that stops it.
func startXray(cfg []byte, logs *LogSink) (runner, error) {
	if logs != nil {
		xrayLogs.Store(logs)
	}
	instance, err := newXrayInstance(cfg)
	if err != nil {
		return nil, err
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		return nil, fmt.Errorf("start Xray: %w", err)
	}
	return runnerFunc(func() { _ = instance.Close() }), nil
}
