//go:build darwin && !cgo

package audio

func newAppSessionController() appSessionController {
	return nil
}
