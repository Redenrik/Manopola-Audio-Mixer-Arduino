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
	ReadState(target config.TargetType, name string) (TargetState, error)
	ListTargets() ([]DiscoveredTarget, error)
}

type TargetCapabilities struct {
	Volume *bool `json:"volume,omitempty"`
	Mute   *bool `json:"mute,omitempty"`
}

type DiscoveredTarget struct {
	ID           string              `json:"id"`
	Type         config.TargetType   `json:"type"`
	Name         string              `json:"name,omitempty"`
	Selector     string              `json:"selector,omitempty"`
	Aliases      []string            `json:"aliases,omitempty"`
	Capabilities *TargetCapabilities `json:"capabilities,omitempty"`
}

type TargetState struct {
	Available    bool `json:"available"`
	Volume       int  `json:"volume"`
	Muted        bool `json:"muted"`
	SessionCount int  `json:"sessionCount,omitempty"`
}

type unsupportedTargetError struct {
	target config.TargetType
}

func (e unsupportedTargetError) Error() string {
	return fmt.Sprintf("target not supported yet: %s", e.target)
}

func (e unsupportedTargetError) Unwrap() error {
	return ErrUnsupported
}

func (e unsupportedTargetError) Is(target error) bool {
	if target == nil {
		return false
	}
	if target == ErrUnsupported {
		return true
	}
	switch other := target.(type) {
	case unsupportedTargetError:
		return e.target == other.target
	case *unsupportedTargetError:
		return other != nil && e.target == other.target
	default:
		return false
	}
}

func NewBackend() Backend {
	return newBackend()
}

func Unsupported(target config.TargetType) error {
	return unsupportedTargetError{target: target}
}

func boolPtr(value bool) *bool {
	out := value
	return &out
}
