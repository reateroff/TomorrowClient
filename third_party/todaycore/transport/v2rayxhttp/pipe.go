// Uplink buffering pipe for XHTTP packet-up mode.
//
// Xray implements this with transport/pipe + buf.MultiBuffer. sing-box has no
// equivalent, so this is a byte-oriented reimplementation with the same
// observable behaviour:
//
//   - Write() blocks while the buffered amount is at or above the size limit,
//     so that back-pressure reaches the proxy layer.
//   - ReadChunk() coalesces everything currently buffered into a single chunk
//     of at most maxLen bytes. This is what batches many small conn.Write()
//     calls into one large POST; without it throughput collapses.
//   - Interrupt() aborts both directions immediately (used when an upload
//     request fails).
//
// Licensed under GPL-3.0-or-later.

package v2rayxhttp

import (
	"io"
	"sync"
)

type uploadPipe struct {
	access    sync.Mutex
	cond      *sync.Cond
	buffer    []byte
	sizeLimit int
	closed    bool
	err       error
}

func newUploadPipe(sizeLimit int) *uploadPipe {
	if sizeLimit < 0 {
		sizeLimit = 0
	}
	p := &uploadPipe{sizeLimit: sizeLimit}
	p.cond = sync.NewCond(&p.access)
	return p
}

func (p *uploadPipe) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	p.access.Lock()
	defer p.access.Unlock()
	written := 0
	for written < len(b) {
		for !p.closed && p.sizeLimit > 0 && len(p.buffer) >= p.sizeLimit {
			p.cond.Wait()
		}
		if p.closed {
			if p.err != nil {
				return written, p.err
			}
			return written, io.ErrClosedPipe
		}
		chunk := len(b) - written
		if p.sizeLimit > 0 {
			space := p.sizeLimit - len(p.buffer)
			// WithSizeLimit(0) in Xray still lets single bytes through; keep
			// at least one byte of progress so we can never deadlock.
			if space < 1 {
				space = 1
			}
			if chunk > space {
				chunk = space
			}
		}
		p.buffer = append(p.buffer, b[written:written+chunk]...)
		written += chunk
		p.cond.Broadcast()
	}
	return written, nil
}

// ReadChunk returns up to maxLen buffered bytes, blocking until data is
// available or the pipe is closed.
func (p *uploadPipe) ReadChunk(maxLen int) ([]byte, error) {
	p.access.Lock()
	defer p.access.Unlock()
	for len(p.buffer) == 0 {
		if p.closed {
			if p.err != nil {
				return nil, p.err
			}
			return nil, io.EOF
		}
		p.cond.Wait()
	}
	if maxLen <= 0 || maxLen > len(p.buffer) {
		maxLen = len(p.buffer)
	}
	chunk := make([]byte, maxLen)
	copy(chunk, p.buffer[:maxLen])
	p.buffer = p.buffer[maxLen:]
	if len(p.buffer) == 0 {
		p.buffer = nil
	}
	p.cond.Broadcast()
	return chunk, nil
}

// Close performs a graceful shutdown: buffered data stays readable.
func (p *uploadPipe) Close() error {
	p.access.Lock()
	defer p.access.Unlock()
	if !p.closed {
		p.closed = true
		p.cond.Broadcast()
	}
	return nil
}

// Interrupt aborts the pipe and discards buffered data.
func (p *uploadPipe) Interrupt(err error) {
	p.access.Lock()
	defer p.access.Unlock()
	if p.closed && p.err != nil {
		return
	}
	p.closed = true
	if err == nil {
		err = io.ErrClosedPipe
	}
	p.err = err
	p.buffer = nil
	p.cond.Broadcast()
}
