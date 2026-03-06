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

func NewBackend() Backend {
	return newBackend()
}

func Unsupported(target config.TargetType) error {
	return fmt.Errorf("target not supported yet: %s", target)
}

func boolPtr(value bool) *bool {
	out := value
	return &out
}
