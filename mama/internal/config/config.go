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

debug: false

# Target availability depends on platform/backend:
# - master_out: available on all supported hosts
# - mic_in / line_in: available when capture endpoints are exposed by the OS
# - app / group: available when per-app audio sessions are available and active
# Keep defaults on master_out so first run works end-to-end everywhere.
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
	Knob             int               `yaml:"knob" json:"knob"`
	Target           TargetType        `yaml:"target" json:"target"`
	Name             string            `yaml:"name,omitempty" json:"name,omitempty"` // app/group legacy alias and optional endpoint selector token for system targets
	Selector         *Selector         `yaml:"selector,omitempty" json:"selector,omitempty"`
	Selectors        []Selector        `yaml:"selectors,omitempty" json:"selectors,omitempty"`
	Priority         int               `yaml:"priority,omitempty" json:"priority,omitempty"`
	Step             float64           `yaml:"step" json:"step"` // e.g. 0.02 = 2% per detent
	Sensitivity      SensitivityPreset `yaml:"sensitivity,omitempty" json:"sensitivity,omitempty"`
	FallbackToMaster bool              `yaml:"fallback_to_master,omitempty" json:"fallback_to_master,omitempty"`
	FallbackTarget   TargetType        `yaml:"fallback_target,omitempty" json:"fallback_target,omitempty"`
	FallbackName     string            `yaml:"fallback_name,omitempty" json:"fallback_name,omitempty"`
}

type SensitivityPreset string

const (
	SensitivitySlow   SensitivityPreset = "slow"
	SensitivityNormal SensitivityPreset = "normal"
	SensitivityFast   SensitivityPreset = "fast"
)

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

type GroupDefinition struct {
	Name      string     `yaml:"name" json:"name"`
	Selectors []Selector `yaml:"selectors" json:"selectors"`
}

type MappingTemplate struct {
	ID       string    `yaml:"id" json:"id"`
	Name     string    `yaml:"name" json:"name"`
	Mappings []Mapping `yaml:"mappings" json:"mappings"`
}

type SerialCfg struct {
	Port string `yaml:"port" json:"port"`
	Baud int    `yaml:"baud" json:"baud"`
}

