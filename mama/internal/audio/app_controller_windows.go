//go:build windows

package audio

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca"
	"golang.org/x/sys/windows"
	"mama/internal/config"
)

type windowsAppSessionController struct{}

type windowsAppSession struct {
	id         string
	name       string
	exe        string
	exeNoExt   string
	path       string
	aliases    []string
	exeAliases []string
	volume     int
	isMuted    bool
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
		target := DiscoveredTarget{
			ID:       "app:" + s.id,
			Type:     config.TargetApp,
			Name:     name,
			Selector: name,
		}
		if len(s.aliases) > 0 {
			target.Aliases = append(target.Aliases, s.aliases...)
		}
		targets = append(targets, target)
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
	found := false
	err := forEachWindowsSession(func(session windowsAppSession, volume *iSimpleAudioVolume) (bool, error) {
		if session.id != id {
			return false, nil
		}
		found = true
		if err := apply(volume); err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("audio session %q no longer available", id)
	}
	return nil
}

func (w *windowsAppSessionController) listSessions() ([]windowsAppSession, error) {
	sessions := make([]windowsAppSession, 0, 16)
	err := forEachWindowsSession(func(session windowsAppSession, volume *iSimpleAudioVolume) (bool, error) {
		var level float32
		if err := volume.GetMasterVolume(&level); err != nil {
			return false, nil
		}
		var muted bool
		if err := volume.GetMute(&muted); err != nil {
			return false, nil
		}
		session.volume = int(math.Floor(float64(level*100.0 + 0.5)))
		session.isMuted = muted
		sessions = append(sessions, session)
		return false, nil
	})
	return sessions, err
}

func forEachWindowsSession(fn func(session windowsAppSession, volume *iSimpleAudioVolume) (bool, error)) (err error) {
	return withCOMApartment(func() error {
		var mmde *wca.IMMDeviceEnumerator
		if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			return fmt.Errorf("create device enumerator: %w", err)
		}
		defer mmde.Release()

		var devices *wca.IMMDeviceCollection
		if err := mmde.EnumAudioEndpoints(wca.ERender, wca.DEVICE_STATE_ACTIVE, &devices); err != nil {
			return fmt.Errorf("enumerate render endpoints: %w", err)
		}
		defer devices.Release()

		var endpointCount uint32
		if err := devices.GetCount(&endpointCount); err != nil {
			return fmt.Errorf("get render endpoint count: %w", err)
		}

		seen := make(map[string]struct{}, endpointCount*4)

		for i := uint32(0); i < endpointCount; i++ {
			var device *wca.IMMDevice
			if err := devices.Item(i, &device); err != nil || device == nil {
				continue
			}

			endpointID, err := endpointID(device)
			if err != nil {
				device.Release()
				continue
			}

			stop, err := forEachEndpointSession(device, endpointID, seen, fn)
			device.Release()
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}

		return nil
	})
}

func forEachEndpointSession(device *wca.IMMDevice, endpointID string, seen map[string]struct{}, fn func(session windowsAppSession, volume *iSimpleAudioVolume) (bool, error)) (bool, error) {
	var mgr *iAudioSessionManager2
	if err := activateIMMDevice(device, iidIAudioSessionManager2, wca.CLSCTX_ALL, &mgr); err != nil {
		return false, fmt.Errorf("activate IAudioSessionManager2: %w", err)
	}
	defer mgr.Release()

	var enum *iAudioSessionEnumerator
	if err := mgr.GetSessionEnumerator(&enum); err != nil {
		return false, fmt.Errorf("get audio session enumerator: %w", err)
	}
	defer enum.Release()

	count, err := enum.GetCount()
	if err != nil {
		return false, fmt.Errorf("get audio session count: %w", err)
	}

	for i := 0; i < count; i++ {
		ctrl, err := enum.GetSession(i)
		if err != nil || ctrl == nil {
			continue
		}

		ctrl2 := &iAudioSessionControl2{}
		if err := ctrl.PutQueryInterface(wca.IID_IAudioSessionControl2, &ctrl2); err != nil || ctrl2 == nil {
			ctrl.Release()
			continue
		}

		simple := &iSimpleAudioVolume{}
		if err := ctrl.PutQueryInterface(wca.IID_ISimpleAudioVolume, &simple); err != nil || simple == nil {
			ctrl2.Release()
			ctrl.Release()
			continue
		}

		pid, err := ctrl2.GetProcessID()
		if err != nil || pid == 0 {
			simple.Release()
			ctrl2.Release()
			ctrl.Release()
			continue
		}

		session := makeWindowsAppSession(ctrl2, pid, endpointID, i)
		key := windowsSessionDedupKey(session)
		if key == "" {
			simple.Release()
			ctrl2.Release()
			ctrl.Release()
			continue
		}
		if _, exists := seen[key]; exists {
			simple.Release()
			ctrl2.Release()
			ctrl.Release()
			continue
		}
		seen[key] = struct{}{}

		stop, callbackErr := fn(session, simple)
		simple.Release()
		ctrl2.Release()
		ctrl.Release()
		if callbackErr != nil {
			return false, callbackErr
		}
		if stop {
			return true, nil
		}
	}

	return false, nil
}

