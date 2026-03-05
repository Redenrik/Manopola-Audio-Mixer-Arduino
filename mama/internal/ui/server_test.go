package ui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mama/internal/config"
	"mama/internal/mixer"
)

func TestHandlePortTest_MethodNotAllowed(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodGet, "/api/port-test", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandlePortTest_InvalidJSON(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodPost, "/api/port-test", strings.NewReader("{"))
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandlePortTest_MissingPort(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodPost, "/api/port-test", strings.NewReader(`{"baud":115200}`))
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandlePortAutoDetect_MethodNotAllowed(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodPost, "/api/port-autodetect", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandlePortAutoDetect_DetectsFirstMatchingPort(t *testing.T) {
	s := New("config.yaml")
	s.listPorts = func() ([]string, error) {
		return []string{"COM1", "COM3"}, nil
	}
	s.probeProtocolHello = func(port string, baud int, _ time.Duration) (int, error) {
		if baud != 115200 {
			t.Fatalf("unexpected baud: %d", baud)
		}
		if port == "COM3" {
			return 1, nil
		}
		return 0, errors.New("no MAMA protocol hello detected")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/port-autodetect?baud=115200", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var body struct {
		Detected        string `json:"detected"`
		ProtocolVersion int    `json:"protocolVersion"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Detected != "COM3" {
		t.Fatalf("detected = %q, want COM3", body.Detected)
	}
	if body.ProtocolVersion != 1 {
		t.Fatalf("protocolVersion = %d, want 1", body.ProtocolVersion)
	}
}

func TestHandlePortAutoDetect_NoMatch(t *testing.T) {
	s := New("config.yaml")
	s.listPorts = func() ([]string, error) {
		return []string{"COM7"}, nil
	}
	s.probeProtocolHello = func(port string, baud int, _ time.Duration) (int, error) {
		if port != "COM7" || baud != 115200 {
			t.Fatalf("unexpected probe input: port=%s baud=%d", port, baud)
		}
		return 0, errors.New("no MAMA protocol hello detected")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/port-autodetect?baud=115200", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var body struct {
		Detected string `json:"detected"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Detected != "" {
		t.Fatalf("detected = %q, want empty", body.Detected)
	}
}

func TestHandlePortAutoDetect_UsesConnectedRuntimePort(t *testing.T) {
	s := New("config.yaml")
	s.listPorts = func() ([]string, error) {
		return []string{"COM1", "COM7"}, nil
	}
	probeCalls := 0
	s.probeProtocolHello = func(_ string, _ int, _ time.Duration) (int, error) {
		probeCalls++
		return 0, errors.New("probe should not run when runtime is already connected")
	}
	s.runtimeSnapshot = func() (config.SerialCfg, mixer.Status, bool) {
		return config.SerialCfg{Port: "COM9", Baud: 115200}, mixer.Status{
			State:           "connected",
			Connected:       true,
			ProtocolVersion: 1,
		}, true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/port-autodetect?baud=9600", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var body struct {
		Detected        string `json:"detected"`
		Baud            int    `json:"baud"`
		ProtocolVersion int    `json:"protocolVersion"`
		Attempts        []struct {
			Port   string `json:"port"`
			Ok     bool   `json:"ok"`
			Source string `json:"source"`
		} `json:"attempts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Detected != "COM9" {
		t.Fatalf("detected = %q, want COM9", body.Detected)
	}
	if body.Baud != 115200 {
		t.Fatalf("baud = %d, want 115200", body.Baud)
	}
	if body.ProtocolVersion != 1 {
		t.Fatalf("protocolVersion = %d, want 1", body.ProtocolVersion)
	}
	if probeCalls != 0 {
		t.Fatalf("probe calls = %d, want 0", probeCalls)
	}
	if len(body.Attempts) == 0 || body.Attempts[0].Source != "runtime" || !body.Attempts[0].Ok {
		t.Fatalf("expected runtime-backed successful attempt, got %+v", body.Attempts)
	}
}

func TestHandlePortAutoDetect_RuntimeConnectedWithoutProtocolFallsBackToProbe(t *testing.T) {
	s := New("config.yaml")
	s.listPorts = func() ([]string, error) {
		return []string{"COM1", "COM3"}, nil
	}
	probeCalls := 0
	s.probeProtocolHello = func(port string, baud int, _ time.Duration) (int, error) {
		probeCalls++
		if baud != 115200 {
			t.Fatalf("unexpected baud: %d", baud)
		}
		if port == "COM3" {
			return 1, nil
		}
		return 0, errors.New("no MAMA protocol hello detected")
	}
	s.runtimeSnapshot = func() (config.SerialCfg, mixer.Status, bool) {
		return config.SerialCfg{Port: "COM9", Baud: 115200}, mixer.Status{
			State:           "connected",
			Connected:       true,
			ProtocolVersion: 0,
		}, true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/port-autodetect?baud=115200", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var body struct {
		Detected string `json:"detected"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Detected != "COM3" {
		t.Fatalf("detected = %q, want COM3", body.Detected)
	}
	if probeCalls != 2 {
		t.Fatalf("probe calls = %d, want 2", probeCalls)
	}
}

func TestHandleIndex_HasCoreSetupControls(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, `<h2 id="wizardHeading">`) {
		t.Fatalf("did not expect wizard section in index html")
	}
	if strings.Contains(body, `id="wizardStart"`) {
		t.Fatalf("did not expect wizard controls in index html")
	}
	if !strings.Contains(body, "templateSelect") {
		t.Fatalf("expected template selector in index html")
	}
	if !strings.Contains(body, "languageSelect") {
		t.Fatalf("expected language selector in index html")
	}
	if !strings.Contains(body, "const i18n =") {
		t.Fatalf("expected localization dictionary in index html")
	}
	if !strings.Contains(body, "applyTemplate") {
		t.Fatalf("expected template apply control in index html")
	}
	if !strings.Contains(body, "backupConfig") {
		t.Fatalf("expected backup config control in index html")
	}
	if !strings.Contains(body, "restoreConfig") {
		t.Fatalf("expected restore config control in index html")
	}
	if !strings.Contains(body, "exportConfig") {
		t.Fatalf("expected export config control in index html")
	}
	if !strings.Contains(body, "importConfig") {
		t.Fatalf("expected import config control in index html")
	}
	if !strings.Contains(body, "role=\"status\" aria-live=\"polite\"") {
		t.Fatalf("expected aria-live status regions in index html")
	}
	if !strings.Contains(body, "remove.setAttribute(\"aria-label\", \"Remove mapping row\")") {
		t.Fatalf("expected accessible mapping remove control label in index html")
	}
	if !strings.Contains(body, "input:focus-visible") {
		t.Fatalf("expected keyboard focus-visible styling in index html")
	}
}

func TestHandleStartup_Get(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodGet, "/api/startup", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandleStartup_PostUnsupportedOnNonWindows(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodPost, "/api/startup", strings.NewReader(`{"enabled":true}`))
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleRuntime_NoService(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodGet, "/api/runtime", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	var body struct {
		Running bool `json:"running"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Running {
		t.Fatalf("expected running=false without service")
	}
}

func TestHandleRuntime_WithService(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(config.DefaultYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	service, err := mixer.NewService(cfg, &fakeTargetsBackend{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	s := New(cfgPath)
	s.SetMixerService(service)
	req := httptest.NewRequest(http.MethodGet, "/api/runtime", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	var body struct {
		Running bool `json:"running"`
		Status  struct {
			State string `json:"state"`
		} `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !body.Running {
		t.Fatalf("expected running=true with service")
	}
	if body.Status.State == "" {
		t.Fatalf("expected runtime status state to be populated")
	}
}

func TestHandleMappingStatus_ReturnsActiveMappings(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(config.DefaultYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	s := New(cfgPath)
	s.backend = &fakeTargetsBackend{}
	req := httptest.NewRequest(http.MethodGet, "/api/mapping-status", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var body struct {
		Mappings []struct {
			Knob      int    `json:"knob"`
			Target    string `json:"target"`
			Available bool   `json:"available"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(body.Mappings) != 5 {
		t.Fatalf("expected 5 mappings, got %d", len(body.Mappings))
	}
	if body.Mappings[0].Knob != 1 || body.Mappings[0].Target != "master_out" {
		t.Fatalf("unexpected first mapping: %+v", body.Mappings[0])
	}
}
