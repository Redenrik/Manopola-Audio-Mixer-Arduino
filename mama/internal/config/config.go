package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultBaud = 115200
)

var defaultMappings = []Mapping{
	{Knob: 1, Target: TargetMasterOut, Step: 0.02},
	{Knob: 2, Target: TargetMasterOut, Step: 0.02},
	{Knob: 3, Target: TargetMasterOut, Step: 0.02},
	{Knob: 4, Target: TargetMasterOut, Step: 0.02},
	{Knob: 5, Target: TargetMasterOut, Step: 0.02},
}

const DefaultYAML = `serial:
  port: "COM3"     # Windows example. On Linux: /dev/ttyACM0 or /dev/ttyUSB0
  baud: 115200

debug: true

# Current backend support:
# - implemented: master_out
# - placeholders for future releases: mic_in, line_in, app, group
# Keep default mappings on master_out so first run works end-to-end.
# step = volume change per encoder "click" (0.01 = 1%, 0.02 = 2%, etc.)
mappings:
  - knob: 1
    target: master_out
    step: 0.02

  - knob: 2
    target: master_out
    step: 0.02

  - knob: 3
    target: master_out
    step: 0.02

  - knob: 4
    target: master_out
    step: 0.02

  - knob: 5
    target: master_out
    step: 0.02
`

type TargetType string

const (
	TargetMasterOut TargetType = "master_out"
	TargetMicIn     TargetType = "mic_in"
	TargetLineIn    TargetType = "line_in"
	TargetApp       TargetType = "app"   // per process
	TargetGroup     TargetType = "group" // group of apps/processes
)

type Mapping struct {
	Knob   int        `yaml:"knob" json:"knob"`
	Target TargetType `yaml:"target" json:"target"`
	Name   string     `yaml:"name,omitempty" json:"name,omitempty"` // for app/group identification
	Step   float64    `yaml:"step" json:"step"`                     // e.g. 0.02 = 2% per detent
}

type SerialCfg struct {
	Port string `yaml:"port" json:"port"`
	Baud int    `yaml:"baud" json:"baud"`
}

type Config struct {
	Serial   SerialCfg `yaml:"serial" json:"serial"`
	Mappings []Mapping `yaml:"mappings" json:"mappings"`
	Debug    bool      `yaml:"debug" json:"debug"`
}

type rawConfig struct {
	Serial      SerialCfg    `yaml:"serial"`
	Mappings    []rawMapping `yaml:"mappings"`
	Debug       *bool        `yaml:"debug"`
	LegacyPort  string       `yaml:"port"`
	LegacyBaud  int          `yaml:"baud"`
	LegacyKnobs []rawMapping `yaml:"knobs"`
}

type rawMapping struct {
	Knob       int        `yaml:"knob"`
	Target     TargetType `yaml:"target"`
	Name       string     `yaml:"name"`
	Step       float64    `yaml:"step"`
	LegacyID   int        `yaml:"id"`
	LegacyType TargetType `yaml:"type"`
	LegacyApp  string     `yaml:"app"`
	LegacyStep float64    `yaml:"volume_step"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	return ParseYAML(b)
}

func ParseYAML(b []byte) (Config, error) {
	var raw rawConfig
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return Config{}, err
	}

	c := migrateRawConfig(raw)

	return Validate(c)
}

func migrateRawConfig(raw rawConfig) Config {
	c := Config{
		Serial: raw.Serial,
		Debug:  raw.Debug != nil && *raw.Debug,
	}

	if c.Serial.Port == "" {
		c.Serial.Port = raw.LegacyPort
	}
	if c.Serial.Baud == 0 {
		c.Serial.Baud = raw.LegacyBaud
	}

	mappings := raw.Mappings
	if len(mappings) == 0 {
		mappings = raw.LegacyKnobs
	}

	c.Mappings = make([]Mapping, 0, len(mappings))
	for _, rm := range mappings {
		m := Mapping{
			Knob:   rm.Knob,
			Target: rm.Target,
			Name:   rm.Name,
			Step:   rm.Step,
		}

		if m.Knob == 0 {
			m.Knob = rm.LegacyID
		}
		if m.Target == "" {
			m.Target = rm.LegacyType
		}
		if m.Name == "" {
			m.Name = rm.LegacyApp
		}
		if m.Step == 0 {
			m.Step = rm.LegacyStep
		}

		c.Mappings = append(c.Mappings, m)
	}

	return c
}

func Validate(c Config) (Config, error) {
	c.Serial.Port = strings.TrimSpace(c.Serial.Port)
	if c.Serial.Baud == 0 {
		c.Serial.Baud = DefaultBaud
	}
	if c.Serial.Baud < 0 {
		return Config{}, fmt.Errorf("serial.baud must be > 0")
	}
	if c.Serial.Port == "" {
		return Config{}, fmt.Errorf("serial.port must be set")
	}
	if len(c.Mappings) == 0 {
		return Config{}, fmt.Errorf("at least one mapping is required")
	}

	seenKnobs := make(map[int]struct{}, len(c.Mappings))
	for i := range c.Mappings {
		m := &c.Mappings[i]

		m.Name = strings.TrimSpace(m.Name)

		if m.Knob <= 0 {
			return Config{}, fmt.Errorf("invalid knob id: %d", m.Knob)
		}
		if _, exists := seenKnobs[m.Knob]; exists {
			return Config{}, fmt.Errorf("duplicate mapping for knob %d", m.Knob)
		}
		seenKnobs[m.Knob] = struct{}{}

		switch m.Target {
		case TargetMasterOut, TargetMicIn, TargetLineIn, TargetApp, TargetGroup:
			// valid
		default:
			return Config{}, fmt.Errorf("invalid target for knob %d: %q", m.Knob, m.Target)
		}

		switch m.Target {
		case TargetApp, TargetGroup:
			if m.Name == "" {
				return Config{}, fmt.Errorf("target %q for knob %d requires name", m.Target, m.Knob)
			}
		default:
			if m.Name != "" {
				return Config{}, fmt.Errorf("target %q for knob %d must not set name", m.Target, m.Knob)
			}
		}

		if math.IsNaN(m.Step) || math.IsInf(m.Step, 0) || m.Step <= 0 || m.Step > 1 {
			return Config{}, fmt.Errorf("invalid step for knob %d: %f", m.Knob, m.Step)
		}
	}

	return c, nil
}

func Save(path string, c Config) error {
	c, err := Validate(c)
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o644)
}

func EnsureDefaultFile(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false, err
		}
	}

	if err := os.WriteFile(path, []byte(DefaultYAML), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func Default() Config {
	mappings := make([]Mapping, len(defaultMappings))
	copy(mappings, defaultMappings)

	return Config{
		Serial: SerialCfg{
			Port: "COM3",
			Baud: DefaultBaud,
		},
		Debug:    true,
		Mappings: mappings,
	}
}

func KnownTargets() []TargetType {
	return []TargetType{
		TargetMasterOut,
		TargetMicIn,
		TargetLineIn,
		TargetApp,
		TargetGroup,
	}
}

func ResolveDefaultPath() string {
	const localDefault = "config.yaml"
	if _, err := os.Stat(localDefault); err == nil {
		return localDefault
	}

	const sourceDefault = "internal/config/default.yaml"
	if _, err := os.Stat(sourceDefault); err == nil {
		return sourceDefault
	}

	// Release artifacts can run portable with a side-by-side config.
	return localDefault
}

func (c Config) MappingForKnob(knob int) (Mapping, bool) {
	for _, m := range c.Mappings {
		if m.Knob == knob {
			return m, true
		}
	}
	return Mapping{}, false
}
