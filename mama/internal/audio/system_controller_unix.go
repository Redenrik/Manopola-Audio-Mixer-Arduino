//go:build !windows

package audio

import "github.com/itchyny/volume-go"

func (systemVolumeController) GetVolume() (int, error) { return volume.GetVolume() }
func (systemVolumeController) SetVolume(v int) error   { return volume.SetVolume(v) }
func (systemVolumeController) GetMuted() (bool, error) { return volume.GetMuted() }
func (systemVolumeController) Mute() error             { return volume.Mute() }
func (systemVolumeController) Unmute() error           { return volume.Unmute() }
