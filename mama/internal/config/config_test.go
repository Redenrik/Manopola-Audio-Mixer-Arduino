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

func TestSaveAndReload(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg.yaml")

	in := Config{
		Serial: SerialCfg{
			Port: " COM7 ",
		},
		Debug: true,
		Mappings: []Mapping{
			{
				Knob:   1,
				Target: TargetApp,
				Name:   "  Discord  ",
				Step:   0.03,
			},
		},
	}

	if err := Save(p, in); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	out, err := Load(p)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if out.Serial.Baud != DefaultBaud {
		t.Fatalf("expected default baud %d, got %d", DefaultBaud, out.Serial.Baud)
	}
	if out.Serial.Port != "COM7" {
		t.Fatalf("expected trimmed port COM7, got %q", out.Serial.Port)
	}
	if out.Mappings[0].Name != "Discord" {
		t.Fatalf("expected trimmed name Discord, got %q", out.Mappings[0].Name)
	}
}

func TestEnsureDefaultFileCreatesWhenMissing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nested", "mama.yaml")

	created, err := EnsureDefaultFile(p)
	if err != nil {
		t.Fatalf("EnsureDefaultFile error: %v", err)
	}
	if !created {
		t.Fatal("expected file to be created")
	}

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("default file should be valid, got error: %v", err)
	}
	if len(cfg.Mappings) == 0 {
		t.Fatal("default file should contain mappings")
	}

	created, err = EnsureDefaultFile(p)
	if err != nil {
		t.Fatalf("EnsureDefaultFile second call error: %v", err)
	}
	if created {
		t.Fatal("expected second call to be no-op")
	}
}
