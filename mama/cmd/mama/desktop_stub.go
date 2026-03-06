//go:build !windows && (!(darwin || linux) || !cgo)

package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

func defaultDesktopMode() bool {
	return true
}

func runDesktopShell(ctx context.Context, _ context.CancelFunc, url string, openBrowser bool, startHidden bool) error {
	shouldOpen := openBrowser && !startHidden
	if shouldOpen {
		go func() {
			waitDesktopURLReady(ctx, url, 5*time.Second)
			openURL(url)
		}()
	}

	if startHidden {
		fmt.Printf("Desktop shell is running hidden on %s (native tray is not available in this build).\n", runtime.GOOS)
	} else if !shouldOpen {
		fmt.Println("Desktop shell started without opening the browser UI (use -open=true to auto-open).")
	}

	<-ctx.Done()
	return nil
}
