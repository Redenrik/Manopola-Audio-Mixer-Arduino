//go:build !windows && !darwin

package ui

func startupStatusForConfig(_ string) (startupStatus, error) {
	return startupStatus{Supported: false, Enabled: false, Message: "startup integration is currently available on Windows and macOS packages"}, nil
}

func configureStartupForConfig(_ string, _ bool) (startupStatus, error) {
	return startupStatus{Supported: false, Enabled: false, Message: "startup integration is currently available on Windows and macOS packages"}, nil
}
