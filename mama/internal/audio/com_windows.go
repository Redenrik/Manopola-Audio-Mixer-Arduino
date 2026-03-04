//go:build windows

package audio

import (
	"runtime"

	"github.com/go-ole/go-ole"
)

const (
	hResultSFalse         = uintptr(0x00000001)
	hResultRPCChangedMode = uintptr(0x80010106)
)

func withCOMApartment(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	initialized, err := initializeCOM()
	if err != nil {
		return err
	}
	if initialized {
		defer ole.CoUninitialize()
	}

	return fn()
}

func initializeCOM() (bool, error) {
	if ok, err := tryCOMInit(ole.COINIT_APARTMENTTHREADED); ok || err == nil {
		return ok, err
	} else if err != nil {
		oleErr, isOleErr := err.(*ole.OleError)
		if !isOleErr || oleErr.Code() != hResultRPCChangedMode {
			return false, err
		}
	}

	return tryCOMInit(ole.COINIT_MULTITHREADED)
}

func tryCOMInit(mode uint32) (bool, error) {
	err := ole.CoInitializeEx(0, mode)
	if err == nil {
		return true, nil
	}
	oleErr, ok := err.(*ole.OleError)
	if !ok {
		return false, err
	}
	if oleErr.Code() == hResultSFalse {
		return true, nil
	}
	return false, err
}
