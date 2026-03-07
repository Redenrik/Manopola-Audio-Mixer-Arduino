package audio

import "errors"

var (
	// ErrTargetUnavailable indicates a configured target is currently unavailable
	// (for example, an app session that is not active right now).
	ErrTargetUnavailable = errors.New("target unavailable")
	// ErrPermissionDenied indicates the current process lacks OS permission for the request.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrUnsupported indicates the active platform/runtime cannot support the request.
	ErrUnsupported = errors.New("unsupported")
	// ErrOutputDeviceUnavailable indicates the current default output device is unavailable.
	ErrOutputDeviceUnavailable = errors.New("output device unavailable")
	// ErrTransientTapFailure indicates a temporary tap/aggregate device failure.
	ErrTransientTapFailure = errors.New("transient tap failure")
)
