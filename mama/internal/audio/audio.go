package audio

import (
	"fmt"

	"mama/internal/config"
)

type Backend interface {
	// Apply a relative change (delta steps * step size).
	// step is a scalar 0..1, e.g. 0.02 = 2%.
	Adjust(target config.TargetType, name string, step float64, deltaSteps int) error
	ToggleMute(target config.TargetType, name string) error
	ListTargets() ([]string, error) // optional for UI later
}

func NewBackend() Backend {
	return newBackend()
}

func Unsupported(target config.TargetType) error {
	return fmt.Errorf("target not supported yet: %s", target)
}