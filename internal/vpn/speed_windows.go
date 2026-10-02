//go:build windows

package vpn

import (
	"TomorrowClient/internal/model"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SpeedResult struct {
	DownloadMbps float64    `json:"downloadMbps"`
	Bytes        int64      `json:"bytes"`
	DurationMs   int64      `json:"durationMs"`
	Core         model.Core `json:"core"`
}

// DownloadSpeed uses a private instance of the selected core, NOT the system's
// default route. Bounded to 25 MB and 15 seconds; no adapter/routing changes.
func DownloadSpeed(p model.Profile, c model.Core) (SpeedResult, error) {
	return downloadSpeed(p, c, "https://speed.cloudflare.com/__down?bytes=25000000", 15*time.Second)
}

func downloadSpeed(p model.Profile, c model.Core, target string, timeout time.Duration) (SpeedResult, error) {
	result := SpeedResult{Core: c}
	_, err := probeTransport(p, c, ProbeOptions{Timeout: timeout}, func(tr *http.Transport, o ProbeOptions) (time.Duration, error) {
		defer tr.CloseIdleConnections()
		ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
		if err != nil {
			return 0, err
		}
		client := &http.Client{Transport: tr, Timeout: o.Timeout}
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return 0, fmt.Errorf("speed test HTTP %d", resp.StatusCode)
		}
		n, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 25000000))
		elapsed := time.Since(start)
		// A deadline after receiving data ends the timed sample; other I/O errors fail.
		if err != nil && ctx.Err() == nil {
			return 0, err
		}
		if n < 65536 {
			return 0, fmt.Errorf("слишком мало данных для измерения: %d байт", n)
		}
		result.Bytes = n
		result.DurationMs = elapsed.Milliseconds()
		result.DownloadMbps = float64(n) * 8 / elapsed.Seconds() / 1e6
		return elapsed, nil
	})
	return result, err
}
