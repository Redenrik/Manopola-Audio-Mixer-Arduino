package ui

type startupStatus struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Message   string `json:"message,omitempty"`
}
