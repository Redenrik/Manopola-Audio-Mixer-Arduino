//go:build !windows

package audio

func newLineInVolumeController() volumeController {
	return newMicVolumeController()
}
