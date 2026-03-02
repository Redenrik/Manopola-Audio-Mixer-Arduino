//go:build windows

package audio

import "mama/internal/config"

type windowsAppSessionController struct{}

func newAppSessionController() appSessionController {
	return &windowsAppSessionController{}
}

func (w *windowsAppSessionController) Adjust(selectorToken string, step float64, deltaSteps int) error {
	return Unsupported(config.TargetApp)
}

func (w *windowsAppSessionController) ToggleMute(selectorToken string) error {
	return Unsupported(config.TargetApp)
}

func (w *windowsAppSessionController) AdjustGroup(selectors []config.Selector, step float64, deltaSteps int) error {
	return Unsupported(config.TargetGroup)
}

func (w *windowsAppSessionController) ToggleMuteGroup(selectors []config.Selector) error {
	return Unsupported(config.TargetGroup)
}

func (w *windowsAppSessionController) ListTargets() ([]DiscoveredTarget, error) {
	return nil, nil
}
