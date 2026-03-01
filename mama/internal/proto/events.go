package proto

import (
	"errors"
	"fmt"
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
	KnobID          int // 1..5
	Delta           int // encoder delta steps (can be >1 or < -1)
	ProtocolVersion int // populated for EventProtocolHello
}

func IsProtocolCompatible(version int) bool {
	return version == HostProtocolVersion
}

func ParseLine(line string) (Event, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Event{}, errors.New("empty line")
	}

	// Protocol hello: V:<version>
	if strings.HasPrefix(line, "V:") {
		version, err := strconv.Atoi(strings.TrimPrefix(line, "V:"))
		if err != nil || version <= 0 {
			return Event{}, fmt.Errorf("bad protocol version: %q", line)
		}
		return Event{Kind: EventProtocolHello, ProtocolVersion: version}, nil
	}

	// Encoder: E<id>:+<n> or E<id>:-<n>
	if strings.HasPrefix(line, "E") {
		parts := strings.SplitN(line[1:], ":", 2)
		if len(parts) != 2 {
			return Event{}, fmt.Errorf("bad encoder format: %q", line)
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil || id < 1 || id > 32 {
			return Event{}, fmt.Errorf("bad knob id: %q", parts[0])
		}
		delta, err := strconv.Atoi(parts[1])
		if err != nil {
			return Event{}, fmt.Errorf("bad delta: %q", parts[1])
		}
		if delta == 0 {
			return Event{}, errors.New("delta=0")
		}
		return Event{Kind: EventEncoderDelta, KnobID: id, Delta: delta}, nil
	}

	// Button: B<id>:1
	if strings.HasPrefix(line, "B") {
		parts := strings.SplitN(line[1:], ":", 2)
		if len(parts) != 2 {
			return Event{}, fmt.Errorf("bad button format: %q", line)
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil || id < 1 || id > 32 {
			return Event{}, fmt.Errorf("bad knob id: %q", parts[0])
		}
		pressed, err := strconv.Atoi(parts[1])
		if err != nil {
			return Event{}, fmt.Errorf("bad button value: %q", parts[1])
		}
		if pressed != 1 {
			return Event{}, fmt.Errorf("unsupported button value: %d", pressed)
		}
		return Event{Kind: EventButtonPress, KnobID: id}, nil
	}

	return Event{}, fmt.Errorf("unknown event: %q", line)
}
