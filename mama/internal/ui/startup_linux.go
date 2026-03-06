//go:build linux

package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const linuxStartupDesktopFileName = "io.mama.mixer.desktop"

func startupStatusForConfig(_ string) (startupStatus, error) {
	desktopPath, err := linuxStartupDesktopPath()
	if err != nil {
		return startupStatus{}, err
	}
	_, statErr := os.Stat(desktopPath)
	enabled := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return startupStatus{}, statErr
	}
	status := startupStatus{Supported: true, Enabled: enabled}
	if enabled {
		status.Message = "startup is enabled via XDG autostart"
	}
	return status, nil
}

func configureStartupForConfig(cfgPath string, enabled bool) (startupStatus, error) {
	desktopPath, err := linuxStartupDesktopPath()
	if err != nil {
		return startupStatus{}, err
	}

	if !enabled {
		if err := os.Remove(desktopPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return startupStatus{}, err
		}
		return startupStatus{Supported: true, Enabled: false, Message: "startup disabled"}, nil
	}

	runtimePath, err := os.Executable()
	if err != nil {
		return startupStatus{}, fmt.Errorf("resolve executable path: %w", err)
	}
	runtimePath, err = filepath.EvalSymlinks(runtimePath)
	if err != nil {
		// Keep the unresolved path if symlink resolution fails.
		runtimePath, _ = os.Executable()
	}
	if _, err := os.Stat(runtimePath); err != nil {
		return startupStatus{}, fmt.Errorf("cannot enable startup: runtime executable not found: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(desktopPath), 0o755); err != nil {
		return startupStatus{}, err
	}

	entry, err := buildLinuxStartupDesktopEntry(runtimePath, cfgPath)
	if err != nil {
		return startupStatus{}, err
	}
	if err := os.WriteFile(desktopPath, entry, 0o644); err != nil {
		return startupStatus{}, err
	}

	return startupStatus{Supported: true, Enabled: true, Message: "startup enabled (applies on next login)"}, nil
}

func linuxStartupDesktopPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(cfgDir, "autostart", linuxStartupDesktopFileName), nil
}

func buildLinuxStartupDesktopEntry(runtimePath, cfgPath string) ([]byte, error) {
	absRuntime, err := filepath.Abs(strings.TrimSpace(runtimePath))
	if err != nil {
		absRuntime = strings.TrimSpace(runtimePath)
	}
	absConfig, err := filepath.Abs(strings.TrimSpace(cfgPath))
	if err != nil {
		absConfig = strings.TrimSpace(cfgPath)
	}
	if absRuntime == "" {
		return nil, fmt.Errorf("runtime path must not be empty")
	}
	if absConfig == "" {
		return nil, fmt.Errorf("config path must not be empty")
	}

	execLine := linuxDesktopExecLine(absRuntime, "-config", absConfig, "-open=false")
	workingDir := strings.TrimSpace(filepath.Dir(absRuntime))
	if workingDir == "" {
		workingDir = "."
	}

	entry := strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Version=1.0",
		"Name=MAMA Audio Mixer",
		"Comment=Start MAMA mixer runtime on login",
		"Exec=" + execLine,
		"Path=" + linuxDesktopFieldValue(workingDir),
		"Terminal=false",
		"X-GNOME-Autostart-enabled=true",
		"",
	}, "\n")
	return []byte(entry), nil
}

func linuxDesktopFieldValue(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
}

func linuxDesktopExecLine(args ...string) string {
	escaped := make([]string, 0, len(args))
	for _, arg := range args {
		escaped = append(escaped, linuxDesktopEscapeArg(arg))
	}
	return strings.Join(escaped, " ")
}

func linuxDesktopEscapeArg(arg string) string {
	raw := strings.TrimSpace(arg)
	if raw == "" {
		return `""`
	}

	escaped := strings.ReplaceAll(raw, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "$", `\$`)
	escaped = strings.ReplaceAll(escaped, "`", "\\`")

	if strings.ContainsAny(raw, " \t\n") {
		return `"` + escaped + `"`
	}
	return escaped
}
