//go:build (darwin || linux) && cgo

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

func defaultDesktopMode() bool {
	return true
}

func runDesktopShell(ctx context.Context, cancel context.CancelFunc, url string, openBrowser bool, startHidden bool) (runErr error) {
	shouldOpen := openBrowser && !startHidden
	if startHidden {
		fmt.Printf("Desktop shell started hidden in tray on %s.\n", runtime.GOOS)
	} else if !shouldOpen {
		fmt.Println("Desktop shell started without opening the browser UI (use -open=true to auto-open).")
	}

	var (
		quitOnce sync.Once
		showItem *systray.MenuItem
		quitItem *systray.MenuItem
	)
	requestQuit := func() {
		quitOnce.Do(func() {
			systray.Quit()
		})
	}

	onReady := func() {
		systray.SetTitle("MAMA")
		systray.SetTooltip("MAMA Audio Mixer")
		setTrayIconIfAvailable()

		showItem = systray.AddMenuItem("Show MAMA", "Open setup and mixer UI")
		systray.AddSeparator()
		quitItem = systray.AddMenuItem("Quit MAMA", "Stop MAMA")

		if shouldOpen {
			go func() {
				waitDesktopURLReady(ctx, url, 5*time.Second)
				openURL(url)
			}()
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					requestQuit()
					return
				case <-showItem.ClickedCh:
					go func() {
						waitDesktopURLReady(ctx, url, 2*time.Second)
						openURL(url)
					}()
				case <-quitItem.ClickedCh:
					cancel()
					requestQuit()
					return
				}
			}
		}()
	}

	onExit := func() {}

	defer func() {
		if recovered := recover(); recovered != nil {
			runErr = fmt.Errorf("desktop tray panic: %v", recovered)
		}
	}()

	systray.Run(onReady, onExit)
	return nil
}

func setTrayIconIfAvailable() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	baseDir := filepath.Dir(exePath)
	candidates := []string{
		"mama-tray.png",
		"mama-app.png",
	}
	for _, name := range candidates {
		path := filepath.Join(baseDir, name)
		icon, err := os.ReadFile(path)
		if err != nil || len(icon) == 0 {
			continue
		}
		systray.SetIcon(icon)
		if runtime.GOOS == "darwin" {
			systray.SetTemplateIcon(icon, icon)
		}
		return
	}
}
