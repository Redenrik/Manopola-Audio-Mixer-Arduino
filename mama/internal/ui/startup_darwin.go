//go:build darwin

package ui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const macStartupPlistName = "io.mama.mixer.startup.plist"
const macStartupLabel = "io.mama.mixer.startup"

func startupStatusForConfig(_ string) (startupStatus, error) {
	plistPath, err := macStartupPlistPath()
	if err != nil {
		return startupStatus{}, err
	}
	_, statErr := os.Stat(plistPath)
	enabled := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return startupStatus{}, statErr
	}
	status := startupStatus{Supported: true, Enabled: enabled}
	if enabled {
		status.Message = "startup is enabled via LaunchAgents"
	}
	return status, nil
}

func configureStartupForConfig(cfgPath string, enabled bool) (startupStatus, error) {
	plistPath, err := macStartupPlistPath()
	if err != nil {
		return startupStatus{}, err
	}

	if !enabled {
		if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return startupStatus{}, err
		}
		return startupStatus{Supported: true, Enabled: false, Message: "startup disabled"}, nil
	}

	runtimePath := filepath.Join(filepath.Dir(cfgPath), "mama")
	if _, err := os.Stat(runtimePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return startupStatus{}, fmt.Errorf("cannot enable startup: %s not found next to config", filepath.Base(runtimePath))
		}
		return startupStatus{}, err
	}

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return startupStatus{}, err
	}

	plist, err := buildMacStartupPlist(runtimePath, filepath.Join(filepath.Dir(cfgPath), "config.yaml"))
	if err != nil {
		return startupStatus{}, err
	}
	if err := os.WriteFile(plistPath, plist, 0o644); err != nil {
		return startupStatus{}, err
	}
	return startupStatus{Supported: true, Enabled: true, Message: "startup enabled (applies on next login)"}, nil
}

func macStartupPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", macStartupPlistName), nil
}

func buildMacStartupPlist(runtimePath, configPath string) ([]byte, error) {
	absRuntime, err := filepath.Abs(runtimePath)
	if err != nil {
		absRuntime = runtimePath
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		absConfig = configPath
	}
	workingDir := filepath.Dir(absRuntime)

	esc := func(v string) (string, error) {
		var b bytes.Buffer
		if err := escapeXML(&b, v); err != nil {
			return "", err
		}
		return b.String(), nil
	}

	runtimeEsc, err := esc(absRuntime)
	if err != nil {
		return nil, err
	}
	configEsc, err := esc(absConfig)
	if err != nil {
		return nil, err
	}
	workingDirEsc, err := esc(workingDir)
	if err != nil {
		return nil, err
	}

	plist := strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
		`<plist version="1.0">`,
		`  <dict>`,
		`    <key>Label</key>`,
		`    <string>` + macStartupLabel + `</string>`,
		`    <key>ProgramArguments</key>`,
		`    <array>`,
		`      <string>` + runtimeEsc + `</string>`,
		`      <string>-config</string>`,
		`      <string>` + configEsc + `</string>`,
		`      <string>-open=false</string>`,
		`      <string>-start-hidden=true</string>`,
		`    </array>`,
		`    <key>WorkingDirectory</key>`,
		`    <string>` + workingDirEsc + `</string>`,
		`    <key>RunAtLoad</key>`,
		`    <true/>`,
		`  </dict>`,
		`</plist>`,
		"",
	}, "\n")
	return []byte(plist), nil
}

func escapeXML(dst *bytes.Buffer, in string) error {
	for _, r := range in {
		switch r {
		case '&':
			dst.WriteString("&amp;")
		case '<':
			dst.WriteString("&lt;")
		case '>':
			dst.WriteString("&gt;")
		case '"':
			dst.WriteString("&quot;")
		case '\'':
			dst.WriteString("&apos;")
		default:
			if _, err := dst.WriteRune(r); err != nil {
				return err
			}
		}
	}
	return nil
}
