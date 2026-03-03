//go:build !windows

package ui

type startupStatus struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Message   string `json:"message,omitempty"`
}

func startupStatusForConfig(_ string) (startupStatus, error) {
	return startupStatus{Supported: false, Enabled: false, Message: "startup integration is currently available on Windows packages"}, nil
}

func configureStartupForConfig(_ string, _ bool) (startupStatus, error) {
	return startupStatus{Supported: false, Enabled: false, Message: "startup integration is currently available on Windows packages"}, nil
}
