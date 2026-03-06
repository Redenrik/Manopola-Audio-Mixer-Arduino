//go:build !windows && !linux && !darwin

package audio

func newMicVolumeController() volumeController {
	return nil
}
