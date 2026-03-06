//go:build darwin && cgo

package audio

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mama/internal/config"
)

type darwinAppSession struct {
	id            string
	stateKey      string
	processObject uint32
	pid           int
	name          string
	bundleID      string
	exe           string
	exeNoExt      string
	execPath      string
	aliases       []string
	exeAliases    []string
	volume        int
	isMuted       bool
	volumeCapable bool
	muteCapable   bool
}

type darwinAppSessionController struct {
	bridge          coreAudioBridge
	tap             darwinTapBridge
	refreshInterval time.Duration

	mu       sync.Mutex
	cachedAt time.Time
	cached   []darwinAppSession
}

func newAppSessionController() appSessionController {
	return newDarwinAppSessionController(newCoreAudioBridge(), newDarwinTapBridge())
}

func newDarwinAppSessionController(bridge coreAudioBridge, tap darwinTapBridge) *darwinAppSessionController {
	if bridge == nil {
		bridge = newCoreAudioBridge()
	}
	if tap == nil {
		tap = newDarwinTapBridge()
	}
	return &darwinAppSessionController{
		bridge:          bridge,
		tap:             tap,
		refreshInterval: 250 * time.Millisecond,
	}
}

func (d *darwinAppSessionController) Adjust(selectorToken string, step float64, deltaSteps int) error {
	target, err := d.selectSession(selectorToken)
	if err != nil {
		return err
	}
	if !target.volumeCapable {
		return Unsupported(config.TargetApp)
	}
	if err := d.tap.EnsureTap(target.tapTarget()); err != nil {
		return d.mapWriteError(target.id, err)
	}

	next := target.volume + int(step*100*float64(deltaSteps))
	next = clampPercent(next)

	if err := d.tap.SetGain(target.stateKey, next); err != nil {
		return d.mapWriteError(target.id, err)
	}

	updatedMuted := target.isMuted
	if deltaSteps > 0 && target.isMuted && target.muteCapable {
		if err := d.tap.SetMuted(target.stateKey, false); err == nil {
			updatedMuted = false
		}
	}

	d.rememberWriteState(target.stateKey, &next, &updatedMuted)
	return nil
}

func (d *darwinAppSessionController) ToggleMute(selectorToken string) error {
	target, err := d.selectSession(selectorToken)
	if err != nil {
		return err
	}
	if !target.muteCapable {
		return Unsupported(config.TargetApp)
	}
	if err := d.tap.EnsureTap(target.tapTarget()); err != nil {
		return d.mapWriteError(target.id, err)
	}

	next := !target.isMuted
	if err := d.tap.SetMuted(target.stateKey, next); err != nil {
		return d.mapWriteError(target.id, err)
	}

	d.rememberWriteState(target.stateKey, nil, &next)
	return nil
}

func (d *darwinAppSessionController) AdjustGroup(selectors []config.Selector, step float64, deltaSteps int) error {
	targets, err := d.selectSessions(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if !target.volumeCapable {
			return Unsupported(config.TargetGroup)
		}
		if err := d.tap.EnsureTap(target.tapTarget()); err != nil {
			return d.mapWriteError(target.id, err)
		}

		next := clampPercent(target.volume + int(step*100*float64(deltaSteps)))
		if err := d.tap.SetGain(target.stateKey, next); err != nil {
			return d.mapWriteError(target.id, err)
		}

		updatedMuted := target.isMuted
		if deltaSteps > 0 && target.isMuted && target.muteCapable {
			if err := d.tap.SetMuted(target.stateKey, false); err == nil {
				updatedMuted = false
			}
		}
		d.rememberWriteState(target.stateKey, &next, &updatedMuted)
	}
	return nil
}

func (d *darwinAppSessionController) ToggleMuteGroup(selectors []config.Selector) error {
	targets, err := d.selectSessions(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if !target.muteCapable {
			return Unsupported(config.TargetGroup)
		}
		if err := d.tap.EnsureTap(target.tapTarget()); err != nil {
			return d.mapWriteError(target.id, err)
		}

		next := !target.isMuted
		if err := d.tap.SetMuted(target.stateKey, next); err != nil {
			return d.mapWriteError(target.id, err)
		}
		d.rememberWriteState(target.stateKey, nil, &next)
	}
	return nil
}

func (d *darwinAppSessionController) ReadState(selectorToken string) (TargetState, error) {
	target, err := d.selectSession(selectorToken)
	if err != nil {
		return TargetState{}, err
	}
	return TargetState{
		Available:    true,
		Volume:       target.volume,
		Muted:        target.isMuted,
		SessionCount: 1,
	}, nil
}

