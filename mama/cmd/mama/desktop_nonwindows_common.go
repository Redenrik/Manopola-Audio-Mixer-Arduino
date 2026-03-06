//go:build !windows

package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func waitDesktopURLReady(ctx context.Context, url string, timeout time.Duration) {
	base := strings.TrimSpace(url)
	if base == "" {
		return
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 350 * time.Millisecond}
	statusURL := strings.TrimRight(base, "/") + "/api/status"
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		if err != nil {
			return
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(120 * time.Millisecond):
		}
	}
}
