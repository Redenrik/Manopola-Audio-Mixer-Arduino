package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	return p
}

func TestLoadValid(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
  baud: 115200
mappings:
  - knob: 1
    target: master_out
    step: 0.02
`)
	if _, err := Load(cfgPath); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
}

func TestLoadRejectsDuplicateKnob(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
  baud: 115200
mappings:
  - knob: 1
    target: master_out
    step: 0.02
  - knob: 1
    target: master_out
    step: 0.02
`)
	if _, err := Load(cfgPath); err == nil {
		t.Fatal("expected error for duplicate knob")
	}
}

func TestLoadRejectsGroupWithoutName(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
mappings:
  - knob: 1
    target: group
    step: 0.02
`)
	if _, err := Load(cfgPath); err == nil {
		t.Fatal("expected error for missing group name")
	}
}

func TestLoadRejectsNegativeBaud(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
  baud: -1
mappings:
  - knob: 1
    target: master_out
    step: 0.02
`)
	if _, err := Load(cfgPath); err == nil {
		t.Fatal("expected error for negative baud")
	}
}

func TestLoadRejectsNaNStep(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
mappings:
  - knob: 1
    target: master_out
    step: .nan
`)
	if _, err := Load(cfgPath); err == nil {
		t.Fatal("expected error for NaN step")
	}
}

func TestLoadTrimsFields(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "  COM3  "
mappings:
  - knob: 1
    target: app
    name: "  Discord  "
    step: 0.02
`)
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Serial.Port != "COM3" {
		t.Fatalf("expected trimmed serial port, got %q", cfg.Serial.Port)
	}
	if cfg.Mappings[0].Name != "Discord" {
		t.Fatalf("expected trimmed mapping name, got %q", cfg.Mappings[0].Name)
	}
}
