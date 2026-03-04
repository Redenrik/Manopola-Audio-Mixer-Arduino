//go:build windows

package audio

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca"
	"golang.org/x/sys/windows"
)

func activateIMMDevice[T any](device *wca.IMMDevice, iid *ole.GUID, clsctx uint32, out **T) error {
	if device == nil {
		return errors.New("nil IMMDevice")
	}
	if iid == nil {
		return errors.New("nil interface GUID")
	}
	if out == nil {
		return errors.New("nil output pointer")
	}

	hr, _, _ := syscall.Syscall6(
		device.VTable().Activate,
		5,
		uintptr(unsafe.Pointer(device)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(clsctx),
		0,
		uintptr(unsafe.Pointer(out)),
		0,
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func getIMMDeviceID(device *wca.IMMDevice) (string, error) {
	if device == nil {
		return "", errors.New("nil IMMDevice")
	}

	var idPtr *uint16
	hr, _, _ := syscall.Syscall(
		device.VTable().GetId,
		2,
		uintptr(unsafe.Pointer(device)),
		uintptr(unsafe.Pointer(&idPtr)),
		0,
	)
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	if idPtr == nil {
		return "", errors.New("empty endpoint id")
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(idPtr)))

	id := windows.UTF16PtrToString(idPtr)
	if id == "" {
		return "", errors.New("empty endpoint id")
	}
	return id, nil
}
