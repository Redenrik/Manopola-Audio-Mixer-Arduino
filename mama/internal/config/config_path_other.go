//go:build !darwin

package config

func platformDefaultPath() (string, bool) {
	return "", false
}
