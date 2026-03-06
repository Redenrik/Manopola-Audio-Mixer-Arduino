package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mama/internal/audio"
	"mama/internal/config"
)

type fakeTargetsBackend struct {
	targets        []audio.DiscoveredTarget
	states         map[string]audio.TargetState
	supportedTypes []config.TargetType
	err            error
}

func (f *fakeTargetsBackend) Adjust(target config.TargetType, name string, step float64, deltaSteps int) error {
	return nil
}

func (f *fakeTargetsBackend) ToggleMute(target config.TargetType, name string) error {
	return nil
}

func (f *fakeTargetsBackend) ReadState(target config.TargetType, name string) (audio.TargetState, error) {
	if f.states != nil {
		key := string(target) + "|" + name
		if state, ok := f.states[key]; ok {
			return state, nil
		}
	}
	return audio.TargetState{
		Available: true,
		Volume:    50,
		Muted:     false,
	}, nil
}

func (f *fakeTargetsBackend) ListTargets() ([]audio.DiscoveredTarget, error) {
	return f.targets, f.err
}

func (f *fakeTargetsBackend) SupportedTargetTypes() []config.TargetType {
	return append([]config.TargetType(nil), f.supportedTypes...)
}

func TestHandleTargetsIncludesDiscoveredTargets(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(config.DefaultYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	s := New(cfgPath)
	s.backend = &fakeTargetsBackend{targets: []audio.DiscoveredTarget{{
		ID:   "system:master_out",
		Type: config.TargetMasterOut,
		Name: "System Output",
	}}}

	req := httptest.NewRequest(http.MethodGet, "/api/targets", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body struct {
		Known          []string                 `json:"known"`
		AvailableTypes []string                 `json:"availableTypes"`
		Supported      []string                 `json:"supported"`
		Discovered     []audio.DiscoveredTarget `json:"discovered"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(body.Known) == 0 {
		t.Fatalf("known targets should not be empty")
	}
	if len(body.AvailableTypes) == 0 {
		t.Fatalf("availableTypes should not be empty")
	}
	if len(body.Supported) != 1 || body.Supported[0] != "system:master_out" {
		t.Fatalf("supported = %v, want [system:master_out]", body.Supported)
	}
	if len(body.Discovered) != 1 || body.Discovered[0].Type != config.TargetMasterOut {
		t.Fatalf("discovered = %+v, want one master_out target", body.Discovered)
	}
}

func TestHandleTargets_UsesBackendAvailableTypes(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(config.DefaultYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	s := New(cfgPath)
	s.backend = &fakeTargetsBackend{
		supportedTypes: []config.TargetType{config.TargetMasterOut, config.TargetMicIn},
		targets: []audio.DiscoveredTarget{
			{ID: "system:master_out", Type: config.TargetMasterOut, Name: "System Output"},
			{ID: "system:mic_in", Type: config.TargetMicIn, Name: "System Microphone"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/targets", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body struct {
		AvailableTypes []string `json:"availableTypes"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(body.AvailableTypes) != 2 {
		t.Fatalf("availableTypes = %v, want 2 entries", body.AvailableTypes)
	}
	if body.AvailableTypes[0] != string(config.TargetMasterOut) || body.AvailableTypes[1] != string(config.TargetMicIn) {
		t.Fatalf("availableTypes = %v, want [master_out mic_in]", body.AvailableTypes)
	}
}

func TestHandleTargets_IncludesAppSessionDiagnostics(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(config.DefaultYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	s := New(cfgPath)
	s.backend = &fakeTargetsBackend{
		targets: []audio.DiscoveredTarget{
			{ID: "system:master_out", Type: config.TargetMasterOut, Name: "System Output"},
			{
				ID:   "app:1",
				Type: config.TargetApp,
				Name: "Discord",
				Capabilities: &audio.TargetCapabilities{
					Volume: boolPtr(true),
					Mute:   boolPtr(true),
				},
			},
			{
				ID:   "app:2",
				Type: config.TargetApp,
				Name: "Spotify",
				Capabilities: &audio.TargetCapabilities{
					Volume: boolPtr(false),
					Mute:   boolPtr(true),
				},
			},
			{
				ID:   "app:3",
				Type: config.TargetApp,
				Name: "OBS",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/targets", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body struct {
		Diagnostics struct {
			AppSessions struct {
				Total               int `json:"total"`
				WithCapabilities    int `json:"withCapabilities"`
				UnknownCapabilities int `json:"unknownCapabilities"`
				VolumeSupported     int `json:"volumeSupported"`
				MuteSupported       int `json:"muteSupported"`
				FullySupported      int `json:"fullySupported"`
				WriteUnsupported    int `json:"writeUnsupported"`
			} `json:"appSessions"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	got := body.Diagnostics.AppSessions
	if got.Total != 3 || got.WithCapabilities != 2 || got.UnknownCapabilities != 1 || got.VolumeSupported != 1 || got.MuteSupported != 2 || got.FullySupported != 1 || got.WriteUnsupported != 1 {
		t.Fatalf("unexpected app session diagnostics: %+v", got)
	}
}

func boolPtr(v bool) *bool {
	b := v
	return &b
}
