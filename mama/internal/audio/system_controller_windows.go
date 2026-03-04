//go:build windows

package audio

import (
	"errors"
	"math"

	"github.com/moutend/go-wca"
)

func (systemVolumeController) GetVolume() (int, error) {
	return systemVolumeController{}.GetVolumeFor("")
}

func (systemVolumeController) GetVolumeFor(selectorToken string) (int, error) {
	vol, err := invokeEndpointVolumeFor(wca.ERender, selectorToken, func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		var level float32
		if err := aev.GetMasterVolumeLevelScalar(&level); err != nil {
			return nil, err
		}
		return int(math.Floor(float64(level*100.0 + 0.5))), nil
	})
	if vol == nil {
		return 0, err
	}
	return vol.(int), err
}

func (systemVolumeController) SetVolume(volume int) error {
	return systemVolumeController{}.SetVolumeFor("", volume)
}

func (systemVolumeController) SetVolumeFor(selectorToken string, volume int) error {
	if volume < 0 || volume > 100 {
		return errors.New("out of valid volume range")
	}
	_, err := invokeEndpointVolumeFor(wca.ERender, selectorToken, func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		return nil, aev.SetMasterVolumeLevelScalar(float32(volume)/100, nil)
	})
	return err
}

func (systemVolumeController) GetMuted() (bool, error) {
	return systemVolumeController{}.GetMutedFor("")
}

func (systemVolumeController) GetMutedFor(selectorToken string) (bool, error) {
	muted, err := invokeEndpointVolumeFor(wca.ERender, selectorToken, func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		var muted bool
		if err := aev.GetMute(&muted); err != nil {
			return nil, err
		}
		return muted, nil
	})
	if muted == nil {
		return false, err
	}
	return muted.(bool), err
}

func (systemVolumeController) Mute() error {
	return systemVolumeController{}.MuteFor("")
}

func (systemVolumeController) MuteFor(selectorToken string) error {
	_, err := invokeEndpointVolumeFor(wca.ERender, selectorToken, func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		return nil, aev.SetMute(true, nil)
	})
	return err
}

func (systemVolumeController) Unmute() error {
	return systemVolumeController{}.UnmuteFor("")
}

func (systemVolumeController) UnmuteFor(selectorToken string) error {
	_, err := invokeEndpointVolumeFor(wca.ERender, selectorToken, func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		return nil, aev.SetMute(false, nil)
	})
	return err
}