func (d *darwinAppSessionController) ReadGroupState(selectors []config.Selector) (TargetState, error) {
	targets, err := d.selectSessions(selectors)
	if err != nil {
		return TargetState{}, err
	}

	totalVolume := 0
	allMuted := true
	for _, target := range targets {
		totalVolume += target.volume
		if !target.isMuted {
			allMuted = false
		}
	}

	avg := int(math.Round(float64(totalVolume) / float64(len(targets))))
	avg = clampPercent(avg)

	return TargetState{
		Available:    true,
		Volume:       avg,
		Muted:        allMuted,
		SessionCount: len(targets),
	}, nil
}

func (d *darwinAppSessionController) ListTargets() ([]DiscoveredTarget, error) {
	sessions, err := d.listSessions(false)
	if err != nil {
		return nil, err
	}

	targets := make([]DiscoveredTarget, 0, len(sessions))
	for _, session := range sessions {
		name := strings.TrimSpace(session.name)
		if name == "" {
			name = strings.TrimSpace(session.exe)
		}
		if name == "" {
			name = strings.TrimSpace(session.bundleID)
		}
		selector := name
		if selector == "" {
			selector = session.id
		}

		target := DiscoveredTarget{
			ID:       "app:" + session.id,
			Type:     config.TargetApp,
			Name:     name,
			Selector: selector,
			Capabilities: &TargetCapabilities{
				Volume: boolPtr(session.volumeCapable),
				Mute:   boolPtr(session.muteCapable),
			},
		}
		if len(session.aliases) > 0 {
			target.Aliases = append(target.Aliases, session.aliases...)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func (d *darwinAppSessionController) selectSession(selectorToken string) (darwinAppSession, error) {
	sel := parseSelectorToken(selectorToken)
	sessions, err := d.listSessions(true)
	if err != nil {
		return darwinAppSession{}, err
	}
	for _, session := range sessions {
		if darwinSessionMatchesSelector(session, sel) {
			return session, nil
		}
	}
	return darwinAppSession{}, fmt.Errorf("%w: no app session matched selector %q", ErrTargetUnavailable, selectorToken)
}

func (d *darwinAppSessionController) selectSessions(selectors []config.Selector) ([]darwinAppSession, error) {
	sessions, err := d.listSessions(true)
	if err != nil {
		return nil, err
	}
	matched := make([]darwinAppSession, 0, len(sessions))
	seen := make(map[string]struct{}, len(sessions))
	for _, selector := range selectors {
		for _, session := range sessions {
			if !darwinSessionMatchesSelector(session, selector) {
				continue
			}
			if _, exists := seen[session.id]; exists {
				continue
			}
			seen[session.id] = struct{}{}
			matched = append(matched, session)
		}
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w: no app sessions matched group selectors", ErrTargetUnavailable)
	}
	return matched, nil
}

func (d *darwinAppSessionController) listSessions(forceRefresh bool) ([]darwinAppSession, error) {
	d.mu.Lock()
	if !forceRefresh && !d.cachedAt.IsZero() && time.Since(d.cachedAt) < d.refreshInterval {
		cached := cloneDarwinSessions(d.cached)
		d.mu.Unlock()
		return cached, nil
	}
	d.mu.Unlock()

	fetched, err := d.fetchSessions()
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.cached = cloneDarwinSessions(fetched)
	d.cachedAt = time.Now()
	cached := cloneDarwinSessions(d.cached)
	d.mu.Unlock()
	return cached, nil
}

func (d *darwinAppSessionController) fetchSessions() ([]darwinAppSession, error) {
	processes, err := d.bridge.ListProcesses()
	if err != nil {
		return nil, err
	}

	sessions := make([]darwinAppSession, 0, len(processes))
	tapTargets := make([]darwinTapTarget, 0, len(processes))
	for _, process := range processes {
		session, ok := buildDarwinSession(process)
		if !ok {
			continue
		}
		sessions = append(sessions, session)
		tapTargets = append(tapTargets, session.tapTarget())
	}

	supported := d.tap != nil && d.tap.Supported()
	if supported {
		_ = d.tap.Reconcile(tapTargets)
	}

	for i := range sessions {
		state := darwinTapState{Volume: 100}
		if supported {
			state = d.tap.ReadVirtualState(sessions[i].stateKey)
		}
		sessions[i].volume = clampPercent(state.Volume)
		sessions[i].isMuted = state.Muted
		sessions[i].volumeCapable = supported
		sessions[i].muteCapable = supported
	}

	return sessions, nil
}

func (d *darwinAppSessionController) invalidateCache() {
	d.mu.Lock()
	d.cached = nil
	d.cachedAt = time.Time{}
	d.mu.Unlock()
}

func (d *darwinAppSessionController) mapWriteError(sessionID string, err error) error {
	sessions, listErr := d.listSessions(true)
	if listErr != nil {
		return err
	}
	for _, session := range sessions {
		if session.id == sessionID {
			return err
		}
	}
	return fmt.Errorf("%w: app session %q no longer available", ErrTargetUnavailable, sessionID)
}

func (d *darwinAppSessionController) rememberWriteState(stateKey string, volume *int, muted *bool) {
	if stateKey == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	updated := false
	for i := range d.cached {
		if d.cached[i].stateKey != stateKey {
			continue
		}
		if volume != nil {
			d.cached[i].volume = clampPercent(*volume)
		}
		if muted != nil {
			d.cached[i].isMuted = *muted
		}
		updated = true
	}
	if updated {
		d.cachedAt = time.Now()
	}
}

func buildDarwinSession(process coreAudioProcessInfo) (darwinAppSession, bool) {
	if process.ObjectID == 0 || process.PID <= 0 || !process.RunningOutput {
		return darwinAppSession{}, false
	}

	execPath := strings.TrimSpace(process.ExecutablePath)
	exe := strings.TrimSpace(filepath.Base(execPath))
	exeNoExt := strings.TrimSpace(strings.TrimSuffix(exe, filepath.Ext(exe)))
	bundleID := strings.TrimSpace(process.BundleID)
	name := deriveDarwinProcessName(exeNoExt, exe, bundleID)
	aliases := normalizeMatchValues(name, bundleID, exe, exeNoExt, execPath)
	exeAliases := normalizeMatchValues(exe, exeNoExt, bundleID)
	if len(aliases) == 0 && len(exeAliases) > 0 {
		aliases = append(aliases, exeAliases...)
	}

	return darwinAppSession{
		id:            fmt.Sprintf("%d", process.ObjectID),
		stateKey:      darwinTapStateKey(bundleID, execPath, exeNoExt, name, process.ObjectID),
		processObject: process.ObjectID,
		pid:           process.PID,
		name:          name,
		bundleID:      bundleID,
		exe:           exe,
		exeNoExt:      exeNoExt,
		execPath:      execPath,
		aliases:       aliases,
		exeAliases:    exeAliases,
	}, true
}

func (s darwinAppSession) tapTarget() darwinTapTarget {
	return darwinTapTarget{
		Key:           s.stateKey,
		ProcessObject: s.processObject,
		BundleID:      s.bundleID,
		Name:          s.name,
	}
}

func darwinTapStateKey(bundleID, execPath, exeNoExt, name string, processObject uint32) string {
	if value := strings.TrimSpace(bundleID); value != "" {
		return "bundle:" + strings.ToLower(value)
	}
	if value := strings.TrimSpace(execPath); value != "" {
		return "path:" + strings.ToLower(value)
	}
	if value := strings.TrimSpace(exeNoExt); value != "" {
		return "exe:" + strings.ToLower(value)
	}
	if value := strings.TrimSpace(name); value != "" {
		return "name:" + strings.ToLower(value)
	}
	return fmt.Sprintf("process:%d", processObject)
}

func deriveDarwinProcessName(exeNoExt, exe, bundleID string) string {
	if exeNoExt != "" {
		return exeNoExt
	}
	if exe != "" {
		return exe
	}
	if bundleID == "" {
		return ""
	}
	parts := strings.Split(bundleID, ".")
	if len(parts) == 0 {
		return bundleID
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func cloneDarwinSessions(in []darwinAppSession) []darwinAppSession {
	if len(in) == 0 {
		return nil
	}
	out := make([]darwinAppSession, len(in))
	copy(out, in)
	for i := range out {
		if len(out[i].aliases) > 0 {
			out[i].aliases = append([]string(nil), out[i].aliases...)
		}
		if len(out[i].exeAliases) > 0 {
			out[i].exeAliases = append([]string(nil), out[i].exeAliases...)
		}
	}
	return out
}

func darwinSessionMatchesSelector(session darwinAppSession, selector config.Selector) bool {
	if selector.Kind == config.SelectorExe {
		return selectorMatchesAnyValue(selector, session.exeAliases)
	}
	return selectorMatchesAnyValue(selector, session.aliases)
}
