//go:build darwin && cgo

package audio

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"mama/internal/config"
)

type fakeCoreAudioBridge struct {
	processes []coreAudioProcessInfo
}

func (f *fakeCoreAudioBridge) ListProcesses() ([]coreAudioProcessInfo, error) {
	out := make([]coreAudioProcessInfo, len(f.processes))
	copy(out, f.processes)
	return out, nil
}

type fakeDarwinTapBridge struct {
	supported bool
	states    map[string]darwinTapState
	ensureErr map[string]error

	liveTargets   map[string]darwinTapTarget
	ensureCalls   []darwinTapTarget
	setGainCalls  map[string]int
	setMutedCalls map[string]bool
}

func (f *fakeDarwinTapBridge) Supported() bool {
	return f.supported
}

func (f *fakeDarwinTapBridge) EnsureTap(target darwinTapTarget) error {
	if f.liveTargets == nil {
		f.liveTargets = map[string]darwinTapTarget{}
	}
	f.ensureCalls = append(f.ensureCalls, target)
	f.liveTargets[target.RuntimeKey] = target
	if err := f.ensureErr[target.RuntimeKey]; err != nil {
		state := f.ReadVirtualState(target.StateKey)
		state.Wanted = true
		state.Attached = false
		f.saveState(target.StateKey, state)
		return err
	}
	state := f.ReadVirtualState(target.StateKey)
	state.Wanted = true
	state.Attached = true
	f.saveState(target.StateKey, state)
	return nil
}

func (f *fakeDarwinTapBridge) ReleaseTap(stateKey string) error {
	for runtimeKey, target := range f.liveTargets {
		if target.StateKey == stateKey {
			delete(f.liveTargets, runtimeKey)
		}
	}
	state := f.ReadVirtualState(stateKey)
	state.Attached = false
	state.Wanted = false
	f.saveState(stateKey, state)
	return nil
}

func (f *fakeDarwinTapBridge) SetGain(stateKey string, volume int) error {
	if f.setGainCalls == nil {
		f.setGainCalls = map[string]int{}
	}
	state := f.ReadVirtualState(stateKey)
	state.Volume = clampPercent(volume)
	state.Attached = true
	state.Wanted = true
	f.setGainCalls[stateKey] = state.Volume
	f.saveState(stateKey, state)
	return nil
}

func (f *fakeDarwinTapBridge) SetMuted(stateKey string, muted bool) error {
	if f.setMutedCalls == nil {
		f.setMutedCalls = map[string]bool{}
	}
	state := f.ReadVirtualState(stateKey)
	state.Muted = muted
	state.Attached = true
	state.Wanted = true
	f.setMutedCalls[stateKey] = muted
	f.saveState(stateKey, state)
	return nil
}

func (f *fakeDarwinTapBridge) ReadVirtualState(stateKey string) darwinTapState {
	if f.states == nil {
		f.states = map[string]darwinTapState{}
	}
	if state, ok := f.states[stateKey]; ok {
		return state
	}
	return darwinTapState{Volume: 100}
}

func (f *fakeDarwinTapBridge) Reconcile(targets []darwinTapTarget) error {
	if f.liveTargets == nil {
		f.liveTargets = map[string]darwinTapTarget{}
	}
	f.liveTargets = map[string]darwinTapTarget{}
	for _, target := range targets {
		f.liveTargets[target.RuntimeKey] = target
	}
	return nil
}

func (f *fakeDarwinTapBridge) Stats(stateKey string) (darwinTapStats, error) {
	state := f.ReadVirtualState(stateKey)
	if !state.Attached {
		return darwinTapStats{}, errors.New("not attached")
	}
	return darwinTapStats{
		Attached:        true,
		Callbacks:       1,
		Frames:          1024,
		RenderCallbacks: 1,
		RenderFrames:    1024,
		Channels:        2,
		BitsPerChannel:  32,
		BytesPerFrame:   8,
		SampleRate:      48000,
	}, nil
}

func (f *fakeDarwinTapBridge) saveState(stateKey string, state darwinTapState) {
	if f.states == nil {
		f.states = map[string]darwinTapState{}
	}
	f.states[stateKey] = state
}

func sortedRuntimeKeys(targets []darwinTapTarget) []string {
	keys := make([]string, 0, len(targets))
	for _, target := range targets {
		keys = append(keys, target.RuntimeKey)
	}
	sort.Strings(keys)
	return keys
}

