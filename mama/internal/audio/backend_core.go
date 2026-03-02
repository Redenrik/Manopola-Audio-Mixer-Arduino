package audio

import (
	"fmt"
	"math"
	"strings"

	"github.com/itchyny/volume-go"
	"mama/internal/config"
)

type volumeController interface {
	GetVolume() (int, error)
	SetVolume(int) error
	GetMuted() (bool, error)
	Mute() error
	Unmute() error
}

type systemVolumeController struct{}

func (systemVolumeController) GetVolume() (int, error) { return volume.GetVolume() }
func (systemVolumeController) SetVolume(v int) error   { return volume.SetVolume(v) }
func (systemVolumeController) GetMuted() (bool, error) { return volume.GetMuted() }
func (systemVolumeController) Mute() error             { return volume.Mute() }
func (systemVolumeController) Unmute() error           { return volume.Unmute() }

type baseBackend struct {
	master volumeController
	mic    volumeController
	lineIn volumeController
	app    appSessionController
}

type appSessionController interface {
	Adjust(selectorToken string, step float64, deltaSteps int) error
	ToggleMute(selectorToken string) error
	ListTargets() ([]DiscoveredTarget, error)
}

func (b *baseBackend) controllerFor(target config.TargetType) volumeController {
	switch target {
	case config.TargetMasterOut:
		return b.master
	case config.TargetMicIn:
		return b.mic
	case config.TargetLineIn:
		return b.lineIn
	default:
		return nil
	}
}

func (b *baseBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
	if target == config.TargetApp {
		if b.app == nil {
			return Unsupported(target)
		}
		return b.app.Adjust(name, step, deltaSteps)
	}

	controller := b.controllerFor(target)
	if controller == nil {
		return Unsupported(target)
	}
	cur, err := controller.GetVolume()
	if err != nil {
		return err
	}
	curF := float64(cur) / 100.0
	next := curF + (step * float64(deltaSteps))
	next = math.Max(0, math.Min(1, next))
	return controller.SetVolume(int(math.Round(next * 100)))
}

func (b *baseBackend) ToggleMute(target config.TargetType, name string) error {
	if target == config.TargetApp {
		if b.app == nil {
			return Unsupported(target)
		}
		return b.app.ToggleMute(name)
	}

	controller := b.controllerFor(target)
	if controller == nil {
		return Unsupported(target)
	}
	muted, err := controller.GetMuted()
	if err != nil {
		return err
	}
	if muted {
		return controller.Unmute()
	}
	return controller.Mute()
}

func (b *baseBackend) ListTargets() ([]DiscoveredTarget, error) {
	targets := []DiscoveredTarget{
		{ID: "system:master_out", Type: config.TargetMasterOut, Name: "System Output"},
	}
	if b.mic != nil {
		targets = append(targets, DiscoveredTarget{ID: "system:mic_in", Type: config.TargetMicIn, Name: "System Microphone"})
	}
	if b.lineIn != nil {
		targets = append(targets, DiscoveredTarget{ID: "system:line_in", Type: config.TargetLineIn, Name: "System Line Input"})
	}
	if b.app != nil {
		appTargets, err := b.app.ListTargets()
		if err != nil {
			return nil, fmt.Errorf("list app targets: %w", err)
		}
		targets = append(targets, appTargets...)
	}
	return targets, nil
}

func parseSelectorToken(token string) config.Selector {
	kind := config.SelectorExact
	value := strings.TrimSpace(token)
	if left, right, ok := strings.Cut(value, ":"); ok {
		candidate := config.SelectorKind(strings.TrimSpace(left))
		switch candidate {
		case config.SelectorExact, config.SelectorContains, config.SelectorPrefix, config.SelectorSuffix, config.SelectorGlob, config.SelectorExe:
			kind = candidate
			value = strings.TrimSpace(right)
		}
	}
	return config.Selector{Kind: kind, Value: value}
}
