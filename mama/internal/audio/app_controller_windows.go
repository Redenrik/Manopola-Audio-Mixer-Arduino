//go:build windows

package audio

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca"
	"golang.org/x/sys/windows"
	"mama/internal/config"
)

type windowsAppSessionController struct{}

type windowsAppSession struct {
	id      string
	name    string
	exe     string
	volume  int
	isMuted bool
}

// go-wca v0.1.1 contains a malformed IID_IAudioSessionManager2 string; keep a local valid GUID.
var iidIAudioSessionManager2 = ole.NewGUID("{77AA99A0-1BD6-484F-8BC7-2C654C9A9B6F}")

func newAppSessionController() appSessionController {
	return &windowsAppSessionController{}
}

func (w *windowsAppSessionController) Adjust(selectorToken string, step float64, deltaSteps int) error {
	target, err := w.selectSession(selectorToken)
	if err != nil {
		return err
	}
	next := target.volume + int(step*100*float64(deltaSteps))
	if next < 0 {
		next = 0
	}
	if next > 100 {
		next = 100
	}
	return w.applyToSessionByID(target.id, func(volume *iSimpleAudioVolume) error {
		return volume.SetMasterVolume(float32(next)/100.0, nil)
	})
}

func (w *windowsAppSessionController) ToggleMute(selectorToken string) error {
	target, err := w.selectSession(selectorToken)
	if err != nil {
		return err
	}
	next := !target.isMuted
	return w.applyToSessionByID(target.id, func(volume *iSimpleAudioVolume) error {
		return volume.SetMute(next, nil)
	})
}

