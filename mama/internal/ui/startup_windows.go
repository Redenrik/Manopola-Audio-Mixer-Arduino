//go:build windows

package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const startupScriptName = "MAMA Start Mixer.cmd"

type startupStatus struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Message   string `json:"message,omitempty"`
}

func startupStatusForConfig(cfgPath string) (startupStatus, error) {
	scriptPath, err := startupScriptPath()
	if err != nil {
		return startupStatus{}, err
	}
	_, statErr := os.Stat(scriptPath)
	enabled := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return startupStatus{}, statErr
	}
	status := startupStatus{Supported: true, Enabled: enabled}
	if enabled {
		status.Message = fmt.Sprintf("startup is enabled via %s", startupScriptName)
	}
	return status, nil
}

func configureStartupForConfig(cfgPath string, enabled bool) (startupStatus, error) {
	scriptPath, err := startupScriptPath()
	if err != nil {
		return startupStatus{}, err
	}
	if !enabled {
		if err := os.Remove(scriptPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return startupStatus{}, err
		}
		return startupStatus{Supported: true, Enabled: false, Message: "startup disabled"}, nil
	}

	launcherPath := filepath.Join(filepath.Dir(cfgPath), "Start Mixer.cmd")
	if _, err := os.Stat(launcherPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return startupStatus{}, fmt.Errorf("cannot enable startup: %s not found next to config", filepath.Base(launcherPath))
		}
		return startupStatus{}, err
	}

	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return startupStatus{}, err
	}

	script := buildStartupScript(launcherPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		return startupStatus{}, err
	}
	return startupStatus{Supported: true, Enabled: true, Message: "startup enabled"}, nil
}

func startupScriptPath() (string, error) {
	appData := strings.TrimSpace(os.Getenv("APPDATA"))
	if appData == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve startup folder: APPDATA and UserConfigDir unavailable: %w", err)
		}
		appData = cfgDir
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", startupScriptName), nil
}

func buildStartupScript(launcherPath string) string {
	absLauncher := launcherPath
	if v, err := filepath.Abs(launcherPath); err == nil {
		absLauncher = v
	}
	dir := filepath.Dir(absLauncher)
	return "@echo off\r\n" +
		"cd /d \"" + dir + "\"\r\n" +
		"call \"" + absLauncher + "\"\r\n"
}
