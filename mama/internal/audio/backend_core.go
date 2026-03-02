package audio

import (
	"math"

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
}

func (b *baseBackend) controllerFor(target config.TargetType) volumeController {
	switch target {
	case config.TargetMasterOut:
		return b.master
	case config.TargetMicIn:
		return b.mic
	default:
		return nil
	}
}

func (b *baseBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
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
	return targets, nil
}
