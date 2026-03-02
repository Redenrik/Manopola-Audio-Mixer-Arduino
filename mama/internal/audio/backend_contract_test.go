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

type fakeAppSessionController struct {
	adjustCalls      []string
	toggleCalls      []string
	adjustGroupCalls [][]config.Selector
	toggleGroupCalls [][]config.Selector
	targets          []DiscoveredTarget
	adjustErr        error
	toggleErr        error
	listErr          error
}

func (f *fakeAppSessionController) Adjust(selectorToken string, step float64, deltaSteps int) error {
	f.adjustCalls = append(f.adjustCalls, selectorToken)
	return f.adjustErr
}

func (f *fakeAppSessionController) ToggleMute(selectorToken string) error {
	f.toggleCalls = append(f.toggleCalls, selectorToken)
	return f.toggleErr
}

func (f *fakeAppSessionController) ListTargets() ([]DiscoveredTarget, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.targets, nil
}

func (f *fakeAppSessionController) AdjustGroup(selectors []config.Selector, step float64, deltaSteps int) error {
	dup := append([]config.Selector(nil), selectors...)
	f.adjustGroupCalls = append(f.adjustGroupCalls, dup)
	return f.adjustErr
}

func (f *fakeAppSessionController) ToggleMuteGroup(selectors []config.Selector) error {
	dup := append([]config.Selector(nil), selectors...)
	f.toggleGroupCalls = append(f.toggleGroupCalls, dup)
	return f.toggleErr
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
	unsupported := []config.TargetType{}

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

func TestBackendListTargetsIncludesMicAndLineInWhenAvailable(t *testing.T) {
	b := &baseBackend{master: &fakeVolumeController{}, mic: &fakeVolumeController{}, lineIn: &fakeVolumeController{}}
	targets, err := b.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets() error = %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("ListTargets() len = %d, want 3", len(targets))
	}
	if targets[1].Type != config.TargetMicIn || targets[1].ID != "system:mic_in" {
		t.Fatalf("ListTargets()[1] = %#v, want mic target", targets[1])
	}
	if targets[2].Type != config.TargetLineIn || targets[2].ID != "system:line_in" {
		t.Fatalf("ListTargets()[2] = %#v, want line-in target", targets[2])
	}
}

func TestBackendLineInContract(t *testing.T) {
	fake := &fakeVolumeController{curVolume: 30}
	b := &baseBackend{master: &fakeVolumeController{}, lineIn: fake}

	if err := b.Adjust(config.TargetLineIn, "", 0.10, 3); err != nil {
		t.Fatalf("Adjust(line_in) error = %v", err)
	}
	if got := fake.setVolumeCalls[0]; got != 60 {
		t.Fatalf("SetVolume() = %d, want 60", got)
	}

	if err := b.ToggleMute(config.TargetLineIn, ""); err != nil {
		t.Fatalf("ToggleMute(line_in) error = %v", err)
	}
	if fake.muteCalls != 1 {
		t.Fatalf("muteCalls = %d, want 1", fake.muteCalls)
	}
}

func TestBackendLineInUnsupportedWithoutController(t *testing.T) {
	b := &baseBackend{master: &fakeVolumeController{}, lineIn: nil}
	if err := b.Adjust(config.TargetLineIn, "", 0.1, 1); err == nil {
		t.Fatal("Adjust(line_in) expected unsupported error")
	}
}

func TestBackendAppContract(t *testing.T) {
	app := &fakeAppSessionController{}
	b := &baseBackend{master: &fakeVolumeController{}, app: app}

	if err := b.Adjust(config.TargetApp, "exact:Discord", 0.02, 1); err != nil {
		t.Fatalf("Adjust(app) error = %v", err)
	}
	if len(app.adjustCalls) != 1 || app.adjustCalls[0] != "exact:Discord" {
		t.Fatalf("adjustCalls = %v, want [exact:Discord]", app.adjustCalls)
	}

	if err := b.ToggleMute(config.TargetApp, "exact:Discord"); err != nil {
		t.Fatalf("ToggleMute(app) error = %v", err)
	}
	if len(app.toggleCalls) != 1 || app.toggleCalls[0] != "exact:Discord" {
		t.Fatalf("toggleCalls = %v, want [exact:Discord]", app.toggleCalls)
	}
}

func TestBackendListTargetsIncludesAppTargetsWhenAvailable(t *testing.T) {
	app := &fakeAppSessionController{targets: []DiscoveredTarget{{ID: "app:77", Type: config.TargetApp, Name: "Discord"}}}
	b := &baseBackend{master: &fakeVolumeController{}, app: app}
	targets, err := b.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("ListTargets() len = %d, want 2", len(targets))
	}
	if targets[1].Type != config.TargetApp || targets[1].ID != "app:77" {
		t.Fatalf("ListTargets()[1] = %#v, want app target", targets[1])
	}
}

func TestBackendAppUnsupportedWithoutController(t *testing.T) {
	b := &baseBackend{master: &fakeVolumeController{}, app: nil}
	if err := b.Adjust(config.TargetApp, "exact:Discord", 0.02, 1); err == nil {
		t.Fatal("Adjust(app) expected unsupported error")
	}
}

func TestBackendGroupContract(t *testing.T) {
	app := &fakeAppSessionController{}
	b := &baseBackend{master: &fakeVolumeController{}, app: app}

	token := `[{"kind":"exact","value":"Discord"},{"kind":"exe","value":"spotify"}]`
	if err := b.Adjust(config.TargetGroup, token, 0.02, 1); err != nil {
		t.Fatalf("Adjust(group) error = %v", err)
	}
	if len(app.adjustGroupCalls) != 1 || len(app.adjustGroupCalls[0]) != 2 {
		t.Fatalf("adjustGroupCalls = %#v, want one call with two selectors", app.adjustGroupCalls)
	}

	if err := b.ToggleMute(config.TargetGroup, token); err != nil {
		t.Fatalf("ToggleMute(group) error = %v", err)
	}
	if len(app.toggleGroupCalls) != 1 || len(app.toggleGroupCalls[0]) != 2 {
		t.Fatalf("toggleGroupCalls = %#v, want one call with two selectors", app.toggleGroupCalls)
	}
}

func TestBackendGroupUnsupportedWithoutController(t *testing.T) {
	b := &baseBackend{master: &fakeVolumeController{}, app: nil}
	if err := b.Adjust(config.TargetGroup, `[{"kind":"exact","value":"Discord"}]`, 0.02, 1); err == nil {
		t.Fatal("Adjust(group) expected unsupported error")
	}
}

func TestParseSelectorToken(t *testing.T) {
	sel := parseSelectorToken("exe:discord")
	if sel.Kind != config.SelectorExe || sel.Value != "discord" {
		t.Fatalf("unexpected selector: %#v", sel)
	}
	fallback := parseSelectorToken("Discord")
	if fallback.Kind != config.SelectorExact || fallback.Value != "Discord" {
		t.Fatalf("unexpected fallback selector: %#v", fallback)
	}
}

func TestParseGroupSelectorsToken(t *testing.T) {
	selectors, err := parseGroupSelectorsToken(`[{"kind":"exact","value":"Discord"}]`)
	if err != nil {
		t.Fatalf("parseGroupSelectorsToken() error = %v", err)
	}
	if len(selectors) != 1 || selectors[0].Kind != config.SelectorExact {
		t.Fatalf("unexpected selectors: %#v", selectors)
	}
}
