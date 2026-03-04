package audio

import "errors"

// ErrTargetUnavailable indicates a configured target is currently unavailable
// (for example, an app session that is not active right now).
var ErrTargetUnavailable = errors.New("target unavailable")
