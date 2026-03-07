//go:build darwin

package config

import (
	"os"
	"path/filepath"
	"strings"
)

func platformDefaultPath() (string, bool) {
	exePath, err := os.Executable()
	if err != nil {
		return "", false
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", false
	}

	return darwinAppBundleConfigPath(exePath, userConfigDir)
}

func darwinAppBundleConfigPath(exePath, userConfigDir string) (string, bool) {
	exePath = strings.TrimSpace(exePath)
	userConfigDir = strings.TrimSpace(userConfigDir)
	if exePath == "" || userConfigDir == "" {
		return "", false
	}
	exePath = filepath.Clean(exePath)
	userConfigDir = filepath.Clean(userConfigDir)

	macOSDir := filepath.Base(filepath.Dir(exePath))
	contentsDir := filepath.Base(filepath.Dir(filepath.Dir(exePath)))
	bundleDir := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(exePath))))
	if !strings.EqualFold(macOSDir, "MacOS") || !strings.EqualFold(contentsDir, "Contents") || !strings.HasSuffix(strings.ToLower(bundleDir), ".app") {
		return "", false
	}

	return filepath.Join(userConfigDir, "MAMA", "config.yaml"), true
}
