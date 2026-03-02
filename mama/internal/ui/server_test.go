package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestHandleIndex_IncludesWizard(t *testing.T) {
	s := New("config.yaml")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "First-Run Wizard") {
		t.Fatalf("expected wizard section in index html")
	}
	if !strings.Contains(body, "wizardStart") {
		t.Fatalf("expected wizard controls in index html")
	}
	if !strings.Contains(body, "templateSelect") {
		t.Fatalf("expected template selector in index html")
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
}
