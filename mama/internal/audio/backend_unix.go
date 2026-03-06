//go:build !windows

package audio

import "mama/internal/config"

type unixBackend struct{ baseBackend }

func newBackend() Backend {
	return &unixBackend{
		baseBackend: baseBackend{master: systemVolumeController{}, mic: newMicVolumeController(), lineIn: newLineInVolumeController(), app: newAppSessionController()},
	}
}

func (b *unixBackend) SupportedTargetTypes() []config.TargetType {
	return b.baseBackend.supportedTargetTypes()
}