func makeWindowsAppSession(ctrl2 *iAudioSessionControl2, pid uint32, endpointID string, index int) windowsAppSession {
	processInfo := processInfoFromPID(pid)
	displayName, _ := ctrl2.GetDisplayName()
	displayName = strings.TrimSpace(displayName)
	sessionID, _ := ctrl2.GetSessionInstanceIdentifier()
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		if fallback, err := ctrl2.GetSessionIdentifier(); err == nil {
			sessionID = strings.TrimSpace(fallback)
		}
	}
	if sessionID == "" {
		if processInfo.path != "" {
			sessionID = processInfo.path + "#" + endpointID
		} else {
			sessionID = fmt.Sprintf("pid:%d:%s:%d", pid, endpointID, index)
		}
	}

	name := resolveSessionName(displayName, processInfo)
	aliases := normalizeMatchValues(displayName, processInfo.displayName, processInfo.exe, processInfo.exeNoExt, processInfo.path, name)
	exeAliases := normalizeMatchValues(processInfo.exe, processInfo.exeNoExt)
	if len(aliases) == 0 && len(exeAliases) > 0 {
		aliases = append(aliases, exeAliases...)
	}

	return windowsAppSession{
		id:         sessionID,
		name:       name,
		exe:        processInfo.exe,
		exeNoExt:   processInfo.exeNoExt,
		path:       processInfo.path,
		aliases:    aliases,
		exeAliases: exeAliases,
	}
}

func windowsSessionDedupKey(session windowsAppSession) string {
	for _, value := range []string{session.id, session.path, session.exe} {
		key := strings.ToLower(strings.TrimSpace(value))
		if key != "" {
			return key
		}
	}
	return ""
}

type windowsProcessInfo struct {
	path        string
	exe         string
	exeNoExt    string
	displayName string
}

var processInfoCache sync.Map

func processInfoFromPID(pid uint32) windowsProcessInfo {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return windowsProcessInfo{}
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return windowsProcessInfo{}
	}

	fullPath := strings.TrimSpace(syscall.UTF16ToString(buf[:size]))
	if fullPath == "" {
		return windowsProcessInfo{}
	}

	cacheKey := strings.ToLower(fullPath)
	if cached, ok := processInfoCache.Load(cacheKey); ok {
		if info, ok := cached.(windowsProcessInfo); ok {
			return info
		}
	}

	exeName := strings.TrimSpace(filepath.Base(fullPath))
	exeLower := strings.ToLower(exeName)
	info := windowsProcessInfo{
		path:     strings.ToLower(fullPath),
		exe:      exeLower,
		exeNoExt: strings.TrimSuffix(exeLower, strings.ToLower(filepath.Ext(exeLower))),
	}

	description := strings.TrimSpace(fileDescriptionFromExecutable(fullPath))
	if description != "" {
		info.displayName = description
	} else {
		info.displayName = strings.TrimSpace(strings.TrimSuffix(exeName, filepath.Ext(exeName)))
	}

	processInfoCache.Store(cacheKey, info)
	return info
}

