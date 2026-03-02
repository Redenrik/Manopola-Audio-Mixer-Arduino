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
# - implemented: app (Unix hosts with pactl sink-input control)
# - placeholder for future releases: group
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
	Knob      int        `yaml:"knob" json:"knob"`
	Target    TargetType `yaml:"target" json:"target"`
	Name      string     `yaml:"name,omitempty" json:"name,omitempty"` // legacy alias; migrated to selector(s) for app/group
	Selector  *Selector  `yaml:"selector,omitempty" json:"selector,omitempty"`
	Selectors []Selector `yaml:"selectors,omitempty" json:"selectors,omitempty"`
	Priority  int        `yaml:"priority,omitempty" json:"priority,omitempty"`
	Step      float64    `yaml:"step" json:"step"` // e.g. 0.02 = 2% per detent
}

type SelectorKind string

const (
	SelectorExact    SelectorKind = "exact"
	SelectorContains SelectorKind = "contains"
	SelectorPrefix   SelectorKind = "prefix"
	SelectorSuffix   SelectorKind = "suffix"
	SelectorGlob     SelectorKind = "glob"
	SelectorExe      SelectorKind = "exe"
)

type Selector struct {
	Kind  SelectorKind `yaml:"kind" json:"kind"`
	Value string       `yaml:"value" json:"value"`
}

type SerialCfg struct {
	Port string `yaml:"port" json:"port"`
	Baud int    `yaml:"baud" json:"baud"`
}