func (w *windowsAppSessionController) AdjustGroup(selectors []config.Selector, step float64, deltaSteps int) error {
	targets, err := w.selectSessions(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		next := target.volume + int(step*100*float64(deltaSteps))
		if next < 0 {
			next = 0
		}
		if next > 100 {
			next = 100
		}
		if err := w.applyToSessionByID(target.id, func(volume *iSimpleAudioVolume) error {
			return volume.SetMasterVolume(float32(next)/100.0, nil)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (w *windowsAppSessionController) ToggleMuteGroup(selectors []config.Selector) error {
	targets, err := w.selectSessions(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		next := !target.isMuted
		if err := w.applyToSessionByID(target.id, func(volume *iSimpleAudioVolume) error {
			return volume.SetMute(next, nil)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (w *windowsAppSessionController) ListTargets() ([]DiscoveredTarget, error) {
	sessions, err := w.listSessions()
	if err != nil {
		return nil, err
	}
	targets := make([]DiscoveredTarget, 0, len(sessions))
	for _, s := range sessions {
		name := s.name
		if name == "" {
			name = s.exe
		}
		targets = append(targets, DiscoveredTarget{ID: "app:" + s.id, Type: config.TargetApp, Name: name})
	}
	return targets, nil
}

func (w *windowsAppSessionController) selectSession(selectorToken string) (windowsAppSession, error) {
	sel := parseSelectorToken(selectorToken)
	sessions, err := w.listSessions()
	if err != nil {
		return windowsAppSession{}, err
	}
	for _, s := range sessions {
		if windowsSessionMatchesSelector(s, sel) {
			return s, nil
		}
	}
	return windowsAppSession{}, fmt.Errorf("%w: no app session matched selector %q", ErrTargetUnavailable, selectorToken)
}

func (w *windowsAppSessionController) selectSessions(selectors []config.Selector) ([]windowsAppSession, error) {
	sessions, err := w.listSessions()
	if err != nil {
		return nil, err
	}
	matched := make([]windowsAppSession, 0, len(sessions))
	seen := make(map[string]struct{}, len(sessions))
	for _, selector := range selectors {
		for _, s := range sessions {
			if !windowsSessionMatchesSelector(s, selector) {
				continue
			}
			if _, exists := seen[s.id]; exists {
				continue
			}
			seen[s.id] = struct{}{}
			matched = append(matched, s)
		}
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w: no app sessions matched group selectors", ErrTargetUnavailable)
	}
	return matched, nil
}

func (w *windowsAppSessionController) applyToSessionByID(id string, apply func(volume *iSimpleAudioVolume) error) error {
	return withWindowsSessions(func(sessions []windowsAppSessionHandle) error {
		for _, s := range sessions {
			if s.id != id {
				continue
			}
			return apply(s.volume)
		}
		return fmt.Errorf("audio session %q no longer available", id)
	})
}

func (w *windowsAppSessionController) listSessions() ([]windowsAppSession, error) {
	var sessions []windowsAppSession
	err := withWindowsSessions(func(handles []windowsAppSessionHandle) error {
		sessions = make([]windowsAppSession, 0, len(handles))
		for _, h := range handles {
			var level float32
			if err := h.volume.GetMasterVolume(&level); err != nil {
				continue
			}
			var muted bool
			if err := h.volume.GetMute(&muted); err != nil {
				continue
			}
			sessions = append(sessions, windowsAppSession{
				id:      h.id,
				name:    h.name,
				exe:     h.exe,
				volume:  int(math.Floor(float64(level*100.0 + 0.5))),
				isMuted: muted,
			})
		}
		return nil
	})
	return sessions, err
}

type windowsAppSessionHandle struct {
	id      string
	name    string
	exe     string
	control *iAudioSessionControl2
	volume  *iSimpleAudioVolume
}

func withWindowsSessions(fn func([]windowsAppSessionHandle) error) (err error) {
	return withCOMApartment(func() error {
		var mmde *wca.IMMDeviceEnumerator
		if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			return fmt.Errorf("create device enumerator: %w", err)
		}
		defer mmde.Release()

		var mmd *wca.IMMDevice
		if err = mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
			return fmt.Errorf("get default render endpoint: %w", err)
		}
		defer mmd.Release()

		var mgr *iAudioSessionManager2
		if err = activateIMMDevice(mmd, iidIAudioSessionManager2, wca.CLSCTX_ALL, &mgr); err != nil {
			return fmt.Errorf("activate IAudioSessionManager2: %w", err)
		}
		defer mgr.Release()

		var enum *iAudioSessionEnumerator
		if err = mgr.GetSessionEnumerator(&enum); err != nil {
			return fmt.Errorf("get audio session enumerator: %w", err)
		}
		defer enum.Release()

		count, err := enum.GetCount()
		if err != nil {
			return fmt.Errorf("get audio session count: %w", err)
		}

		sessions := make([]windowsAppSessionHandle, 0, count)
		for i := 0; i < count; i++ {
			ctrl, err := enum.GetSession(i)
			if err != nil {
				continue
			}
			if ctrl == nil {
				continue
			}
			defer ctrl.Release()

			ctrl2 := &iAudioSessionControl2{}
			if err := ctrl.PutQueryInterface(wca.IID_IAudioSessionControl2, &ctrl2); err != nil {
				continue
			}
			defer ctrl2.Release()

			pid, err := ctrl2.GetProcessID()
			if err != nil || pid == 0 {
				continue
			}

			simple := &iSimpleAudioVolume{}
			if err := ctrl.PutQueryInterface(wca.IID_ISimpleAudioVolume, &simple); err != nil {
				continue
			}
			defer simple.Release()

			exePath, exeName := processInfoFromPID(pid)
			sessionID := fmt.Sprintf("pid:%d:%d", pid, i)
			if exePath != "" {
				sessionID = exePath
			}
			sessions = append(sessions, windowsAppSessionHandle{
				id:      sessionID,
				name:    exeName,
				exe:     exeName,
				control: ctrl2,
				volume:  simple,
			})
		}

		return fn(sessions)
	})
}

func processInfoFromPID(pid uint32) (string, string) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", ""
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", ""
	}
	full := syscall.UTF16ToString(buf[:size])
	exe := strings.TrimSpace(filepath.Base(full))
	return strings.ToLower(full), strings.ToLower(exe)
}

func windowsSessionMatchesSelector(s windowsAppSession, selector config.Selector) bool {
	value := strings.TrimSpace(selector.Value)
	if value == "" {
		return false
	}
	value = strings.ToLower(value)

	target := strings.ToLower(strings.TrimSpace(s.name))
	if selector.Kind == config.SelectorExe {
		target = strings.ToLower(strings.TrimSpace(s.exe))
	}
	if target == "" {
		return false
	}

	switch selector.Kind {
	case config.SelectorExact, config.SelectorExe:
		return target == value
	case config.SelectorContains:
		return strings.Contains(target, value)
	case config.SelectorPrefix:
		return strings.HasPrefix(target, value)
	case config.SelectorSuffix:
		return strings.HasSuffix(target, value)
	case config.SelectorGlob:
		ok, err := filepath.Match(value, target)
		return err == nil && ok
	default:
		return false
	}
}

type iAudioSessionManager2 struct{ ole.IUnknown }

type iAudioSessionManager2Vtbl struct {
	ole.IUnknownVtbl
	GetAudioSessionControl        uintptr
	GetSimpleAudioVolume          uintptr
	GetSessionEnumerator          uintptr
	RegisterSessionNotification   uintptr
	UnregisterSessionNotification uintptr
	RegisterDuckNotification      uintptr
	UnregisterDuckNotification    uintptr
}

func (v *iAudioSessionManager2) VTable() *iAudioSessionManager2Vtbl {
	return (*iAudioSessionManager2Vtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iAudioSessionManager2) GetSessionEnumerator(sessionEnum **iAudioSessionEnumerator) error {
	hr, _, _ := syscall.Syscall(
		v.VTable().GetSessionEnumerator,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(sessionEnum)),
		0,
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

type iAudioSessionEnumerator struct{ ole.IUnknown }

type iAudioSessionEnumeratorVtbl struct {
	ole.IUnknownVtbl
	GetCount   uintptr
	GetSession uintptr
}

func (v *iAudioSessionEnumerator) VTable() *iAudioSessionEnumeratorVtbl {
	return (*iAudioSessionEnumeratorVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iAudioSessionEnumerator) GetCount() (int, error) {
	var count int32
	hr, _, _ := syscall.Syscall(
		v.VTable().GetCount,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&count)),
		0,
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return int(count), nil
}

func (v *iAudioSessionEnumerator) GetSession(index int) (*ole.IUnknown, error) {
	var session *ole.IUnknown
	hr, _, _ := syscall.Syscall(
		v.VTable().GetSession,
		3,
		uintptr(unsafe.Pointer(v)),
		uintptr(int32(index)),
		uintptr(unsafe.Pointer(&session)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return session, nil
}

type iAudioSessionControl2 struct{ ole.IUnknown }

type iAudioSessionControl2Vtbl struct {
	ole.IUnknownVtbl
	GetState                           uintptr
	GetDisplayName                     uintptr
	SetDisplayName                     uintptr
	GetIconPath                        uintptr
	SetIconPath                        uintptr
	GetGroupingParam                   uintptr
	SetGroupingParam                   uintptr
	RegisterAudioSessionNotification   uintptr
	UnregisterAudioSessionNotification uintptr
	GetSessionIdentifier               uintptr
	GetSessionInstanceIdentifier       uintptr
	GetProcessID                       uintptr
	IsSystemSoundsSession              uintptr
	SetDuckingPreference               uintptr
}

func (v *iAudioSessionControl2) VTable() *iAudioSessionControl2Vtbl {
	return (*iAudioSessionControl2Vtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iAudioSessionControl2) GetProcessID() (uint32, error) {
	var pid uint32
	hr, _, _ := syscall.Syscall(
		v.VTable().GetProcessID,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&pid)),
		0,
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return pid, nil
}

type iSimpleAudioVolume struct{ ole.IUnknown }

type iSimpleAudioVolumeVtbl struct {
	ole.IUnknownVtbl
	SetMasterVolume uintptr
	GetMasterVolume uintptr
	SetMute         uintptr
	GetMute         uintptr
}

func (v *iSimpleAudioVolume) VTable() *iSimpleAudioVolumeVtbl {
	return (*iSimpleAudioVolumeVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iSimpleAudioVolume) SetMasterVolume(level float32, eventContext *ole.GUID) error {
	hr, _, _ := syscall.Syscall(
		v.VTable().SetMasterVolume,
		3,
		uintptr(unsafe.Pointer(v)),
		uintptr(*(*uint32)(unsafe.Pointer(&level))),
		uintptr(unsafe.Pointer(eventContext)),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *iSimpleAudioVolume) GetMasterVolume(level *float32) error {
	hr, _, _ := syscall.Syscall(
		v.VTable().GetMasterVolume,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(level)),
		0,
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *iSimpleAudioVolume) SetMute(mute bool, eventContext *ole.GUID) error {
	var m uintptr
	if mute {
		m = 1
	}
	hr, _, _ := syscall.Syscall(
		v.VTable().SetMute,
		3,
		uintptr(unsafe.Pointer(v)),
		m,
		uintptr(unsafe.Pointer(eventContext)),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *iSimpleAudioVolume) GetMute(muted *bool) error {
	var m int32
	hr, _, _ := syscall.Syscall(
		v.VTable().GetMute,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&m)),
		0,
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	*muted = m != 0
	return nil
}