func resolveSessionName(displayName string, processInfo windowsProcessInfo) string {
	if name := strings.TrimSpace(displayName); name != "" {
		return name
	}
	if name := strings.TrimSpace(processInfo.displayName); name != "" {
		return name
	}
	if name := strings.TrimSpace(processInfo.exeNoExt); name != "" {
		return name
	}
	return strings.TrimSpace(processInfo.exe)
}

type langCodePage struct {
	Language uint16
	CodePage uint16
}

func fileDescriptionFromExecutable(path string) string {
	var zero windows.Handle
	infoSize, err := windows.GetFileVersionInfoSize(path, &zero)
	if err != nil || infoSize == 0 {
		return ""
	}

	versionInfo := make([]byte, infoSize)
	if err := windows.GetFileVersionInfo(path, 0, infoSize, unsafe.Pointer(&versionInfo[0])); err != nil {
		return ""
	}

	translations := queryFileTranslations(versionInfo)
	for _, entry := range translations {
		block := fmt.Sprintf(`\StringFileInfo\%04x%04x\FileDescription`, entry.Language, entry.CodePage)
		if value := queryVersionString(versionInfo, block); value != "" {
			return value
		}
	}

	for _, fallback := range []string{`\StringFileInfo\040904b0\FileDescription`, `\StringFileInfo\040904e4\FileDescription`} {
		if value := queryVersionString(versionInfo, fallback); value != "" {
			return value
		}
	}
	return ""
}

func queryFileTranslations(versionInfo []byte) []langCodePage {
	if len(versionInfo) == 0 {
		return nil
	}

	var ptr unsafe.Pointer
	var length uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&versionInfo[0]), `\VarFileInfo\Translation`, unsafe.Pointer(&ptr), &length); err != nil {
		return nil
	}
	if ptr == nil || length < 4 {
		return nil
	}
	count := int(length / 4)
	return unsafe.Slice((*langCodePage)(ptr), count)
}

func queryVersionString(versionInfo []byte, block string) string {
	if len(versionInfo) == 0 {
		return ""
	}

	var valuePtr *uint16
	var length uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&versionInfo[0]), block, unsafe.Pointer(&valuePtr), &length); err != nil {
		return ""
	}
	if valuePtr == nil || length == 0 {
		return ""
	}
	return strings.TrimSpace(windows.UTF16PtrToString(valuePtr))
}

func windowsSessionMatchesSelector(s windowsAppSession, selector config.Selector) bool {
	aliases := s.aliases
	exeAliases := s.exeAliases
	if len(aliases) == 0 || len(exeAliases) == 0 {
		if len(aliases) == 0 {
			aliases = normalizeMatchValues(s.name, s.exe, s.exeNoExt, s.path)
		}
		if len(exeAliases) == 0 {
			exeAliases = normalizeMatchValues(s.exe, s.exeNoExt)
		}
	}

	if selector.Kind == config.SelectorExe {
		return selectorMatchesAnyValue(selector, exeAliases)
	}
	return selectorMatchesAnyValue(selector, aliases)
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

func (v *iAudioSessionControl2) GetDisplayName() (string, error) {
	return callCOMStringMethod(v, v.VTable().GetDisplayName)
}

func (v *iAudioSessionControl2) GetSessionIdentifier() (string, error) {
	return callCOMStringMethod(v, v.VTable().GetSessionIdentifier)
}

func (v *iAudioSessionControl2) GetSessionInstanceIdentifier() (string, error) {
	return callCOMStringMethod(v, v.VTable().GetSessionInstanceIdentifier)
}

func callCOMStringMethod(v *iAudioSessionControl2, method uintptr) (string, error) {
	var str *uint16
	hr, _, _ := syscall.Syscall(
		method,
		2,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&str)),
		0,
	)
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	if str == nil {
		return "", nil
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(str)))
	return strings.TrimSpace(windows.UTF16PtrToString(str)), nil
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