type Config struct {
	Serial        SerialCfg `yaml:"serial" json:"serial"`
	Mappings      []Mapping `yaml:"mappings" json:"mappings"`
	Profiles      []Profile `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	ActiveProfile string    `yaml:"active_profile,omitempty" json:"active_profile,omitempty"`
	Debug         bool      `yaml:"debug" json:"debug"`
}

type Profile struct {
	Name     string    `yaml:"name" json:"name"`
	Mappings []Mapping `yaml:"mappings" json:"mappings"`
}

type rawConfig struct {
	Serial        SerialCfg    `yaml:"serial"`
	Mappings      []rawMapping `yaml:"mappings"`
	Profiles      []rawProfile `yaml:"profiles"`
	ActiveProfile string       `yaml:"active_profile"`
	Debug         *bool        `yaml:"debug"`
	LegacyPort    string       `yaml:"port"`
	LegacyBaud    int          `yaml:"baud"`
	LegacyKnobs   []rawMapping `yaml:"knobs"`
}

type rawProfile struct {
	Name     string       `yaml:"name"`
	Mappings []rawMapping `yaml:"mappings"`
}

type rawMapping struct {
	Knob       int        `yaml:"knob"`
	Target     TargetType `yaml:"target"`
	Name       string     `yaml:"name"`
	Selector   *Selector  `yaml:"selector"`
	Selectors  []Selector `yaml:"selectors"`
	Step       float64    `yaml:"step"`
	Priority   int        `yaml:"priority"`
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
		Serial:        raw.Serial,
		Profiles:      make([]Profile, 0, len(raw.Profiles)),
		ActiveProfile: raw.ActiveProfile,
		Debug:         raw.Debug != nil && *raw.Debug,
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
		c.Mappings = append(c.Mappings, migrateRawMapping(rm))
	}

	for _, rp := range raw.Profiles {
		profile := Profile{
			Name:     rp.Name,
			Mappings: make([]Mapping, 0, len(rp.Mappings)),
		}
		for _, rm := range rp.Mappings {
			profile.Mappings = append(profile.Mappings, migrateRawMapping(rm))
		}
		c.Profiles = append(c.Profiles, profile)
	}

	return c
}

func migrateRawMapping(rm rawMapping) Mapping {
	m := Mapping{
		Knob:      rm.Knob,
		Target:    rm.Target,
		Name:      rm.Name,
		Selector:  rm.Selector,
		Selectors: rm.Selectors,
		Priority:  rm.Priority,
		Step:      rm.Step,
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

	return m
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
	if len(c.Mappings) == 0 && len(c.Profiles) == 0 {
		return Config{}, fmt.Errorf("at least one mapping is required")
	}

	if err := validateMappingSet(c.Mappings); err != nil {
		return Config{}, err
	}

	profileNames := make(map[string]struct{}, len(c.Profiles))
	for i := range c.Profiles {
		p := &c.Profiles[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			return Config{}, fmt.Errorf("profile %d: name must be set", i+1)
		}
		if _, exists := profileNames[strings.ToLower(p.Name)]; exists {
			return Config{}, fmt.Errorf("duplicate profile name %q", p.Name)
		}
		profileNames[strings.ToLower(p.Name)] = struct{}{}

		if len(p.Mappings) == 0 {
			return Config{}, fmt.Errorf("profile %q must define at least one mapping", p.Name)
		}
		if err := validateMappingSet(p.Mappings); err != nil {
			return Config{}, fmt.Errorf("profile %q: %w", p.Name, err)
		}
	}

	c.ActiveProfile = strings.TrimSpace(c.ActiveProfile)
	if len(c.Profiles) > 0 {
		if c.ActiveProfile == "" {
			c.ActiveProfile = c.Profiles[0].Name
		}
		if _, exists := profileNames[strings.ToLower(c.ActiveProfile)]; !exists {
			return Config{}, fmt.Errorf("active_profile %q does not match any defined profile", c.ActiveProfile)
		}
	} else {
		c.ActiveProfile = ""
	}

	return c, nil
}

func validateMappingSet(mappings []Mapping) error {
	seenKnobs := make(map[int]struct{}, len(mappings))
	for i := range mappings {
		m := &mappings[i]

		m.Name = strings.TrimSpace(m.Name)
		if m.Selector != nil {
			m.Selector.Value = strings.TrimSpace(m.Selector.Value)
		}
		for j := range m.Selectors {
			m.Selectors[j].Value = strings.TrimSpace(m.Selectors[j].Value)
		}

		if m.Knob <= 0 {
			return fmt.Errorf("invalid knob id: %d", m.Knob)
		}
		if _, exists := seenKnobs[m.Knob]; exists {
			return fmt.Errorf("duplicate mapping for knob %d", m.Knob)
		}
		seenKnobs[m.Knob] = struct{}{}

		switch m.Target {
		case TargetMasterOut, TargetMicIn, TargetLineIn, TargetApp, TargetGroup:
			// valid
		default:
			return fmt.Errorf("invalid target for knob %d: %q", m.Knob, m.Target)
		}

		if err := validateMappingSelectors(m); err != nil {
			return fmt.Errorf("knob %d: %w", m.Knob, err)
		}

		if math.IsNaN(m.Step) || math.IsInf(m.Step, 0) || m.Step <= 0 || m.Step > 1 {
			return fmt.Errorf("invalid step for knob %d: %f", m.Knob, m.Step)
		}
		if m.Priority < 0 {
			return fmt.Errorf("invalid priority for knob %d: %d", m.Knob, m.Priority)
		}
	}

	if err := validateMappingPrecedence(mappings); err != nil {
		return err
	}

	return nil
}

func validateMappingPrecedence(mappings []Mapping) error {
	for i := 0; i < len(mappings); i++ {
		for j := i + 1; j < len(mappings); j++ {
			a := mappings[i]
			b := mappings[j]

			if !isPrecedenceTarget(a.Target) || !isPrecedenceTarget(b.Target) || a.Target != b.Target {
				continue
			}
			if !mappingsOverlap(a, b) {
				continue
			}

			if precedenceScore(a) == precedenceScore(b) {
				return fmt.Errorf("ambiguous precedence between knob %d and knob %d for target %q; set distinct priority values", a.Knob, b.Knob, a.Target)
			}
		}
	}

	return nil
}

func isPrecedenceTarget(target TargetType) bool {
	return target == TargetApp || target == TargetGroup
}

func mappingsOverlap(a, b Mapping) bool {
	switch a.Target {
	case TargetApp:
		if a.Selector == nil || b.Selector == nil {
			return false
		}
		return selectorsEqual(*a.Selector, *b.Selector)
	case TargetGroup:
		for _, left := range a.Selectors {
			for _, right := range b.Selectors {
				if selectorsEqual(left, right) {
					return true
				}
			}
		}
	}

	return false
}

func selectorsEqual(a, b Selector) bool {
	return a.Kind == b.Kind && strings.EqualFold(strings.TrimSpace(a.Value), strings.TrimSpace(b.Value))
}

func precedenceScore(m Mapping) int {
	return (m.Priority * 1000) + mappingSpecificity(m)
}

func mappingSpecificity(m Mapping) int {
	switch m.Target {
	case TargetApp:
		if m.Selector == nil {
			return 0
		}
		return selectorSpecificity(*m.Selector)
	case TargetGroup:
		max := 0
		for _, selector := range m.Selectors {
			if score := selectorSpecificity(selector); score > max {
				max = score
			}
		}
		return max
	default:
		return 0
	}
}

func selectorSpecificity(selector Selector) int {
	length := len(strings.TrimSpace(selector.Value))
	if length > 100 {
		length = 100
	}

	switch selector.Kind {
	case SelectorExact:
		return 600 + length
	case SelectorExe:
		return 500 + length
	case SelectorPrefix, SelectorSuffix:
		return 400 + length
	case SelectorContains:
		return 300 + length
	case SelectorGlob:
		return 200 + length
	default:
		return length
	}
}

func validateMappingSelectors(m *Mapping) error {
	switch m.Target {
	case TargetApp:
		if len(m.Selectors) > 0 {
			return fmt.Errorf("target %q must use selector, not selectors", m.Target)
		}
		if m.Selector == nil {
			if m.Name == "" {
				return fmt.Errorf("target %q requires selector (or legacy name)", m.Target)
			}
			m.Selector = &Selector{Kind: SelectorExact, Value: m.Name}
		}
		if m.Name != "" {
			m.Name = ""
		}
		return validateSelector(*m.Selector)
	case TargetGroup:
		if m.Selector != nil {
			return fmt.Errorf("target %q must use selectors array", m.Target)
		}
		if len(m.Selectors) == 0 {
			if m.Name == "" {
				return fmt.Errorf("target %q requires selectors (or legacy name)", m.Target)
			}
			m.Selectors = []Selector{{Kind: SelectorExact, Value: m.Name}}
		}
		if m.Name != "" {
			m.Name = ""
		}
		for _, selector := range m.Selectors {
			if err := validateSelector(selector); err != nil {
				return err
			}
		}
		return nil
	default:
		if m.Name != "" || m.Selector != nil || len(m.Selectors) > 0 {
			return fmt.Errorf("target %q must not set name/selector(s)", m.Target)
		}
		return nil
	}
}

func validateSelector(selector Selector) error {
	selector.Value = strings.TrimSpace(selector.Value)
	if selector.Value == "" {
		return fmt.Errorf("selector value must not be empty")
	}

	switch selector.Kind {
	case SelectorExact, SelectorContains, SelectorPrefix, SelectorSuffix, SelectorGlob, SelectorExe:
		return nil
	default:
		return fmt.Errorf("invalid selector kind %q", selector.Kind)
	}
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
	for _, m := range c.ActiveMappings() {
		if m.Knob == knob {
			return m, true
		}
	}
	return Mapping{}, false
}

func (c Config) ActiveMappings() []Mapping {
	if len(c.Profiles) == 0 {
		return c.Mappings
	}

	active := c.ActiveProfile
	if active == "" {
		active = c.Profiles[0].Name
	}
	for _, p := range c.Profiles {
		if strings.EqualFold(p.Name, active) {
			return p.Mappings
		}
	}

	return c.Mappings
}
