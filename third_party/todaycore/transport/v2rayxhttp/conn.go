// This file is derived from Xray-core (transport/internet/splithttp/connection.go
// and client.go), licensed under the Mozilla Public License 2.0.
// Source: https://github.com/XTLS/Xray-core (v26.9.9)
//
// Modifications for sing-box are licensed under GPL-3.0-or-later.

package v2rayxhttp

import (
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
)

// splitConn glues the (possibly separate) uplink and downlink halves of an
// XHTTP session into a single net.Conn.
type splitConn struct {
	writer     io.WriteCloser
	reader     io.ReadCloser
	remoteAddr net.Addr
	localAddr  net.Addr
	onClose    func()
	closeOnce  sync.Once
}

func (c *splitConn) Write(b []byte) (int, error) {
	return c.writer.Write(b)
}

func (c *splitConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

// Close differs from Xray on purpose.
//
// Xray's splitConn.Close() has `if err2 != nil { return err }`, which returns
// the (nil) uplink error and silently swallows the downlink error. Here both
// errors are reported. Also guarded with sync.Once because sing-box closes
// connections from several layers.
func (c *splitConn) Close() error {
	c.closeOnce.Do(func() {
		if c.onClose != nil {
			c.onClose()
		}
	})
	var err, err2 error
	if c.writer != nil {
		err = c.writer.Close()
	}
	if c.reader != nil {
		err2 = c.reader.Close()
	}
	return E.Errors(err, err2)
}

// LocalAddr and RemoteAddr fall back to an empty Socksaddr instead of a nil
// net.Addr: sing-box logs and routes on these and would panic on nil.
func (c *splitConn) LocalAddr() net.Addr {
	if c.localAddr == nil {
		return M.Socksaddr{}
	}
	return c.localAddr
}

func (c *splitConn) RemoteAddr() net.Addr {
	if c.remoteAddr == nil {
		return M.Socksaddr{}
	}
	return c.remoteAddr
}

// Deadlines cannot be implemented over HTTP request/response bodies.
// Xray returns nil here; returning os.ErrInvalid tells sing-box to apply its
// own read deadline shim instead of silently ignoring the request.
func (c *splitConn) SetDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *splitConn) SetReadDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *splitConn) SetWriteDeadline(t time.Time) error {
	return os.ErrInvalid
}

func (c *splitConn) NeedAdditionalReadDeadline() bool {
	return true
}

// uploadWriter enforces the exact per-POST size limit on top of uploadPipe.
type uploadWriter struct {
	*uploadPipe
	maxLen int32
}

func (w uploadWriter) Write(b []byte) (int, error) {
	return w.uploadPipe.Write(b)
}

// WaitReadCloser lets Dial() return before the downlink response headers have
// arrived. Reads block until the body shows up.
type WaitReadCloser struct {
	wait   chan struct{}
	close  sync.Once
	reader atomic.Pointer[io.ReadCloser]
}

func NewWaitReadCloser() *WaitReadCloser {
	return &WaitReadCloser{wait: make(chan struct{})}
}

func (w *WaitReadCloser) Set(rc io.ReadCloser) {
	w.reader.Store(&rc)
	select {
	case <-w.wait:
		// Already closed while we were waiting for the response: drop it.
		if p := w.reader.Swap(nil); p != nil {
			(*p).Close()
		}
		return
	default:
	}
	w.close.Do(func() { close(w.wait) })
}

func (w *WaitReadCloser) Read(b []byte) (int, error) {
	rc := w.reader.Load()
	if rc == nil {
		<-w.wait
		if rc = w.reader.Load(); rc == nil {
			return 0, io.ErrClosedPipe
		}
	}
	return (*rc).Read(b)
}

func (w *WaitReadCloser) Close() error {
	w.close.Do(func() { close(w.wait) })
	if p := w.reader.Swap(nil); p != nil {
		return (*p).Close()
	}
	return nil
}
