//go:build windows

// Package vpn is the connection engine. It owns the active core subprocess,
// tracks the connection state, and pushes live status snapshots to the UI.
package vpn

import (
	"context"
	"sync"
	"time"

	"TomorrowClient/internal/model"
)

// Emitter pushes a status snapshot to the frontend. app.go wires this to the
// Wails runtime event emitter.
type Emitter func(model.Status)

// Engine manages a single active connection.
type Engine struct {
	mu       sync.Mutex
	state    model.ConnState
	core     model.Core
	profile  *model.Profile
	errMsg   string
	connAt   int64
	stats    model.Stats
	baseRx   uint64 // adapter counters at connect time (for per-session totals)
	baseTx   uint64
	lastRx   uint64
	lastTx   uint64
	lastTick time.Time

	sb   *sbRunner
	xr   *xRunner
	emit Emitter

	statsCancel context.CancelFunc
}

// New creates an engine with the given status emitter.
func New(emit Emitter) *Engine {
	return &Engine{state: model.StateDisconnected, emit: emit}
}

// Status returns the current snapshot.
func (e *Engine) Status() model.Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshot()
}

// snapshot builds a Status; caller must hold the lock.
func (e *Engine) snapshot() model.Status {
	return model.Status{
		State:         e.state,
		Core:          e.core,
		ActiveProfile: e.profile,
		Error:         e.errMsg,
		Stats:         e.stats,
		ConnectedAt:   e.connAt,
	}
}

// push emits the current snapshot; caller must hold the lock.
func (e *Engine) push() {
	if e.emit != nil {
		e.emit(e.snapshot())
	}
}

func (e *Engine) setState(st model.ConnState, errMsg string) {
	e.state = st
	e.errMsg = errMsg
	e.push()
}

// Connect brings the tunnel up for the given profile using the selected core.
func (e *Engine) Connect(p model.Profile, s model.AppSettings) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state == model.StateConnected || e.state == model.StateConnecting {
		return nil
	}

	e.core = s.Core
	e.profile = &p
	e.stats = model.Stats{}
	e.baseRx, e.baseTx = 0, 0
	e.setState(model.StateConnecting, "")

	var err error
	switch s.Core {
	case model.CoreXray:
		e.xr = &xRunner{}
		err = e.xr.start(p, s)
	default: // sing-box is the default
		e.sb = &sbRunner{}
		err = e.sb.start(p, s)
	}
	if err != nil {
		e.teardownLocked()
		e.setState(model.StateError, err.Error())
		return err
	}

	e.connAt = time.Now().UnixMilli()
	e.lastTick = time.Now()
	e.setState(model.StateConnected, "")
	e.startStatsLoop()
	return nil
}

// Disconnect tears the tunnel down.
func (e *Engine) Disconnect() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == model.StateDisconnected {
		return
	}
	e.teardownLocked()
	e.stats = model.Stats{}
	e.connAt = 0
	e.setState(model.StateDisconnected, "")
}

// teardownLocked stops the running core and stats loop; caller holds the lock.
func (e *Engine) teardownLocked() {
	if e.statsCancel != nil {
		e.statsCancel()
		e.statsCancel = nil
	}
	if e.sb != nil {
		e.sb.stop()
		e.sb = nil
	}
	if e.xr != nil {
		e.xr.stop()
		e.xr = nil
	}
}

// startStatsLoop polls the WinTun adapter counters once a second and pushes
// updated traffic figures to the UI. Caller holds the lock.
func (e *Engine) startStatsLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	e.statsCancel = cancel
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.tickStats()
			}
		}
	}()
}

// tickStats reads the adapter counters and updates totals + speed.
func (e *Engine) tickStats() {
	rx, tx, ok := adapterBytes()
	if !ok {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != model.StateConnected {
		return
	}
	// First reading establishes the baseline so totals are per-session.
	if e.baseRx == 0 && e.baseTx == 0 {
		e.baseRx, e.baseTx = rx, tx
		e.lastRx, e.lastTx = rx, tx
		e.lastTick = time.Now()
		return
	}

	now := time.Now()
	dt := now.Sub(e.lastTick).Seconds()
	if dt <= 0 {
		dt = 1
	}
	e.stats.Download = rx - e.baseRx
	e.stats.Upload = tx - e.baseTx
	e.stats.DownloadSpeed = uint64(float64(rx-e.lastRx) / dt)
	e.stats.UploadSpeed = uint64(float64(tx-e.lastTx) / dt)
	e.lastRx, e.lastTx = rx, tx
	e.lastTick = now
	e.push()
}
