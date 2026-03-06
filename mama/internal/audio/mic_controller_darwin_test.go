//go:build darwin

package audio

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestParseBoundedPercent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "normal", input: "42", want: 42},
		{name: "clamp_low", input: "-3", want: 0},
		{name: "clamp_high", input: "300", want: 100},
		{name: "invalid", input: "x", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseBoundedPercent(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestDarwinMicVolumeControllerMuteUnmuteRoundTrip(t *testing.T) {
	t.Parallel()

	currentVolume := 67
	scripts := make([]string, 0, 8)

	controller := &darwinMicVolumeController{
		lastNonZero: 50,
		run: func(script string) (string, error) {
			scripts = append(scripts, script)
			switch {
			case script == "input volume of (get volume settings)":
				return strconv.Itoa(currentVolume), nil
			case strings.HasPrefix(script, "set volume input volume "):
				raw := strings.TrimPrefix(script, "set volume input volume ")
				v, err := strconv.Atoi(strings.TrimSpace(raw))
				if err != nil {
					return "", err
				}
				currentVolume = v
				return "", nil
			default:
				return "", fmt.Errorf("unexpected script: %s", script)
			}
		},
	}

	if err := controller.Mute(); err != nil {
		t.Fatalf("Mute() error = %v", err)
	}
	if currentVolume != 0 {
		t.Fatalf("currentVolume after mute = %d want 0", currentVolume)
	}

	controller.mu.Lock()
	saved := controller.lastNonZero
	controller.mu.Unlock()
	if saved != 67 {
		t.Fatalf("lastNonZero after mute = %d want 67", saved)
	}

	if err := controller.Unmute(); err != nil {
		t.Fatalf("Unmute() error = %v", err)
	}
	if currentVolume != 67 {
		t.Fatalf("currentVolume after unmute = %d want 67", currentVolume)
	}

	if len(scripts) < 3 {
		t.Fatalf("expected scripts to be executed, got %v", scripts)
	}
}
