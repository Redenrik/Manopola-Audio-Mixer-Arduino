//go:build !windows

package audio

type unixBackend struct{ baseBackend }

func newBackend() Backend {
	return &unixBackend{
		baseBackend: baseBackend{master: systemVolumeController{}, mic: newMicVolumeController(), lineIn: newLineInVolumeController(), app: newAppSessionController()},
	}
}
