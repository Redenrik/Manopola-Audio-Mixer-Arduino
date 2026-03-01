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
