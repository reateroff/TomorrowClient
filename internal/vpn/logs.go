//go:build windows

package vpn

import (
	"bufio"
	"io"
	"sync"
)

// logBufferSize is how many recent log lines are kept in memory for the UI.
const logBufferSize = 500

// LogSink is a ring buffer of the running core's log lines. It keeps the last
// logBufferSize lines and forwards each new line to an optional listener so the
// UI can live-tail the output.
type LogSink struct {
	mu    sync.Mutex
	lines []string
	onLog func(string)
}

// newLogSink creates an empty sink with the given per-line callback.
func newLogSink(onLog func(string)) *LogSink {
	return &LogSink{lines: make([]string, 0, logBufferSize), onLog: onLog}
}

// append stores a line, trimming the buffer to its cap, and notifies listeners.
func (s *LogSink) append(line string) {
	s.mu.Lock()
	s.lines = append(s.lines, line)
	if len(s.lines) > logBufferSize {
		s.lines = s.lines[len(s.lines)-logBufferSize:]
	}
	cb := s.onLog
	s.mu.Unlock()
	if cb != nil {
		cb(line)
	}
}

// Snapshot returns a copy of the buffered lines.
func (s *LogSink) Snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.lines))
	copy(out, s.lines)
	return out
}

// Clear empties the buffer.
func (s *LogSink) Clear() {
	s.mu.Lock()
	s.lines = s.lines[:0]
	s.mu.Unlock()
}

// pump reads r line by line into the sink until EOF. Meant to run in a goroutine
// over a subprocess stdout/stderr pipe.
func (s *LogSink) pump(r io.Reader, prefix string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		s.append(prefix + sc.Text())
	}
}
