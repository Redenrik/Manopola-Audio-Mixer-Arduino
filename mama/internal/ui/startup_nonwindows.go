//go:build !windows && !darwin && !linux

package ui

func startupStatusForConfig(_ string) (startupStatus, error) {
	return startupStatus{Supported: false, Enabled: false, Message: "startup integration is not supported on this platform"}, nil
}

func configureStartupForConfig(_ string, _ bool) (startupStatus, error) {
	return startupStatus{Supported: false, Enabled: false, Message: "startup integration is not supported on this platform"}, nil
}
