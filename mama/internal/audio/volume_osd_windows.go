//go:build windows

package audio

import (
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"
)

const (
	wmAppCommand         = 0x0319
	appCommandVolumeMute = 8
	appCommandVolumeDown = 9
	appCommandVolumeUp   = 10
	smtoAbortIfHung      = 0x0002
	appCommandTimeoutMS  = 80
	flyoutMinIntervalMS  = 120
)

var (
	lastFlyoutAtUnixMilli         int64
	user32ProcGetShellWindow      = windows.NewLazySystemDLL("user32.dll").NewProc("GetShellWindow")
	user32ProcSendMessageTimeoutW = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
)

func showWindowsVolumeFlyoutForAdjust(master volumeController) {
	if master == nil {
		return
	}
	now := time.Now().UnixMilli()
	if now-atomic.LoadInt64(&lastFlyoutAtUnixMilli) < flyoutMinIntervalMS {
		return
	}
	atomic.StoreInt64(&lastFlyoutAtUnixMilli, now)

	// Force OSD with a neutral volume-key sequence, then restore exact state.
	// Without restore, single-detent changes can be canceled by the synthetic up/down pair.
	expectedVolume, volumeErr := getVolume(master, "")
	expectedMuted, mutedErr := getMuted(master, "")

	if volumeErr == nil && expectedVolume >= 100 {
		sendVolumeAppCommand(appCommandVolumeDown)
		sendVolumeAppCommand(appCommandVolumeUp)
	} else {
		sendVolumeAppCommand(appCommandVolumeUp)
		sendVolumeAppCommand(appCommandVolumeDown)
	}
	// Let WM_APPCOMMAND events settle before we read/restore.
	time.Sleep(35 * time.Millisecond)

	if volumeErr == nil {
		if actualVolume, err := getVolume(master, ""); err == nil && actualVolume != expectedVolume {
			_ = setVolume(master, "", expectedVolume)
		}
	}
	if mutedErr == nil {
		if actualMuted, err := getMuted(master, ""); err == nil && actualMuted != expectedMuted {
			if expectedMuted {
				_ = mute(master, "")
			} else {
				_ = unmute(master, "")
			}
		}
	}
}

func showWindowsVolumeFlyoutForMuteToggle(master volumeController) {
	if master == nil {
		return
	}
	now := time.Now().UnixMilli()
	if now-atomic.LoadInt64(&lastFlyoutAtUnixMilli) < flyoutMinIntervalMS {
		return
	}
	atomic.StoreInt64(&lastFlyoutAtUnixMilli, now)

	// Toggle mute twice to show OSD while preserving the already-applied mute state.
	// This avoids volume up/down commands that can accidentally unmute.
	expectedMuted, expectedErr := getMuted(master, "")
	sendVolumeAppCommand(appCommandVolumeMute)
	sendVolumeAppCommand(appCommandVolumeMute)

	if expectedErr != nil {
		return
	}
	actualMuted, err := getMuted(master, "")
	if err != nil || actualMuted == expectedMuted {
		return
	}
	if expectedMuted {
		_ = mute(master, "")
	} else {
		_ = unmute(master, "")
	}
}

func sendVolumeAppCommand(command uintptr) {
	hwnd, _, _ := user32ProcGetShellWindow.Call()
	if hwnd == 0 {
		return
	}
	lParam := command << 16
	_, _, _ = user32ProcSendMessageTimeoutW.Call(
		hwnd,
		uintptr(wmAppCommand),
		hwnd,
		lParam,
		uintptr(smtoAbortIfHung),
		uintptr(appCommandTimeoutMS),
		0,
	)
}
