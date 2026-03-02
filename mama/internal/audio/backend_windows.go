//go:build windows

package audio

type windowsBackend struct{ baseBackend }

func newBackend() Backend {
	return &windowsBackend{
		baseBackend: baseBackend{volume: systemVolumeController{}},
	}
}

func (b *windowsBackend) String() string { return "windowsBackend" }
