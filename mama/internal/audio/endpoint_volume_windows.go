//go:build windows

package audio

import "github.com/moutend/go-wca"

func invokeEndpointVolume(dataFlow uint32, fn func(*wca.IAudioEndpointVolume) (interface{}, error)) (ret interface{}, err error) {
	return invokeEndpointVolumeFor(dataFlow, "", fn)
}

func invokeEndpointVolumeFor(dataFlow uint32, selectorToken string, fn func(*wca.IAudioEndpointVolume) (interface{}, error)) (ret interface{}, err error) {
	err = withCOMApartment(func() error {
		var mmde *wca.IMMDeviceEnumerator
		if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			return err
		}
		defer mmde.Release()

		mmd, err := selectEndpointDevice(mmde, dataFlow, selectorToken)
		if err != nil {
			return err
		}
		defer mmd.Release()

		var aev *wca.IAudioEndpointVolume
		if err := activateIMMDevice(mmd, wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, &aev); err != nil {
			return err
		}
		defer aev.Release()

		var callErr error
		ret, callErr = fn(aev)
		return callErr
	})
	return ret, err
}
