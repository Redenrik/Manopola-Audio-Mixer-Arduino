//go:build darwin && cgo

package audio

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mama/internal/config"
)

type darwinAppSession struct {
	id            string
	runtimeKey    string
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

type darwinLogicalApp struct {
	id            string
	stateKey      string
	name          string
	bundleID      string
	aliases       []string
	exeAliases    []string
	sessions      []darwinAppSession
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
	tap := newDarwinTapBridge()
	if tap == nil || !tap.Supported() {
		return nil
	}
	return newDarwinAppSessionController(newCoreAudioBridge(), tap)
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
	target, err := d.selectLogicalApp(selectorToken)
	if err != nil {
		return err
	}
	if !target.volumeCapable {
		return Unsupported(config.TargetApp)
	}
	for _, session := range target.sessions {
		if err := d.tap.EnsureTap(session.tapTarget()); err != nil {
			return d.mapWriteError(target.stateKey, err)
		}
	}

	next := clampPercent(target.volume + int(step*100*float64(deltaSteps)))
	if err := d.tap.SetGain(target.stateKey, next); err != nil {
		return d.mapWriteError(target.stateKey, err)
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
	target, err := d.selectLogicalApp(selectorToken)
	if err != nil {
		return err
	}
	if !target.muteCapable {
		return Unsupported(config.TargetApp)
	}
	for _, session := range target.sessions {
		if err := d.tap.EnsureTap(session.tapTarget()); err != nil {
			return d.mapWriteError(target.stateKey, err)
		}
	}

	next := !target.isMuted
	if err := d.tap.SetMuted(target.stateKey, next); err != nil {
		return d.mapWriteError(target.stateKey, err)
	}

	d.rememberWriteState(target.stateKey, nil, &next)
	return nil
}

func (d *darwinAppSessionController) AdjustGroup(selectors []config.Selector, step float64, deltaSteps int) error {
	targets, err := d.selectLogicalApps(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if !target.volumeCapable {
			return Unsupported(config.TargetGroup)
		}
		for _, session := range target.sessions {
			if err := d.tap.EnsureTap(session.tapTarget()); err != nil {
				return d.mapWriteError(target.stateKey, err)
			}
		}

		next := clampPercent(target.volume + int(step*100*float64(deltaSteps)))
		if err := d.tap.SetGain(target.stateKey, next); err != nil {
			return d.mapWriteError(target.stateKey, err)
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
	targets, err := d.selectLogicalApps(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if !target.muteCapable {
			return Unsupported(config.TargetGroup)
		}
		for _, session := range target.sessions {
			if err := d.tap.EnsureTap(session.tapTarget()); err != nil {
				return d.mapWriteError(target.stateKey, err)
			}
		}

		next := !target.isMuted
		if err := d.tap.SetMuted(target.stateKey, next); err != nil {
			return d.mapWriteError(target.stateKey, err)
		}
		d.rememberWriteState(target.stateKey, nil, &next)
	}
	return nil
}

func (d *darwinAppSessionController) ReadState(selectorToken string) (TargetState, error) {
	if d.tap == nil || !d.tap.Supported() {
		return TargetState{}, Unsupported(config.TargetApp)
	}
	target, err := d.selectLogicalApp(selectorToken)
	if err != nil {
		return TargetState{}, err
	}
	for _, session := range target.sessions {
		if err := d.tap.EnsureTap(session.tapTarget()); err != nil {
			return TargetState{}, d.mapWriteError(target.stateKey, err)
		}
	}
	state := d.tap.ReadVirtualState(target.stateKey)
	return TargetState{
		Available:    state.Attached,
		Volume:       clampPercent(state.Volume),
		Muted:        state.Muted,
		SessionCount: 1,
	}, nil
}

func (d *darwinAppSessionController) ReadGroupState(selectors []config.Selector) (TargetState, error) {
	if d.tap == nil || !d.tap.Supported() {
		return TargetState{}, Unsupported(config.TargetGroup)
	}
	targets, err := d.selectLogicalApps(selectors)
	if err != nil {
		return TargetState{}, err
	}

	totalVolume := 0
	allMuted := true
	allAttached := true
	for _, target := range targets {
		for _, session := range target.sessions {
			if err := d.tap.EnsureTap(session.tapTarget()); err != nil {
				return TargetState{}, d.mapWriteError(target.stateKey, err)
			}
		}
		state := d.tap.ReadVirtualState(target.stateKey)
		totalVolume += clampPercent(state.Volume)
		if !state.Muted {
			allMuted = false
		}
		if !state.Attached {
			allAttached = false
		}
	}

	avg := int(math.Round(float64(totalVolume) / float64(len(targets))))
	avg = clampPercent(avg)

	return TargetState{
		Available:    allAttached,
		Volume:       avg,
		Muted:        allMuted,
		SessionCount: len(targets),
	}, nil
}

func (d *darwinAppSessionController) ListTargets() ([]DiscoveredTarget, error) {
	apps, err := d.listLogicalApps(false)
	if err != nil {
		return nil, err
	}

	targets := make([]DiscoveredTarget, 0, len(apps))
	for _, app := range apps {
		name := strings.TrimSpace(app.name)
		if name == "" && len(app.sessions) > 0 {
			name = strings.TrimSpace(app.sessions[0].name)
		}
		if name == "" {
			name = strings.TrimSpace(app.bundleID)
		}
		selector := name
		if selector == "" {
			selector = app.id
		}

		target := DiscoveredTarget{
			ID:       "app:" + app.id,
			Type:     config.TargetApp,
			Name:     name,
			Selector: selector,
			Capabilities: &TargetCapabilities{
				Volume: boolPtr(app.volumeCapable),
				Mute:   boolPtr(app.muteCapable),
			},
		}
		if len(app.aliases) > 0 {
			target.Aliases = append(target.Aliases, app.aliases...)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func (d *darwinAppSessionController) selectLogicalApp(selectorToken string) (darwinLogicalApp, error) {
	sel := parseSelectorToken(selectorToken)
	if app, ok, err := d.findLogicalApp(sel, false); err != nil {
		return darwinLogicalApp{}, err
	} else if ok {
		return app, nil
	}
	if app, ok, err := d.findLogicalApp(sel, true); err != nil {
		return darwinLogicalApp{}, err
	} else if ok {
		return app, nil
	}
	return darwinLogicalApp{}, fmt.Errorf("%w: no app session matched selector %q", ErrTargetUnavailable, selectorToken)
}

func (d *darwinAppSessionController) selectLogicalApps(selectors []config.Selector) ([]darwinLogicalApp, error) {
	matched, err := d.findLogicalApps(selectors, false)
	if err != nil {
		return nil, err
	}
	if len(matched) > 0 {
		return matched, nil
	}
	matched, err = d.findLogicalApps(selectors, true)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w: no app sessions matched group selectors", ErrTargetUnavailable)
	}
	return matched, nil
}

func (d *darwinAppSessionController) listLogicalApps(forceRefresh bool) ([]darwinLogicalApp, error) {
	sessions, err := d.listSessions(forceRefresh)
	if err != nil {
		return nil, err
	}
	return buildDarwinLogicalApps(sessions, d.tap), nil
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

func (d *darwinAppSessionController) mapWriteError(stateKey string, err error) error {
	apps, listErr := d.listLogicalApps(true)
	if listErr != nil {
		return err
	}
	for _, app := range apps {
		if app.stateKey == stateKey {
			return err
		}
	}
	return fmt.Errorf("%w: app session %q no longer available", ErrTargetUnavailable, stateKey)
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

func (d *darwinAppSessionController) findLogicalApp(selector config.Selector, forceRefresh bool) (darwinLogicalApp, bool, error) {
	apps, err := d.listLogicalApps(forceRefresh)
	if err != nil {
		return darwinLogicalApp{}, false, err
	}
	for _, app := range apps {
		if darwinLogicalAppMatchesSelector(app, selector) {
			return app, true, nil
		}
	}
	return darwinLogicalApp{}, false, nil
}

func (d *darwinAppSessionController) findLogicalApps(selectors []config.Selector, forceRefresh bool) ([]darwinLogicalApp, error) {
	apps, err := d.listLogicalApps(forceRefresh)
	if err != nil {
		return nil, err
	}
	matched := make([]darwinLogicalApp, 0, len(apps))
	seen := make(map[string]struct{}, len(apps))
	for _, selector := range selectors {
		for _, app := range apps {
			if !darwinLogicalAppMatchesSelector(app, selector) {
				continue
			}
			if _, exists := seen[app.stateKey]; exists {
				continue
			}
			seen[app.stateKey] = struct{}{}
			matched = append(matched, app)
		}
	}
	return matched, nil
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
		runtimeKey:    darwinTapRuntimeKey(process.ObjectID),
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
		StateKey:      s.stateKey,
		RuntimeKey:    s.runtimeKey,
		ProcessObject: s.processObject,
		BundleID:      s.bundleID,
		Name:          s.name,
	}
}

func darwinTapRuntimeKey(processObject uint32) string {
	return fmt.Sprintf("process:%d", processObject)
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

func buildDarwinLogicalApps(sessions []darwinAppSession, tap darwinTapBridge) []darwinLogicalApp {
	if len(sessions) == 0 {
		return nil
	}
	byKey := make(map[string]*darwinLogicalApp, len(sessions))
	order := make([]string, 0, len(sessions))
	for _, session := range sessions {
		app := byKey[session.stateKey]
		if app == nil {
			app = &darwinLogicalApp{
				id:         session.stateKey,
				stateKey:   session.stateKey,
				name:       session.name,
				bundleID:   session.bundleID,
				aliases:    append([]string(nil), session.aliases...),
				exeAliases: append([]string(nil), session.exeAliases...),
			}
			byKey[session.stateKey] = app
			order = append(order, session.stateKey)
		} else {
			if app.name == "" {
				app.name = session.name
			}
			if app.bundleID == "" {
				app.bundleID = session.bundleID
			}
			app.aliases = appendUniqueStrings(app.aliases, session.aliases...)
			app.exeAliases = appendUniqueStrings(app.exeAliases, session.exeAliases...)
		}
		app.sessions = append(app.sessions, session)
	}

	apps := make([]darwinLogicalApp, 0, len(order))
	for _, key := range order {
		app := *byKey[key]
		sort.Slice(app.sessions, func(i, j int) bool {
			return app.sessions[i].processObject < app.sessions[j].processObject
		})
		supported := tap != nil && tap.Supported()
		state := darwinTapState{Volume: 100}
		if supported {
			state = tap.ReadVirtualState(app.stateKey)
		}
		app.volume = clampPercent(state.Volume)
		app.isMuted = state.Muted
		app.volumeCapable = supported
		app.muteCapable = supported
		apps = append(apps, app)
	}
	return apps
}

func appendUniqueStrings(base []string, values ...string) []string {
	if len(values) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base))
	for _, value := range base {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		base = append(base, value)
	}
	return base
}

func darwinLogicalAppMatchesSelector(app darwinLogicalApp, selector config.Selector) bool {
	if selector.Kind == config.SelectorExe {
		return selectorMatchesAnyValue(selector, app.exeAliases)
	}
	return selectorMatchesAnyValue(selector, app.aliases)
}
