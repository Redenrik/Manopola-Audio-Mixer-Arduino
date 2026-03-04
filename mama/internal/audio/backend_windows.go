//go:build windows

package audio

type windowsBackend struct{ baseBackend }

func newBackend() Backend {
	return &windowsBackend{
		baseBackend: baseBackend{master: systemVolumeController{}, mic: newMicVolumeController(), lineIn: newLineInVolumeController(), app: newAppSessionController()},
	}
}

func (b *windowsBackend) String() string { return "windowsBackend" }

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
