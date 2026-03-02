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

func TestLoadAcceptsAppSelectorSchema(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
mappings:
  - knob: 1
    target: app
    selector:
      kind: exe
      value: "discord.exe"
    step: 0.02
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mappings[0].Selector == nil {
		t.Fatal("expected selector to be present")
	}
	if cfg.Mappings[0].Selector.Kind != SelectorExe {
		t.Fatalf("expected selector kind %q, got %q", SelectorExe, cfg.Mappings[0].Selector.Kind)
	}
}

func TestLoadAcceptsGroupSelectorsSchema(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
mappings:
  - knob: 1
    target: group
    selectors:
      - kind: prefix
        value: "Spotify"
      - kind: glob
        value: "*Game*"
    step: 0.02
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := len(cfg.Mappings[0].Selectors); got != 2 {
		t.Fatalf("expected 2 selectors, got %d", got)
	}
}

func TestLoadRejectsInvalidSelectorKind(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
mappings:
  - knob: 1
    target: app
    selector:
      kind: regex
      value: ".*discord.*"
    step: 0.02
`)
	if _, err := Load(cfgPath); err == nil {
		t.Fatal("expected error for invalid selector kind")
	}
}

func TestLoadMigratesLegacyNameToSelector(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM3"
mappings:
  - knob: 1
    target: app
    name: "Discord"
    step: 0.02
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mappings[0].Name != "" {
		t.Fatalf("expected legacy name to be cleared after migration, got %q", cfg.Mappings[0].Name)
	}
	if cfg.Mappings[0].Selector == nil {
		t.Fatal("expected selector migrated from legacy name")
	}
	if cfg.Mappings[0].Selector.Kind != SelectorExact || cfg.Mappings[0].Selector.Value != "Discord" {
		t.Fatalf("unexpected migrated selector: %#v", cfg.Mappings[0].Selector)
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
	if cfg.Mappings[0].Selector == nil {
		t.Fatal("expected selector migrated from name")
	}
	if cfg.Mappings[0].Selector.Value != "Discord" {
		t.Fatalf("expected trimmed selector value Discord, got %q", cfg.Mappings[0].Selector.Value)
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
	if out.Mappings[0].Selector == nil {
		t.Fatal("expected selector migrated from name")
	}
	if out.Mappings[0].Selector.Value != "Discord" {
		t.Fatalf("expected trimmed selector value Discord, got %q", out.Mappings[0].Selector.Value)
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

func TestLoadMigratesLegacyTopLevelAndMappingFields(t *testing.T) {
	cfgPath := writeConfig(t, `
port: "COM9"
baud: 57600
debug: true
knobs:
  - id: 1
    type: master_out
    volume_step: 0.05
  - id: 2
    type: app
    app: "Discord"
    volume_step: 0.03
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Serial.Port != "COM9" {
		t.Fatalf("expected migrated serial port COM9, got %q", cfg.Serial.Port)
	}
	if cfg.Serial.Baud != 57600 {
		t.Fatalf("expected migrated serial baud 57600, got %d", cfg.Serial.Baud)
	}
	if !cfg.Debug {
		t.Fatal("expected migrated debug=true")
	}
	if got := len(cfg.Mappings); got != 2 {
		t.Fatalf("expected 2 migrated mappings, got %d", got)
	}
	if cfg.Mappings[1].Selector == nil {
		t.Fatal("expected migrated app alias to selector")
	}
	if cfg.Mappings[1].Selector.Value != "Discord" {
		t.Fatalf("expected migrated app alias to selector value Discord, got %q", cfg.Mappings[1].Selector.Value)
	}
}

func TestLoadPrefersCurrentSchemaOverLegacyAliases(t *testing.T) {
	cfgPath := writeConfig(t, `
serial:
  port: "COM11"
  baud: 115200
port: "COM2"
baud: 9600
mappings:
  - knob: 1
    target: master_out
    step: 0.02
knobs:
  - id: 9
    type: app
    app: "ShouldNotApply"
    volume_step: 0.5
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Serial.Port != "COM11" {
		t.Fatalf("expected current schema serial.port to win, got %q", cfg.Serial.Port)
	}
	if cfg.Serial.Baud != 115200 {
		t.Fatalf("expected current schema serial.baud to win, got %d", cfg.Serial.Baud)
	}
	if got := len(cfg.Mappings); got != 1 {
		t.Fatalf("expected 1 mapping from current schema, got %d", got)
	}
	if cfg.Mappings[0].Knob != 1 {
		t.Fatalf("expected current schema mapping knob 1, got %d", cfg.Mappings[0].Knob)
	}
}
