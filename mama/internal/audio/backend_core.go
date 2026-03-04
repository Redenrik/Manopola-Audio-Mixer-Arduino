package audio

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

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

type baseBackend struct {
	master volumeController
	mic    volumeController
	lineIn volumeController
	app    appSessionController
}

type appSessionController interface {
	Adjust(selectorToken string, step float64, deltaSteps int) error
	ToggleMute(selectorToken string) error
	AdjustGroup(selectors []config.Selector, step float64, deltaSteps int) error
	ToggleMuteGroup(selectors []config.Selector) error
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
	if target == config.TargetGroup {
		if b.app == nil {
			return Unsupported(target)
		}
		selectors, err := parseGroupSelectorsToken(name)
		if err != nil {
			return err
		}
		return b.app.AdjustGroup(selectors, step, deltaSteps)
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
	if target == config.TargetGroup {
		if b.app == nil {
			return Unsupported(target)
		}
		selectors, err := parseGroupSelectorsToken(name)
		if err != nil {
			return err
		}
		return b.app.ToggleMuteGroup(selectors)
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
		if err == nil {
			targets = append(targets, appTargets...)
		}
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

func parseGroupSelectorsToken(token string) ([]config.Selector, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("group target requires selectors token")
	}
	var selectors []config.Selector
	if err := json.Unmarshal([]byte(token), &selectors); err != nil {
		return nil, fmt.Errorf("decode group selectors token: %w", err)
	}
	if len(selectors) == 0 {
		return nil, fmt.Errorf("group target requires at least one selector")
	}
	return selectors, nil
}
