package proto

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const HostProtocolVersion = 1

type EventKind int

const (
	EventEncoderDelta EventKind = iota
	EventButtonPress
	EventProtocolHello
)

type Event struct {
	Kind            EventKind
	KnobID          int // 1..32
	Delta           int // encoder delta steps (can be >1 or < -1)
	ProtocolVersion int // populated for EventProtocolHello
}

func IsProtocolCompatible(version int) bool {
	return version == HostProtocolVersion
}

var (
	protocolLinePattern    = regexp.MustCompile(`(?i)^v\s*[:=]\s*([0-9]+)\s*$`)
	protocolTokenPattern   = regexp.MustCompile(`(?i)\bv\s*[:=]\s*([0-9]+)\b`)
	encoderLinePattern     = regexp.MustCompile(`(?i)^[ek]\s*([0-9]+)\s*[:=,]\s*([+-]?\s*[0-9]+)\s*$`)
	encoderTokenPattern    = regexp.MustCompile(`(?i)\b[ek]\s*([0-9]+)\s*[:=,]\s*([+-]?\s*[0-9]+)\b`)
	buttonLinePattern      = regexp.MustCompile(`(?i)^b\s*([0-9]+)\s*[:=,]\s*([a-z0-9]+)\s*$`)
	buttonTokenPattern     = regexp.MustCompile(`(?i)\bb\s*([0-9]+)\s*[:=,]\s*([a-z0-9]+)\b`)
	legacyPressLinePattern = regexp.MustCompile(`(?i)^k\s*([0-9]+)\s*[:=,]\s*(p|press|btn|button|down|m|mute)\s*$`)
	legacyPressTokenPattern = regexp.MustCompile(`(?i)\bk\s*([0-9]+)\s*[:=,]\s*(p|press|btn|button|down|m|mute)\b`)
)

func ParseLine(line string) (Event, error) {
	line = sanitizeLine(line)
	if line == "" {
		return Event{}, errors.New("empty line")
	}

	if ev, ok, err := parseProtocolLine(line, protocolLinePattern); ok || err != nil {
		return ev, err
	}
	if ev, ok, err := parseEncoderLine(line, encoderLinePattern); ok || err != nil {
		return ev, err
	}
	if ev, ok, err := parseButtonLine(line, buttonLinePattern); ok || err != nil {
		return ev, err
	}
	if ev, ok, err := parseLegacyPressLine(line, legacyPressLinePattern); ok || err != nil {
		return ev, err
	}

	// Best-effort fallback for firmwares that prefix events with extra text.
	if ev, ok, err := parseProtocolLine(line, protocolTokenPattern); ok || err != nil {
		return ev, err
	}
	if ev, ok, err := parseEncoderLine(line, encoderTokenPattern); ok || err != nil {
		return ev, err
	}
	if ev, ok, err := parseButtonLine(line, buttonTokenPattern); ok || err != nil {
		return ev, err
	}
	if ev, ok, err := parseLegacyPressLine(line, legacyPressTokenPattern); ok || err != nil {
		return ev, err
	}

	return Event{}, fmt.Errorf("unknown event: %q", line)
}

func sanitizeLine(line string) string {
	line = strings.ReplaceAll(line, "\x00", "")
	return strings.TrimSpace(line)
}

func parseProtocolLine(line string, pattern *regexp.Regexp) (Event, bool, error) {
	m := pattern.FindStringSubmatch(line)
	if len(m) != 2 {
		return Event{}, false, nil
	}
	version, err := parsePositiveInt(m[1])
	if err != nil {
		return Event{}, true, fmt.Errorf("bad protocol version: %q", line)
	}
	return Event{Kind: EventProtocolHello, ProtocolVersion: version}, true, nil
}

func parseEncoderLine(line string, pattern *regexp.Regexp) (Event, bool, error) {
	m := pattern.FindStringSubmatch(line)
	if len(m) != 3 {
		return Event{}, false, nil
	}
	id, err := parseKnobID(m[1])
	if err != nil {
		return Event{}, true, err
	}

	delta, err := parseSignedInt(m[2])
	if err != nil {
		return Event{}, true, fmt.Errorf("bad delta: %q", m[2])
	}
	if delta == 0 {
		return Event{}, true, errors.New("delta=0")
	}
	return Event{Kind: EventEncoderDelta, KnobID: id, Delta: delta}, true, nil
}

func parseButtonLine(line string, pattern *regexp.Regexp) (Event, bool, error) {
	m := pattern.FindStringSubmatch(line)
	if len(m) != 3 {
		return Event{}, false, nil
	}

	id, err := parseKnobID(m[1])
	if err != nil {
		return Event{}, true, err
	}
	if err := validateButtonPressValue(m[2]); err != nil {
		return Event{}, true, err
	}
	return Event{Kind: EventButtonPress, KnobID: id}, true, nil
}

func parseLegacyPressLine(line string, pattern *regexp.Regexp) (Event, bool, error) {
	m := pattern.FindStringSubmatch(line)
	if len(m) != 3 {
		return Event{}, false, nil
	}
	id, err := parseKnobID(m[1])
	if err != nil {
		return Event{}, true, err
	}
	return Event{Kind: EventButtonPress, KnobID: id}, true, nil
}

func parsePositiveInt(raw string) (int, error) {
	v, err := parseSignedInt(raw)
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, fmt.Errorf("must be > 0")
	}
	return v, nil
}

func parseSignedInt(raw string) (int, error) {
	token := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	return strconv.Atoi(token)
}

func parseKnobID(raw string) (int, error) {
	id, err := parsePositiveInt(raw)
	if err != nil || id > 32 {
		return 0, fmt.Errorf("bad knob id: %q", raw)
	}
	return id, nil
}

func validateButtonPressValue(raw string) error {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "1", "p", "press", "down", "true", "btn", "button":
		return nil
	}

	if n, err := parseSignedInt(value); err == nil {
		if n == 1 {
			return nil
		}
		return fmt.Errorf("unsupported button value: %d", n)
	}

	return fmt.Errorf("bad button value: %q", raw)
}
