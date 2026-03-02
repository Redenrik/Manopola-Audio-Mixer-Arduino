//go:build windows

package audio

type windowsBackend struct{ baseBackend }

func newBackend() Backend {
	return &windowsBackend{
		baseBackend: baseBackend{master: systemVolumeController{}, mic: newMicVolumeController()},
	}
}

func (b *windowsBackend) String() string { return "windowsBackend" }
