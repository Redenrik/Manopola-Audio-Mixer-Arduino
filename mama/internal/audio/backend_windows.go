//go:build windows

package audio

import (
	"math"

	"github.com/itchyny/volume-go"
	"mama/internal/config"
)

type windowsBackend struct{}

func newBackend() Backend { return &windowsBackend{} }

// For now: implement master_out using volume-go.
// Per-app + mic/line-in will be added with dedicated Windows Core Audio integration.
func (b *windowsBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
	switch target {
	case config.TargetMasterOut:
		cur, err := volume.GetVolume()
		if err != nil {
			return err
		}
		curF := float64(cur) / 100.0
		next := curF + (step * float64(deltaSteps))
		next = math.Max(0, math.Min(1, next))
		return volume.SetVolume(int(math.Round(next * 100)))
	default:
		return Unsupported(target)
	}
}

func (b *windowsBackend) ToggleMute(target config.TargetType, name string) error {
	switch target {
	case config.TargetMasterOut:
		m, err := volume.GetMuted()
		if err != nil {
			return err
		}
		if m {
			return volume.Unmute()
		}
		return volume.Mute()
	default:
		return Unsupported(target)
	}
}

func (b *windowsBackend) ListTargets() ([]string, error) {
	// Future: enumerate sessions + input devices
	return []string{"system:master_out"}, nil
}

func (b *windowsBackend) String() string { return "windowsBackend" }