type Config struct {
	Serial          SerialCfg         `yaml:"serial" json:"serial"`
	Mappings        []Mapping         `yaml:"mappings" json:"mappings"`
	Groups          []GroupDefinition `yaml:"groups,omitempty" json:"groups,omitempty"`
	Templates       []MappingTemplate `yaml:"templates,omitempty" json:"templates,omitempty"`
	DefaultTemplate string            `yaml:"default_template,omitempty" json:"default_template,omitempty"`
	Profiles        []Profile         `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	ActiveProfile   string            `yaml:"active_profile,omitempty" json:"active_profile,omitempty"`
	Debug           bool              `yaml:"debug" json:"debug"`
}

type Profile struct {
	Name     string    `yaml:"name" json:"name"`
	Mappings []Mapping `yaml:"mappings" json:"mappings"`
}

type rawConfig struct {
	Serial          SerialCfg         `yaml:"serial"`
	Mappings        []rawMapping      `yaml:"mappings"`
	Groups          []GroupDefinition `yaml:"groups"`
	Templates       []MappingTemplate `yaml:"templates"`
	DefaultTemplate string            `yaml:"default_template"`
	Profiles        []rawProfile      `yaml:"profiles"`
	ActiveProfile   string            `yaml:"active_profile"`
	Debug           *bool             `yaml:"debug"`
	LegacyPort      string            `yaml:"port"`
	LegacyBaud      int               `yaml:"baud"`
	LegacyKnobs     []rawMapping      `yaml:"knobs"`
}

type rawProfile struct {
	Name     string       `yaml:"name"`
	Mappings []rawMapping `yaml:"mappings"`
}

type rawMapping struct {
	Knob             int               `yaml:"knob"`
	Target           TargetType        `yaml:"target"`
	Name             string            `yaml:"name"`
	Selector         *Selector         `yaml:"selector"`
	Selectors        []Selector        `yaml:"selectors"`
	Step             float64           `yaml:"step"`
	Priority         int               `yaml:"priority"`
	Sensitivity      SensitivityPreset `yaml:"sensitivity"`
	FallbackToMaster bool              `yaml:"fallback_to_master"`
	FallbackTarget   TargetType        `yaml:"fallback_target"`
	FallbackName     string            `yaml:"fallback_name"`
	LegacyID         int               `yaml:"id"`
	LegacyType       TargetType        `yaml:"type"`
	LegacyApp        string            `yaml:"app"`
	LegacyStep       float64           `yaml:"volume_step"`
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
		Serial:          raw.Serial,
		Groups:          append([]GroupDefinition(nil), raw.Groups...),
		Templates:       append([]MappingTemplate(nil), raw.Templates...),
		DefaultTemplate: raw.DefaultTemplate,
		Profiles:        make([]Profile, 0, len(raw.Profiles)),
		ActiveProfile:   raw.ActiveProfile,
		Debug:           raw.Debug != nil && *raw.Debug,
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
		Knob:             rm.Knob,
		Target:           rm.Target,
		Name:             rm.Name,
		Selector:         rm.Selector,
		Selectors:        rm.Selectors,
		Priority:         rm.Priority,
		Step:             rm.Step,
		Sensitivity:      rm.Sensitivity,
		FallbackToMaster: rm.FallbackToMaster,
		FallbackTarget:   rm.FallbackTarget,
		FallbackName:     rm.FallbackName,
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
	if m.Sensitivity == "" {
		m.Sensitivity = SensitivityNormal
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

	if err := validateGroupDefinitions(c.Groups); err != nil {
		return Config{}, err
	}
	if err := validateTemplateDefinitions(c.Templates); err != nil {
		return Config{}, err
	}
	c.DefaultTemplate = strings.TrimSpace(c.DefaultTemplate)

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

func validateGroupDefinitions(groups []GroupDefinition) error {
	if len(groups) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(groups))
	for i := range groups {
		g := &groups[i]
		g.Name = strings.TrimSpace(g.Name)
		if g.Name == "" {
			return fmt.Errorf("group %d: name must be set", i+1)
		}
		key := strings.ToLower(g.Name)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate group name %q", g.Name)
		}
		seen[key] = struct{}{}
		if len(g.Selectors) == 0 {
			return fmt.Errorf("group %q: selectors must not be empty", g.Name)
		}
		for j := range g.Selectors {
			g.Selectors[j].Value = strings.TrimSpace(g.Selectors[j].Value)
			if err := validateSelector(g.Selectors[j]); err != nil {
				return fmt.Errorf("group %q: %w", g.Name, err)
			}
		}
	}
	return nil
}

func validateTemplateDefinitions(templates []MappingTemplate) error {
	if len(templates) == 0 {
		return nil
	}
	seenIDs := make(map[string]struct{}, len(templates))
	for i := range templates {
		tpl := &templates[i]
		tpl.ID = strings.TrimSpace(tpl.ID)
		tpl.Name = strings.TrimSpace(tpl.Name)
		if tpl.ID == "" {
			return fmt.Errorf("template %d: id must be set", i+1)
		}
		if tpl.Name == "" {
			tpl.Name = tpl.ID
		}
		idKey := strings.ToLower(tpl.ID)
		if _, exists := seenIDs[idKey]; exists {
			return fmt.Errorf("duplicate template id %q", tpl.ID)
		}
		seenIDs[idKey] = struct{}{}
		if len(tpl.Mappings) == 0 {
			return fmt.Errorf("template %q must define at least one mapping", tpl.ID)
		}
		if err := validateMappingSet(tpl.Mappings); err != nil {
			return fmt.Errorf("template %q: %w", tpl.ID, err)
		}
	}
	return nil
}

func validateMappingSet(mappings []Mapping) error {
	seenKnobs := make(map[int]struct{}, len(mappings))
	for i := range mappings {
		m := &mappings[i]

		m.Name = strings.TrimSpace(m.Name)
		m.Sensitivity = normalizeSensitivity(m.Sensitivity)
		m.FallbackName = strings.TrimSpace(m.FallbackName)
		if m.FallbackTarget == "" && m.FallbackToMaster {
			m.FallbackTarget = TargetMasterOut
		}
		if m.FallbackTarget != "" {
			// Canonicalize to the flexible schema while remaining backward-compatible on load.
			m.FallbackToMaster = false
		}
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
		if err := validateSensitivity(m.Sensitivity); err != nil {
			return fmt.Errorf("knob %d: %w", m.Knob, err)
		}
		if m.FallbackTarget != "" {
			if m.Target != TargetApp && m.Target != TargetGroup {
				return fmt.Errorf("knob %d: fallback is allowed only for target app/group", m.Knob)
			}
			switch m.FallbackTarget {
			case TargetMasterOut, TargetMicIn, TargetLineIn, TargetApp, TargetGroup:
				// valid
			default:
				return fmt.Errorf("knob %d: invalid fallback_target %q", m.Knob, m.FallbackTarget)
			}
			if (m.FallbackTarget == TargetApp || m.FallbackTarget == TargetGroup) && m.FallbackName == "" {
				return fmt.Errorf("knob %d: fallback_name is required for fallback_target %q", m.Knob, m.FallbackTarget)
			}
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

func (m Mapping) EffectiveFallback() (TargetType, string, bool) {
	if m.FallbackTarget != "" {
		return m.FallbackTarget, strings.TrimSpace(m.FallbackName), true
	}
	if m.FallbackToMaster {
		return TargetMasterOut, "", true
	}
	return "", "", false
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
		if m.Selector != nil || len(m.Selectors) > 0 {
			return fmt.Errorf("target %q must not set selector(s)", m.Target)
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

func normalizeSensitivity(v SensitivityPreset) SensitivityPreset {
	switch SensitivityPreset(strings.ToLower(strings.TrimSpace(string(v)))) {
	case SensitivitySlow:
		return SensitivitySlow
	case SensitivityFast:
		return SensitivityFast
	default:
		return SensitivityNormal
	}
}

func validateSensitivity(v SensitivityPreset) error {
	switch v {
	case SensitivitySlow, SensitivityNormal, SensitivityFast:
		return nil
	default:
		return fmt.Errorf("invalid sensitivity %q", v)
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
		Debug:    false,
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
