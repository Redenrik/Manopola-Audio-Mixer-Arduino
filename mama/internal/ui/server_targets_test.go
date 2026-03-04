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
	targets []audio.DiscoveredTarget
	states  map[string]audio.TargetState
	err     error
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
		Known      []string                 `json:"known"`
		Supported  []string                 `json:"supported"`
		Discovered []audio.DiscoveredTarget `json:"discovered"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(body.Known) == 0 {
		t.Fatalf("known targets should not be empty")
	}
	if len(body.Supported) != 1 || body.Supported[0] != "system:master_out" {
		t.Fatalf("supported = %v, want [system:master_out]", body.Supported)
	}
	if len(body.Discovered) != 1 || body.Discovered[0].Type != config.TargetMasterOut {
		t.Fatalf("discovered = %+v, want one master_out target", body.Discovered)
	}
}
