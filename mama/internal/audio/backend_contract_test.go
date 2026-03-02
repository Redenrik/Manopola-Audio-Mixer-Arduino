package audio

import (
	"errors"
	"testing"

	"mama/internal/config"
)

type fakeVolumeController struct {
	curVolume int
	muted     bool

	setVolumeCalls []int
	muteCalls      int
	unmuteCalls    int

	getVolumeErr error
	setVolumeErr error
	getMutedErr  error
	muteErr      error
	unmuteErr    error
}

func (f *fakeVolumeController) GetVolume() (int, error) {
	if f.getVolumeErr != nil {
		return 0, f.getVolumeErr
	}
	return f.curVolume, nil
}

func (f *fakeVolumeController) SetVolume(v int) error {
	f.setVolumeCalls = append(f.setVolumeCalls, v)
	if f.setVolumeErr != nil {
		return f.setVolumeErr
	}
	f.curVolume = v
	return nil
}

func (f *fakeVolumeController) GetMuted() (bool, error) {
	if f.getMutedErr != nil {
		return false, f.getMutedErr
	}
	return f.muted, nil
}

func (f *fakeVolumeController) Mute() error {
	f.muteCalls++
	if f.muteErr != nil {
		return f.muteErr
	}
	f.muted = true
	return nil
}

func (f *fakeVolumeController) Unmute() error {
	f.unmuteCalls++
	if f.unmuteErr != nil {
		return f.unmuteErr
	}
	f.muted = false
	return nil
}

func TestBackendAdjustMasterOutContract(t *testing.T) {
	fake := &fakeVolumeController{curVolume: 50}
	b := &baseBackend{master: fake}

	if err := b.Adjust(config.TargetMasterOut, "", 0.02, 3); err != nil {
		t.Fatalf("Adjust() error = %v", err)
	}

	if len(fake.setVolumeCalls) != 1 || fake.setVolumeCalls[0] != 56 {
		t.Fatalf("SetVolume calls = %v, want [56]", fake.setVolumeCalls)
	}
}

func TestBackendAdjustMasterOutClampsContract(t *testing.T) {
	tests := []struct {
		name       string
		start      int
		step       float64
		delta      int
		wantVolume int
	}{
		{name: "upper bound", start: 98, step: 0.02, delta: 5, wantVolume: 100},
		{name: "lower bound", start: 1, step: 0.02, delta: -5, wantVolume: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeVolumeController{curVolume: tc.start}
			b := &baseBackend{master: fake}
			if err := b.Adjust(config.TargetMasterOut, "", tc.step, tc.delta); err != nil {
				t.Fatalf("Adjust() error = %v", err)
			}
			if got := fake.setVolumeCalls[0]; got != tc.wantVolume {
				t.Fatalf("SetVolume() = %d, want %d", got, tc.wantVolume)
			}
		})
	}
}

func TestBackendToggleMuteMasterOutContract(t *testing.T) {
	t.Run("mutes when unmuted", func(t *testing.T) {
		fake := &fakeVolumeController{muted: false}
		b := &baseBackend{master: fake}
		if err := b.ToggleMute(config.TargetMasterOut, ""); err != nil {
			t.Fatalf("ToggleMute() error = %v", err)
		}
		if fake.muteCalls != 1 || fake.unmuteCalls != 0 {
			t.Fatalf("mute/unmute calls = %d/%d, want 1/0", fake.muteCalls, fake.unmuteCalls)
		}
	})

	t.Run("unmutes when muted", func(t *testing.T) {
		fake := &fakeVolumeController{muted: true}
		b := &baseBackend{master: fake}
		if err := b.ToggleMute(config.TargetMasterOut, ""); err != nil {
			t.Fatalf("ToggleMute() error = %v", err)
		}
		if fake.muteCalls != 0 || fake.unmuteCalls != 1 {
			t.Fatalf("mute/unmute calls = %d/%d, want 0/1", fake.muteCalls, fake.unmuteCalls)
		}
	})
}

func TestBackendUnsupportedTargetsContract(t *testing.T) {
	fake := &fakeVolumeController{curVolume: 50}
	b := &baseBackend{master: fake}
	unsupported := []config.TargetType{config.TargetLineIn, config.TargetApp, config.TargetGroup}

	for _, target := range unsupported {
		t.Run(string(target), func(t *testing.T) {
			if err := b.Adjust(target, "", 0.02, 1); err == nil {
				t.Fatalf("Adjust(%s) expected unsupported error", target)
			}
			if err := b.ToggleMute(target, ""); err == nil {
				t.Fatalf("ToggleMute(%s) expected unsupported error", target)
			}
		})
	}
}

func TestBackendErrorPropagationContract(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("adjust get volume", func(t *testing.T) {
		b := &baseBackend{master: &fakeVolumeController{getVolumeErr: errBoom}}
		if err := b.Adjust(config.TargetMasterOut, "", 0.02, 1); !errors.Is(err, errBoom) {
			t.Fatalf("Adjust() error = %v, want %v", err, errBoom)
		}
	})

	t.Run("toggle mute get muted", func(t *testing.T) {
		b := &baseBackend{master: &fakeVolumeController{getMutedErr: errBoom}}
		if err := b.ToggleMute(config.TargetMasterOut, ""); !errors.Is(err, errBoom) {
			t.Fatalf("ToggleMute() error = %v, want %v", err, errBoom)
		}
	})
}

func TestBackendListTargetsContract(t *testing.T) {
	b := &baseBackend{master: &fakeVolumeController{}}
	targets, err := b.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets() error = %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("ListTargets() len = %d, want 1", len(targets))
	}
	if targets[0].ID != "system:master_out" || targets[0].Type != config.TargetMasterOut {
		t.Fatalf("ListTargets()[0] = %#v, want id=system:master_out type=master_out", targets[0])
	}
}

func TestBackendMicInContract(t *testing.T) {
	fake := &fakeVolumeController{curVolume: 40}
	b := &baseBackend{master: &fakeVolumeController{}, mic: fake}

	if err := b.Adjust(config.TargetMicIn, "", 0.05, 2); err != nil {
		t.Fatalf("Adjust(mic_in) error = %v", err)
	}
	if got := fake.setVolumeCalls[0]; got != 50 {
		t.Fatalf("SetVolume() = %d, want 50", got)
	}

	if err := b.ToggleMute(config.TargetMicIn, ""); err != nil {
		t.Fatalf("ToggleMute(mic_in) error = %v", err)
	}
	if fake.muteCalls != 1 {
		t.Fatalf("muteCalls = %d, want 1", fake.muteCalls)
	}
}

func TestBackendMicInUnsupportedWithoutController(t *testing.T) {
	b := &baseBackend{master: &fakeVolumeController{}, mic: nil}
	if err := b.Adjust(config.TargetMicIn, "", 0.1, 1); err == nil {
		t.Fatal("Adjust(mic_in) expected unsupported error")
	}
}

func TestBackendListTargetsIncludesMicWhenAvailable(t *testing.T) {
	b := &baseBackend{master: &fakeVolumeController{}, mic: &fakeVolumeController{}}
	targets, err := b.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("ListTargets() len = %d, want 2", len(targets))
	}
	if targets[1].Type != config.TargetMicIn || targets[1].ID != "system:mic_in" {
		t.Fatalf("ListTargets()[1] = %#v, want mic target", targets[1])
	}
}
