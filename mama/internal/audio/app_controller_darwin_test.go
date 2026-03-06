//go:build darwin && cgo

package audio

import (
	"errors"
	"testing"

	"mama/internal/config"
)

type fakeCoreAudioBridge struct {
	processes []coreAudioProcessInfo

	volume          map[uint32]int
	mute            map[uint32]bool
	volumeSupported map[uint32]bool
	muteSupported   map[uint32]bool

	setVolumeCalls []uint32
	setMuteCalls   []uint32
}

func (f *fakeCoreAudioBridge) ListProcesses() ([]coreAudioProcessInfo, error) {
	out := make([]coreAudioProcessInfo, len(f.processes))
	copy(out, f.processes)
	return out, nil
}

func (f *fakeCoreAudioBridge) GetProcessVolume(processObjectID uint32) (int, bool, error) {
	return f.volume[processObjectID], f.volumeSupported[processObjectID], nil
}

func (f *fakeCoreAudioBridge) SetProcessVolume(processObjectID uint32, volume int) (bool, error) {
	if !f.volumeSupported[processObjectID] {
		return false, nil
	}
	f.volume[processObjectID] = volume
	f.setVolumeCalls = append(f.setVolumeCalls, processObjectID)
	return true, nil
}

func (f *fakeCoreAudioBridge) GetProcessMute(processObjectID uint32) (bool, bool, error) {
	return f.mute[processObjectID], f.muteSupported[processObjectID], nil
}

func (f *fakeCoreAudioBridge) SetProcessMute(processObjectID uint32, muted bool) (bool, error) {
	if !f.muteSupported[processObjectID] {
		return false, nil
	}
	f.mute[processObjectID] = muted
	f.setMuteCalls = append(f.setMuteCalls, processObjectID)
	return true, nil
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
		volume:          map[uint32]int{10: 40},
		mute:            map[uint32]bool{10: true},
		volumeSupported: map[uint32]bool{10: true},
		muteSupported:   map[uint32]bool{10: true},
	}
	controller := newDarwinAppSessionController(bridge)

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
	if got := bridge.volume[10]; got != 44 {
		t.Fatalf("volume=%d want 44", got)
	}
	if len(bridge.setMuteCalls) != 1 {
		t.Fatalf("setMuteCalls=%v want one unmute", bridge.setMuteCalls)
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
		volume:          map[uint32]int{20: 55},
		mute:            map[uint32]bool{20: false},
		volumeSupported: map[uint32]bool{20: false},
		muteSupported:   map[uint32]bool{20: true},
	}
	controller := newDarwinAppSessionController(bridge)

	err := controller.AdjustGroup([]config.Selector{
		{Kind: config.SelectorContains, Value: "discord"},
		{Kind: config.SelectorExe, Value: "discord"},
	}, 0.02, 1)
	if err == nil {
		t.Fatal("AdjustGroup expected unsupported error")
	}
	if !errors.Is(err, Unsupported(config.TargetGroup)) {
		// Unsupported returns a formatted error; compare message for now.
		if err.Error() != Unsupported(config.TargetGroup).Error() {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if len(bridge.setVolumeCalls) != 0 {
		t.Fatalf("setVolumeCalls=%v want none", bridge.setVolumeCalls)
	}
}
