//go:build !windows

package audio

import (
	"strings"
	"testing"

	"mama/internal/config"
)

func TestParsePactlSinkInputs(t *testing.T) {
	out := `Sink Input #42
    Volume: front-left: 32768 /  50% / -18.06 dB
    Mute: no
    Properties:
        application.name = "Discord"
        application.process.binary = "discord"
Sink Input #88
    Volume: front-left: 65536 / 100% / 0.00 dB
    Mute: yes
    Properties:
        application.name = "Spotify"
        application.process.binary = "spotify"`

	sessions := parsePactlSinkInputs(out)
	if len(sessions) != 2 {
		t.Fatalf("len(sessions)=%d want 2", len(sessions))
	}
	if sessions[0].id != "42" || sessions[0].name != "Discord" || sessions[0].exe != "discord" || sessions[0].volume != 50 || sessions[0].isMuted {
		t.Fatalf("unexpected first session: %#v", sessions[0])
	}
	if sessions[1].id != "88" || !sessions[1].isMuted {
		t.Fatalf("unexpected second session: %#v", sessions[1])
	}
}

func TestSessionMatchesSelector(t *testing.T) {
	s := appSession{name: "Discord PTB", exe: "discord"}
	cases := []struct {
		sel  config.Selector
		want bool
	}{
		{config.Selector{Kind: config.SelectorContains, Value: "cord"}, true},
		{config.Selector{Kind: config.SelectorPrefix, Value: "Dis"}, true},
		{config.Selector{Kind: config.SelectorSuffix, Value: "PTB"}, true},
		{config.Selector{Kind: config.SelectorGlob, Value: "Disc*PTB"}, true},
		{config.Selector{Kind: config.SelectorExe, Value: "discord"}, true},
		{config.Selector{Kind: config.SelectorExact, Value: "discord"}, true},
		{config.Selector{Kind: config.SelectorExact, Value: "Discord PTB"}, true},
		{config.Selector{Kind: config.SelectorExact, Value: "Slack"}, false},
	}
	for _, tc := range cases {
		if got := sessionMatchesSelector(s, tc.sel); got != tc.want {
			t.Fatalf("selector %+v got %v want %v", tc.sel, got, tc.want)
		}
	}
}

func TestUnixAppControllerAdjustAndToggle(t *testing.T) {
	var calls []string
	controller := &unixAppSessionController{run: func(name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		calls = append(calls, cmd)
		if len(args) >= 2 && args[0] == "list" && args[1] == "sink-inputs" {
			return `Sink Input #7
    Volume: front-left: 65536 / 100% / 0.00 dB
    Mute: no
    Properties:
        application.name = "Discord"
        application.process.binary = "discord"`, nil
		}
		return "", nil
	}}

	if err := controller.Adjust("exact:Discord", 0.02, -3); err != nil {
		t.Fatalf("Adjust error: %v", err)
	}
	if err := controller.ToggleMute("exe:discord"); err != nil {
		t.Fatalf("ToggleMute error: %v", err)
	}

	if len(calls) < 4 {
		t.Fatalf("calls=%v", calls)
	}
	if calls[1] != "pactl set-sink-input-volume 7 94%" {
		t.Fatalf("unexpected set volume call: %s", calls[1])
	}
	if calls[3] != "pactl set-sink-input-mute 7 1" {
		t.Fatalf("unexpected mute call: %s", calls[3])
	}
}

func TestUnixAppControllerGroupAdjustAndToggle(t *testing.T) {
	var calls []string
	controller := &unixAppSessionController{run: func(name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		calls = append(calls, cmd)
		if len(args) >= 2 && args[0] == "list" && args[1] == "sink-inputs" {
			return `Sink Input #7
    Volume: front-left: 65536 / 100% / 0.00 dB
    Mute: no
    Properties:
        application.name = "Discord"
        application.process.binary = "discord"
Sink Input #9
    Volume: front-left: 32768 / 50% / -18.06 dB
    Mute: yes
    Properties:
        application.name = "Spotify"
        application.process.binary = "spotify"`, nil
		}
		return "", nil
	}}

	selectors := []config.Selector{{Kind: config.SelectorExact, Value: "Discord"}, {Kind: config.SelectorExe, Value: "spotify"}}
	if err := controller.AdjustGroup(selectors, 0.02, -2); err != nil {
		t.Fatalf("AdjustGroup error: %v", err)
	}
	if err := controller.ToggleMuteGroup(selectors); err != nil {
		t.Fatalf("ToggleMuteGroup error: %v", err)
	}

	if len(calls) < 6 {
		t.Fatalf("calls=%v", calls)
	}
	if calls[1] != "pactl set-sink-input-volume 7 96%" || calls[2] != "pactl set-sink-input-volume 9 46%" {
		t.Fatalf("unexpected group volume calls: %v", calls)
	}
	if calls[4] != "pactl set-sink-input-mute 7 1" || calls[5] != "pactl set-sink-input-mute 9 0" {
		t.Fatalf("unexpected group mute calls: %v", calls)
	}
}

func TestUnixAppControllerAdjustVolumeUpUnmutes(t *testing.T) {
	var calls []string
	controller := &unixAppSessionController{run: func(name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		calls = append(calls, cmd)
		if len(args) >= 2 && args[0] == "list" && args[1] == "sink-inputs" {
			return `Sink Input #7
    Volume: front-left: 32768 /  50% / -18.06 dB
    Mute: yes
    Properties:
        application.name = "Discord"
        application.process.binary = "discord"`, nil
		}
		return "", nil
	}}

	if err := controller.Adjust("exact:Discord", 0.02, 2); err != nil {
		t.Fatalf("Adjust error: %v", err)
	}

	if len(calls) < 3 {
		t.Fatalf("calls=%v", calls)
	}
	if calls[1] != "pactl set-sink-input-volume 7 54%" {
		t.Fatalf("unexpected set volume call: %s", calls[1])
	}
	if calls[2] != "pactl set-sink-input-mute 7 0" {
		t.Fatalf("expected unmute call after volume-up, calls=%v", calls)
	}
}

func TestUnixAppControllerAdjustGroupVolumeUpUnmutesMatchedMutedSessions(t *testing.T) {
	var calls []string
	controller := &unixAppSessionController{run: func(name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		calls = append(calls, cmd)
		if len(args) >= 2 && args[0] == "list" && args[1] == "sink-inputs" {
			return `Sink Input #7
    Volume: front-left: 32768 /  50% / -18.06 dB
    Mute: yes
    Properties:
        application.name = "Discord"
        application.process.binary = "discord"
Sink Input #9
    Volume: front-left: 32768 /  50% / -18.06 dB
    Mute: no
    Properties:
        application.name = "Spotify"
        application.process.binary = "spotify"`, nil
		}
		return "", nil
	}}

	selectors := []config.Selector{{Kind: config.SelectorExact, Value: "Discord"}, {Kind: config.SelectorExe, Value: "spotify"}}
	if err := controller.AdjustGroup(selectors, 0.02, 1); err != nil {
		t.Fatalf("AdjustGroup error: %v", err)
	}

	if len(calls) < 4 {
		t.Fatalf("calls=%v", calls)
	}
	if calls[1] != "pactl set-sink-input-volume 7 52%" || calls[3] != "pactl set-sink-input-volume 9 52%" {
		t.Fatalf("unexpected group volume calls: %v", calls)
	}
	if calls[2] != "pactl set-sink-input-mute 7 0" {
		t.Fatalf("expected unmute call for muted session, calls=%v", calls)
	}
}
