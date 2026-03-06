//go:build !windows

package main

import (
	"os/signal"
	"syscall"
)

func ignoreTerminalHangupSignal() {
	signal.Ignore(syscall.SIGHUP)
}
