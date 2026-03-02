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
	volume volumeController
}

func (b *baseBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
	switch target {
	case config.TargetMasterOut:
		cur, err := b.volume.GetVolume()
		if err != nil {
			return err
		}
		curF := float64(cur) / 100.0
		next := curF + (step * float64(deltaSteps))
		next = math.Max(0, math.Min(1, next))
		return b.volume.SetVolume(int(math.Round(next * 100)))
	default:
		return Unsupported(target)
	}
}

func (b *baseBackend) ToggleMute(target config.TargetType, name string) error {
	switch target {
	case config.TargetMasterOut:
		muted, err := b.volume.GetMuted()
		if err != nil {
			return err
		}
		if muted {
			return b.volume.Unmute()
		}
		return b.volume.Mute()
	default:
		return Unsupported(target)
	}
}

func (b *baseBackend) ListTargets() ([]DiscoveredTarget, error) {
	return []DiscoveredTarget{
		{ID: "system:master_out", Type: config.TargetMasterOut, Name: "System Output"},
	}, nil
}
