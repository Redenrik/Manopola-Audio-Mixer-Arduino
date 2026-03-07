//go:build darwin

package config

import (
	"path/filepath"
	"testing"
)

func TestDarwinAppBundleConfigPath(t *testing.T) {
	tests := []struct {
		name          string
		exePath       string
		userConfigDir string
		wantPath      string
		wantOK        bool
	}{
		{
			name:          "app bundle executable uses application support",
			exePath:       "/Applications/MAMA.app/Contents/MacOS/MAMA",
			userConfigDir: "/Users/tester/Library/Application Support",
			wantPath:      "/Users/tester/Library/Application Support/MAMA/config.yaml",
			wantOK:        true,
		},
		{
			name:          "standalone binary keeps legacy resolution",
			exePath:       "/usr/local/bin/mama",
			userConfigDir: "/Users/tester/Library/Application Support",
			wantPath:      "",
			wantOK:        false,
		},
		{
			name:          "empty user config dir rejected",
			exePath:       "/Applications/MAMA.app/Contents/MacOS/MAMA",
			userConfigDir: "",
			wantPath:      "",
			wantOK:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := darwinAppBundleConfigPath(tc.exePath, tc.userConfigDir)
			if gotOK != tc.wantOK {
				t.Fatalf("ok=%t want %t", gotOK, tc.wantOK)
			}
			if filepath.Clean(gotPath) != filepath.Clean(tc.wantPath) {
				t.Fatalf("path=%q want %q", gotPath, tc.wantPath)
			}
		})
	}
}
