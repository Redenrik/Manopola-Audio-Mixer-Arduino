//go:build !windows

package audio

import (
	"math"

	"github.com/itchyny/volume-go"
	"mama/internal/config"
)

type unixBackend struct{}

func newBackend() Backend { return &unixBackend{} }

// volume-go is used for system master volume.
// Per-app volumes will likely be Linux-only via PulseAudio/PipeWire in a later backend.
func (b *unixBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
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

func (b *unixBackend) ToggleMute(target config.TargetType, name string) error {
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

func (b *unixBackend) ListTargets() ([]string, error) {
	return []string{"system:master_out"}, nil
}
