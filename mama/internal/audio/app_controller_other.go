//go:build !windows && !linux && !darwin

package audio

func newAppSessionController() appSessionController {
	return nil
}
