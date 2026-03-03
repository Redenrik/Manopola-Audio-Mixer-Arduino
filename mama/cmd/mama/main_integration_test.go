package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/runtime"
)

type backendCall struct {
	op     string
	target config.TargetType
	name   string
	step   float64
	delta  int
}

type fakeBackend struct{ calls []backendCall }

func (f *fakeBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
	f.calls = append(f.calls, backendCall{op: "adjust", target: target, name: name, step: step, delta: deltaSteps})
	return nil
}
func (f *fakeBackend) ToggleMute(target config.TargetType, name string) error {
	f.calls = append(f.calls, backendCall{op: "toggle", target: target, name: name})
	return nil
}
func (f *fakeBackend) ListTargets() ([]audio.DiscoveredTarget, error) { return nil, nil }

func TestRunSessionFromChannels_RecordedFixtureMixedBurst(t *testing.T) {
	cfg := &config.Config{Mappings: []config.Mapping{{Knob: 1, Target: config.TargetMasterOut, Step: 0.02}}}
	backend := &fakeBackend{}
	metrics := runtime.NewMetrics()

	lineCh, readErrCh := replayFixture(t, "mixed_burst.txt", nil)
	if err := runSessionFromChannels(context.Background(), cfg, backend, metrics, lineCh, readErrCh); err != nil {
		t.Fatalf("runSessionFromChannels returned error: %v", err)
	}

	if len(backend.calls) != 3 {
		t.Fatalf("expected 3 backend calls, got %d", len(backend.calls))
	}
	if got := backend.calls[0]; got.op != "adjust" || got.delta != 2 {
		t.Fatalf("unexpected first call: %+v", got)
	}
	if got := backend.calls[1]; got.op != "toggle" {
		t.Fatalf("unexpected second call: %+v", got)
	}
	if got := backend.calls[2]; got.op != "adjust" || got.delta != -3 {
		t.Fatalf("unexpected third call: %+v", got)
	}

	snapshot := metrics.Snapshot()
	if snapshot.ParseErrors != 2 {
		t.Fatalf("expected parse_errors=2, got %d", snapshot.ParseErrors)
	}
}

func TestRunSessionFromChannels_RecordedFixtureDisconnect(t *testing.T) {
	cfg := &config.Config{Mappings: []config.Mapping{{Knob: 1, Target: config.TargetMasterOut, Step: 0.05}}}
	backend := &fakeBackend{}
	metrics := runtime.NewMetrics()

	disconnectErr := errors.New("serial closed")
	lineCh, readErrCh := replayFixture(t, "disconnect_after_events.txt", disconnectErr)
	err := runSessionFromChannels(context.Background(), cfg, backend, metrics, lineCh, readErrCh)
	if !errors.Is(err, disconnectErr) {
		t.Fatalf("expected disconnect error, got %v", err)
	}

	if len(backend.calls) != 2 {
		t.Fatalf("expected 2 backend calls before disconnect, got %d", len(backend.calls))
	}
}

func replayFixture(t *testing.T, name string, finalErr error) (<-chan string, <-chan error) {
	t.Helper()
	lines := readFixtureLines(t, name)
	lineCh := make(chan string)
	readErrCh := make(chan error, 1)

	go func() {
		defer close(lineCh)
		for _, line := range lines {
			lineCh <- line
		}
		readErrCh <- finalErr
	}()

	return lineCh, readErrCh
}

func readFixtureLines(t *testing.T, name string) []string {
	t.Helper()
	path := filepath.Join("testdata", "serial_fixtures", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func TestBackendTargetName_AppSelector(t *testing.T) {
	m := config.Mapping{Target: config.TargetApp, Selector: &config.Selector{Kind: config.SelectorExe, Value: "discord"}}
	if got := backendTargetName(m); got != "exe:discord" {
		t.Fatalf("backendTargetName()=%q want %q", got, "exe:discord")
	}
}

func TestBackendTargetName_GroupSelectors(t *testing.T) {
	m := config.Mapping{Target: config.TargetGroup, Selectors: []config.Selector{{Kind: config.SelectorExact, Value: "Discord"}, {Kind: config.SelectorExe, Value: "spotify"}}}
	got := backendTargetName(m)
	var selectors []config.Selector
	if err := json.Unmarshal([]byte(got), &selectors); err != nil {
		t.Fatalf("backendTargetName() produced invalid json: %v", err)
	}
	if len(selectors) != 2 || selectors[1].Kind != config.SelectorExe {
		t.Fatalf("unexpected selectors payload: %#v", selectors)
	}
}

func TestRunSessionFromChannels_ProtocolMismatchBestEffort(t *testing.T) {
	cfg := &config.Config{Mappings: []config.Mapping{{Knob: 1, Target: config.TargetMasterOut, Step: 0.02}}}
	backend := &fakeBackend{}
	metrics := runtime.NewMetrics()

	lineCh := make(chan string)
	readErrCh := make(chan error, 1)
	go func() {
		defer close(lineCh)
		lineCh <- "V:2"
		lineCh <- "E1:+1"
		lineCh <- "B1:1"
		readErrCh <- nil
	}()

	if err := runSessionFromChannels(context.Background(), cfg, backend, metrics, lineCh, readErrCh); err != nil {
		t.Fatalf("runSessionFromChannels returned error: %v", err)
	}

	if len(backend.calls) != 2 {
		t.Fatalf("expected 2 backend calls, got %d", len(backend.calls))
	}
	if backend.calls[0].op != "adjust" || backend.calls[1].op != "toggle" {
		t.Fatalf("unexpected backend call sequence: %+v", backend.calls)
	}
}
