//go:build !windows

package main

import (
	"context"
	"fmt"
)

func defaultDesktopMode() bool {
	return false
}

func runDesktopShell(_ context.Context, _ context.CancelFunc, _ string, _ bool, _ bool) error {
	return fmt.Errorf("desktop mode is supported only on Windows")
}
