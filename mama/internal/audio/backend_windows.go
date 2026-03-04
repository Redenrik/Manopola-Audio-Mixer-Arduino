//go:build windows

package audio

import (
	"strings"

	"mama/internal/config"
)

type windowsBackend struct{ baseBackend }

func newBackend() Backend {
	return &windowsBackend{
		baseBackend: baseBackend{master: systemVolumeController{}, mic: newMicVolumeController(), lineIn: newLineInVolumeController(), app: newAppSessionController()},
	}
}

func (b *windowsBackend) String() string { return "windowsBackend" }

func (b *windowsBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
	if err := b.baseBackend.Adjust(target, name, step, deltaSteps); err != nil {
		return err
	}
	if target == config.TargetMasterOut && strings.TrimSpace(name) == "" {
		showWindowsVolumeFlyoutForAdjust(b.baseBackend.master)
	}
	return nil
}

func (b *windowsBackend) ToggleMute(target config.TargetType, name string) error {
	if err := b.baseBackend.ToggleMute(target, name); err != nil {
		return err
	}
	if target == config.TargetMasterOut && strings.TrimSpace(name) == "" {
		showWindowsVolumeFlyoutForMuteToggle(b.baseBackend.master)
	}
	return nil
}

func (b *windowsBackend) ListTargets() ([]DiscoveredTarget, error) {
	targets, err := b.baseBackend.ListTargets()
	if err != nil {
		return nil, err
	}

	extra, extraErr := listWindowsEndpointTargets()
	if extraErr == nil {
		targets = append(targets, extra...)
	}

	if extraErr != nil && len(targets) == 0 {
		return nil, extraErr
	}

	return dedupeTargets(targets), nil
}

func dedupeTargets(targets []DiscoveredTarget) []DiscoveredTarget {
	out := make([]DiscoveredTarget, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		key := string(target.Type) + "|" + target.ID
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	return out
}
