//go:build !windows

package audio

type unixBackend struct{ baseBackend }

func newBackend() Backend {
	return &unixBackend{
		baseBackend: baseBackend{volume: systemVolumeController{}},
	}
}
