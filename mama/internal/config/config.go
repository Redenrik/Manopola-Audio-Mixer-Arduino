package config

import (
	"fmt"
	"math"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type TargetType string

const (
	TargetMasterOut TargetType = "master_out"
	TargetMicIn     TargetType = "mic_in"
	TargetLineIn    TargetType = "line_in"
	TargetApp       TargetType = "app"   // per process
	TargetGroup     TargetType = "group" // group of apps/processes
)

type Mapping struct {
	Knob   int        `yaml:"knob"`
	Target TargetType `yaml:"target"`
	Name   string     `yaml:"name,omitempty"` // for app/group identification
	Step   float64    `yaml:"step"`           // e.g. 0.02 = 2% per detent
}

type SerialCfg struct {
	Port string `yaml:"port"`
	Baud int    `yaml:"baud"`
}

type Config struct {
	Serial   SerialCfg `yaml:"serial"`
	Mappings []Mapping `yaml:"mappings"`
	Debug    bool      `yaml:"debug"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}

	c.Serial.Port = strings.TrimSpace(c.Serial.Port)
	if c.Serial.Baud == 0 {
		c.Serial.Baud = 115200
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

func (c Config) MappingForKnob(knob int) (Mapping, bool) {
	for _, m := range c.Mappings {
		if m.Knob == knob {
			return m, true
		}
	}
	return Mapping{}, false
}