func TestDarwinAppControllerListTargetsAggregatesLogicalAppsAndAdjusts(t *testing.T) {
	bridge := &fakeCoreAudioBridge{
		processes: []coreAudioProcessInfo{
			{
				ObjectID:       10,
				PID:            101,
				RunningOutput:  true,
				BundleID:       "com.spotify.client",
				ExecutablePath: "/Applications/Spotify.app/Contents/MacOS/Spotify",
			},
			{
				ObjectID:       11,
				PID:            102,
				RunningOutput:  true,
				BundleID:       "com.spotify.client",
				ExecutablePath: "/Applications/Spotify.app/Contents/MacOS/Spotify",
			},
		},
	}
	tap := &fakeDarwinTapBridge{
		supported: true,
		states: map[string]darwinTapState{
			"bundle:com.spotify.client": {Volume: 40, Muted: true},
		},
	}
	controller := newDarwinAppSessionController(bridge, tap)

	targets, err := controller.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("len(targets)=%d want 1", len(targets))
	}
	if targets[0].Name != "Spotify" {
		t.Fatalf("target name=%q want Spotify", targets[0].Name)
	}

	if err := controller.Adjust("exe:spotify", 0.02, 2); err != nil {
		t.Fatalf("Adjust error: %v", err)
	}

	const stateKey = "bundle:com.spotify.client"
	if got := tap.setGainCalls[stateKey]; got != 44 {
		t.Fatalf("gain=%d want 44", got)
	}
	if got, ok := tap.setMutedCalls[stateKey]; !ok || got {
		t.Fatalf("setMutedCalls[%q]=%v,%t want false,true", stateKey, got, ok)
	}
	if got := len(tap.ensureCalls); got != 2 {
		t.Fatalf("ensureCalls=%d want 2", got)
	}
	if got := sortedRuntimeKeys(tap.ensureCalls); fmt.Sprint(got) != "[process:10 process:11]" {
		t.Fatalf("ensure runtime keys=%v want [process:10 process:11]", got)
	}
}

func TestDarwinAppControllerReadStateEnsuresAllProcessesOfLogicalApp(t *testing.T) {
	bridge := &fakeCoreAudioBridge{
		processes: []coreAudioProcessInfo{
			{
				ObjectID:       11,
				PID:            101,
				RunningOutput:  true,
				BundleID:       "com.spotify.client",
				ExecutablePath: "/Applications/Spotify.app/Contents/MacOS/Spotify",
			},
			{
				ObjectID:       12,
				PID:            102,
				RunningOutput:  true,
				BundleID:       "com.spotify.client",
				ExecutablePath: "/Applications/Spotify.app/Contents/MacOS/Spotify",
			},
		},
	}
	tap := &fakeDarwinTapBridge{
		supported: true,
		states: map[string]darwinTapState{
			"bundle:com.spotify.client": {Volume: 57, Muted: true},
		},
	}
	controller := newDarwinAppSessionController(bridge, tap)

	state, err := controller.ReadState("exe:spotify")
	if err != nil {
		t.Fatalf("ReadState error: %v", err)
	}
	if len(tap.ensureCalls) != 2 {
		t.Fatalf("ensureCalls=%d want 2", len(tap.ensureCalls))
	}
	if !state.Available || state.Volume != 57 || !state.Muted || state.SessionCount != 1 {
		t.Fatalf("state=%+v want available=true volume=57 muted=true sessions=1", state)
	}
}

func TestDarwinAppControllerGroupDedupAndUnsupported(t *testing.T) {
	bridge := &fakeCoreAudioBridge{
		processes: []coreAudioProcessInfo{
			{
				ObjectID:       20,
				PID:            202,
				RunningOutput:  true,
				BundleID:       "com.discord.Discord",
				ExecutablePath: "/Applications/Discord.app/Contents/MacOS/Discord",
			},
			{
				ObjectID:       21,
				PID:            203,
				RunningOutput:  true,
				BundleID:       "com.discord.Discord",
				ExecutablePath: "/Applications/Discord.app/Contents/MacOS/Discord",
			},
		},
	}
	tap := &fakeDarwinTapBridge{supported: false}
	controller := newDarwinAppSessionController(bridge, tap)

	err := controller.AdjustGroup([]config.Selector{
		{Kind: config.SelectorContains, Value: "discord"},
		{Kind: config.SelectorExe, Value: "discord"},
	}, 0.02, 1)
	if err == nil {
		t.Fatal("AdjustGroup expected unsupported error")
	}
	if !errors.Is(err, Unsupported(config.TargetGroup)) && err.Error() != Unsupported(config.TargetGroup).Error() {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tap.ensureCalls) != 0 {
		t.Fatalf("ensureCalls=%v want none", tap.ensureCalls)
	}
}

func TestDarwinAppControllerReadStateReturnsPermissionError(t *testing.T) {
	bridge := &fakeCoreAudioBridge{
		processes: []coreAudioProcessInfo{
			{
				ObjectID:       12,
				PID:            202,
				RunningOutput:  true,
				BundleID:       "com.discord.Discord",
				ExecutablePath: "/Applications/Discord.app/Contents/MacOS/Discord",
			},
		},
	}
	tap := &fakeDarwinTapBridge{
		supported: true,
		ensureErr: map[string]error{
			"process:12": fmt.Errorf("%w: create process tap", ErrPermissionDenied),
		},
	}
	controller := newDarwinAppSessionController(bridge, tap)

	_, err := controller.ReadState("exe:discord")
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied error, got %v", err)
	}
}
