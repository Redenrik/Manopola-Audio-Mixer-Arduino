//go:build darwin && cgo

package audio

import (
	"errors"
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

	ensureCalls   []darwinTapTarget
	setGainCalls  map[string]int
	setMutedCalls map[string]bool
}

func (f *fakeDarwinTapBridge) Supported() bool {
	return f.supported
}

func (f *fakeDarwinTapBridge) EnsureTap(target darwinTapTarget) error {
	f.ensureCalls = append(f.ensureCalls, target)
	state := f.ReadVirtualState(target.Key)
	state.Wanted = true
	state.Attached = true
	f.saveState(target.Key, state)
	return nil
}

func (f *fakeDarwinTapBridge) ReleaseTap(sessionKey string) error {
	state := f.ReadVirtualState(sessionKey)
	state.Attached = false
	state.Wanted = false
	f.saveState(sessionKey, state)
	return nil
}

func (f *fakeDarwinTapBridge) SetGain(sessionKey string, volume int) error {
	if f.setGainCalls == nil {
		f.setGainCalls = map[string]int{}
	}
	state := f.ReadVirtualState(sessionKey)
	state.Volume = clampPercent(volume)
	state.Attached = true
	state.Wanted = true
	f.setGainCalls[sessionKey] = state.Volume
	f.saveState(sessionKey, state)
	return nil
}

func (f *fakeDarwinTapBridge) SetMuted(sessionKey string, muted bool) error {
	if f.setMutedCalls == nil {
		f.setMutedCalls = map[string]bool{}
	}
	state := f.ReadVirtualState(sessionKey)
	state.Muted = muted
	state.Attached = true
	state.Wanted = true
	f.setMutedCalls[sessionKey] = muted
	f.saveState(sessionKey, state)
	return nil
}

func (f *fakeDarwinTapBridge) ReadVirtualState(sessionKey string) darwinTapState {
	if f.states == nil {
		f.states = map[string]darwinTapState{}
	}
	if state, ok := f.states[sessionKey]; ok {
		return state
	}
	return darwinTapState{Volume: 100}
}

func (f *fakeDarwinTapBridge) Reconcile(targets []darwinTapTarget) error {
	return nil
}

func (f *fakeDarwinTapBridge) Stats(sessionKey string) (darwinTapStats, error) {
	state := f.ReadVirtualState(sessionKey)
	if !state.Attached {
		return darwinTapStats{}, errors.New("not attached")
	}
	return darwinTapStats{Attached: true, Callbacks: 1, Frames: 1024}, nil
}

func (f *fakeDarwinTapBridge) saveState(sessionKey string, state darwinTapState) {
	if f.states == nil {
		f.states = map[string]darwinTapState{}
	}
	f.states[sessionKey] = state
}

func TestDarwinAppControllerSelectAndAdjust(t *testing.T) {
	bridge := &fakeCoreAudioBridge{
		processes: []coreAudioProcessInfo{{
			ObjectID:       10,
			PID:            101,
			RunningOutput:  true,
			BundleID:       "com.spotify.client",
			ExecutablePath: "/Applications/Spotify.app/Contents/MacOS/Spotify",
		}},
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
	if targets[0].Capabilities == nil || targets[0].Capabilities.Volume == nil || targets[0].Capabilities.Mute == nil {
		t.Fatalf("target capabilities missing: %+v", targets[0].Capabilities)
	}
	if !*targets[0].Capabilities.Volume || !*targets[0].Capabilities.Mute {
		t.Fatalf("target capabilities=%+v want both true", targets[0].Capabilities)
	}

	if err := controller.Adjust("exe:spotify", 0.02, 2); err != nil {
		t.Fatalf("Adjust error: %v", err)
	}

	const key = "bundle:com.spotify.client"
	if got := tap.setGainCalls[key]; got != 44 {
		t.Fatalf("gain=%d want 44", got)
	}
	if got, ok := tap.setMutedCalls[key]; !ok || got {
		t.Fatalf("setMutedCalls[%q]=%v,%t want false,true", key, got, ok)
	}
	if got := len(tap.ensureCalls); got != 1 {
		t.Fatalf("ensureCalls=%d want 1", got)
	}
}

func TestDarwinAppControllerGroupDedupAndUnsupported(t *testing.T) {
	bridge := &fakeCoreAudioBridge{
		processes: []coreAudioProcessInfo{{
			ObjectID:       20,
			PID:            202,
			RunningOutput:  true,
			BundleID:       "com.discord.Discord",
			ExecutablePath: "/Applications/Discord.app/Contents/MacOS/Discord",
		}},
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
	if !errors.Is(err, Unsupported(config.TargetGroup)) {
		if err.Error() != Unsupported(config.TargetGroup).Error() {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if len(tap.ensureCalls) != 0 {
		t.Fatalf("ensureCalls=%v want none", tap.ensureCalls)
	}
}
