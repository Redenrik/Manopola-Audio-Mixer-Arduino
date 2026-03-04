//go:build windows

package audio

import (
	"errors"
	"math"

	"github.com/moutend/go-wca"
)

type windowsMicVolumeController struct{}

func newMicVolumeController() volumeController {
	return windowsMicVolumeController{}
}

func (windowsMicVolumeController) GetVolume() (int, error) {
	vol, err := invokeCaptureEndpoint(func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		var level float32
		err := aev.GetMasterVolumeLevelScalar(&level)
		return int(math.Floor(float64(level*100.0 + 0.5))), err
	})
	if vol == nil {
		return 0, err
	}
	return vol.(int), err
}

func (windowsMicVolumeController) SetVolume(volume int) error {
	if volume < 0 || volume > 100 {
		return errors.New("out of valid volume range")
	}
	_, err := invokeCaptureEndpoint(func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		return nil, aev.SetMasterVolumeLevelScalar(float32(volume)/100, nil)
	})
	return err
}

func (windowsMicVolumeController) GetMuted() (bool, error) {
	muted, err := invokeCaptureEndpoint(func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		var muted bool
		err := aev.GetMute(&muted)
		return muted, err
	})
	if muted == nil {
		return false, err
	}
	return muted.(bool), err
}

func (windowsMicVolumeController) Mute() error {
	_, err := invokeCaptureEndpoint(func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		return nil, aev.SetMute(true, nil)
	})
	return err
}

func (windowsMicVolumeController) Unmute() error {
	_, err := invokeCaptureEndpoint(func(aev *wca.IAudioEndpointVolume) (interface{}, error) {
		return nil, aev.SetMute(false, nil)
	})
	return err
}

func invokeCaptureEndpoint(f func(aev *wca.IAudioEndpointVolume) (interface{}, error)) (ret interface{}, err error) {
	return invokeEndpointVolume(wca.ECapture, f)
}
