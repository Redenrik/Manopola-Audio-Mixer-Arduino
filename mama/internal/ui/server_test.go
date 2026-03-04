package ui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
