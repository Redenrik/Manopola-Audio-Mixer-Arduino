package mixer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/proto"
	"mama/internal/runtime"
	serialx "mama/internal/serial"
)

type EventType string

const (
	EventTypeRaw   EventType = "raw"
	EventTypeEvent EventType = "event"
	EventTypeError EventType = "error"
	EventTypeState EventType = "state"
)

type Event struct {
	Type            EventType `json:"type"`
	Time            time.Time `json:"time"`
	Raw             string    `json:"raw,omitempty"`
	Kind            string    `json:"kind,omitempty"`
	KnobID          int       `json:"knobId,omitempty"`
	Delta           int       `json:"delta,omitempty"`
	ProtocolVersion int       `json:"protocolVersion,omitempty"`
	Message         string    `json:"message,omitempty"`
	State           string    `json:"state,omitempty"`
	Port            string    `json:"port,omitempty"`
	Baud            int       `json:"baud,omitempty"`
	Connected       bool      `json:"connected,omitempty"`
	RetryIn         string    `json:"retryIn,omitempty"`
	Snapshot        any       `json:"snapshot,omitempty"`
	Error           string    `json:"error,omitempty"`
}

type Status struct {
	State     string    `json:"state"`
	Port      string    `json:"port,omitempty"`
	Baud      int       `json:"baud,omitempty"`
	Connected bool      `json:"connected"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	RetryIn   string    `json:"retryIn,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Service struct {
	backend audio.Backend
	metrics *runtime.Metrics

	cfgMu sync.RWMutex
	cfg   config.Config

	statusMu sync.RWMutex
	status   Status

	subsMu    sync.RWMutex
	subs      map[int]chan Event
	nextSubID int

	sessionMu     sync.Mutex
	sessionID     uint64
	sessionCancel context.CancelFunc
	sessionClose  func()
}

func NewService(initial config.Config, backend audio.Backend) (*Service, error) {
	cfg, err := config.Validate(initial)
	if err != nil {
		return nil, err
	}
	if backend == nil {
		backend = audio.NewBackend()
	}

	s := &Service{
		backend: backend,
		metrics: runtime.NewMetrics(),
		cfg:     cloneConfig(cfg),
		subs:    make(map[int]chan Event),
		status: Status{
			State:     "starting",
			Port:      cfg.Serial.Port,
			Baud:      cfg.Serial.Baud,
			Message:   "starting",
			UpdatedAt: time.Now().UTC(),
		},
	}
	return s, nil
}

func (s *Service) Run(ctx context.Context) error {
	backoff := runtime.NewBackoff(500*time.Millisecond, 10*time.Second)

	for {
		if ctx.Err() != nil {
			s.setStatus(Status{
				State:     "stopped",
				Connected: false,
				Message:   "stopped",
			})
			return nil
		}

		cfg := s.configSnapshot()
		port := cfg.Serial.Port
		baud := cfg.Serial.Baud

		runtime.Log("serial_state", runtime.Fields{
			"state": "connecting",
			"port":  port,
			"baud":  baud,
		})
		s.setStatus(Status{
			State:     "connecting",
			Port:      port,
			Baud:      baud,
			Connected: false,
			Message:   fmt.Sprintf("connecting to %s @ %d", port, baud),
		})

		reader, err := serialx.Open(port, baud)
		if err != nil {
			s.metrics.IncReconnectCount()
			delay := backoff.Next()
			runtime.Log("serial_state", runtime.Fields{
				"state":    "reconnecting",
				"port":     port,
				"baud":     baud,
				"error":    err,
				"retry_in": delay,
			})
			s.publish(Event{
				Type:     EventTypeError,
				Time:     time.Now().UTC(),
				Message:  "serial connection failed",
				Error:    err.Error(),
				Port:     port,
				Baud:     baud,
				RetryIn:  delay.String(),
				Snapshot: s.metrics.Snapshot(),
			})
			s.setStatus(Status{
				State:     "reconnecting",
				Port:      port,
				Baud:      baud,
				Connected: false,
				Error:     err.Error(),
				RetryIn:   delay.String(),
				Message:   "reconnecting",
			})
			runtime.Log("runtime_metrics", runtime.Fields{"snapshot": s.metrics.Snapshot()})
			if !sleepWithContext(ctx, delay) {
				s.setStatus(Status{
					State:     "stopped",
					Connected: false,
					Message:   "stopped",
				})
				return nil
			}
			continue
		}

		backoff.Reset()
		runtime.Log("serial_state", runtime.Fields{
			"state": "connected",
			"port":  port,
		})
		s.setStatus(Status{
			State:     "connected",
			Port:      port,
			Baud:      baud,
			Connected: true,
			Message:   fmt.Sprintf("connected to %s @ %d", port, baud),
		})

		err = s.runSession(ctx, reader, port, baud)
		runtime.Log("runtime_metrics", runtime.Fields{"snapshot": s.metrics.Snapshot()})
		if ctx.Err() != nil {
			s.setStatus(Status{
				State:     "stopped",
				Connected: false,
				Message:   "stopped",
			})
			return nil
		}

		if errors.Is(err, context.Canceled) {
			s.setStatus(Status{
				State:     "reconnecting",
				Connected: false,
				Message:   "applying updated serial settings",
			})
			continue
		}

		if err != nil {
			s.metrics.IncReconnectCount()
			delay := backoff.Next()
			runtime.Log("serial_state", runtime.Fields{
				"state":    "disconnected",
				"error":    err,
				"retry_in": delay,
			})
			s.publish(Event{
				Type:     EventTypeError,
				Time:     time.Now().UTC(),
				Message:  "serial session ended with error",
				Error:    err.Error(),
				RetryIn:  delay.String(),
				Snapshot: s.metrics.Snapshot(),
			})
			s.setStatus(Status{
				State:     "reconnecting",
				Connected: false,
				Error:     err.Error(),
				RetryIn:   delay.String(),
				Message:   "reconnecting",
			})
			if !sleepWithContext(ctx, delay) {
				s.setStatus(Status{
					State:     "stopped",
					Connected: false,
					Message:   "stopped",
				})
				return nil
			}
			continue
		}

		// Gracefully ended session (rare), reconnect immediately.
		s.setStatus(Status{
			State:     "reconnecting",
			Connected: false,
			Message:   "session ended, reconnecting",
		})
		if !sleepWithContext(ctx, 250*time.Millisecond) {
			s.setStatus(Status{
				State:     "stopped",
				Connected: false,
				Message:   "stopped",
			})
			return nil
		}
	}
}

func (s *Service) runSession(parent context.Context, reader *serialx.Reader, port string, baud int) error {
	lines := make(chan string, 128)
	readErrC := make(chan error, 1)

	ctx, cancel := context.WithCancel(parent)
	closeOnce := sync.OnceFunc(func() {
		_ = reader.Close()
	})

	sessionID := s.setSession(cancel, closeOnce)
	defer s.clearSession(sessionID)
	defer cancel()
	defer closeOnce()

	go func() {
		<-ctx.Done()
		closeOnce()
	}()
	go func() {
		readErrC <- reader.ReadLines(ctx, lines)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErrC:
			if err != nil {
				return err
			}
			return nil
		case line, ok := <-lines:
			if !ok {
				select {
				case err := <-readErrC:
					if err != nil {
						return err
					}
					return nil
				default:
					return fmt.Errorf("serial reader stopped")
				}
			}

			cfg := s.configSnapshot()
			if cfg.Debug {
				runtime.Log("serial_rx", runtime.Fields{"line": line})
			}
			s.publish(Event{
				Type: EventTypeRaw,
				Time: time.Now().UTC(),
				Raw:  line,
				Port: port,
				Baud: baud,
			})

			ev, err := proto.ParseLine(line)
			if err != nil {
				s.metrics.IncParseErrors()
				if cfg.Debug {
					runtime.Log("parse_error", runtime.Fields{"error": err})
				}
				continue
			}

			switch ev.Kind {
			case proto.EventProtocolHello:
				if proto.IsProtocolCompatible(ev.ProtocolVersion) {
					runtime.Log("protocol_negotiated", runtime.Fields{"firmware": ev.ProtocolVersion, "host": proto.HostProtocolVersion})
				} else {
					runtime.Log("protocol_mismatch", runtime.Fields{"firmware": ev.ProtocolVersion, "host": proto.HostProtocolVersion, "note": "processing_parseable_events_best_effort"})
				}
				s.publish(Event{
					Type:            EventTypeEvent,
					Time:            time.Now().UTC(),
					Kind:            "protocol_hello",
					ProtocolVersion: ev.ProtocolVersion,
					Port:            port,
					Baud:            baud,
				})
				continue
			case proto.EventEncoderDelta:
				s.publish(Event{
					Type:   EventTypeEvent,
					Time:   time.Now().UTC(),
					Kind:   "encoder_delta",
					KnobID: ev.KnobID,
					Delta:  ev.Delta,
					Port:   port,
					Baud:   baud,
				})
			case proto.EventButtonPress:
				s.publish(Event{
					Type:   EventTypeEvent,
					Time:   time.Now().UTC(),
					Kind:   "button_press",
					KnobID: ev.KnobID,
					Port:   port,
					Baud:   baud,
				})
			default:
				s.metrics.IncDroppedEvents()
				continue
			}

			m, ok := cfg.MappingForKnob(ev.KnobID)
			if !ok {
				s.metrics.IncDroppedEvents()
				if cfg.Debug {
					runtime.Log("event_dropped", runtime.Fields{"knob_id": ev.KnobID, "reason": "unmapped_knob"})
				}
				continue
			}

			switch ev.Kind {
			case proto.EventEncoderDelta:
				if err := s.backend.Adjust(m.Target, backendTargetName(m), m.Step, ev.Delta); err != nil {
					if errors.Is(err, audio.ErrTargetUnavailable) {
						if cfg.Debug {
							runtime.Log("target_unavailable", runtime.Fields{"knob_id": ev.KnobID, "operation": "adjust"})
						}
						continue
					}
					s.metrics.IncBackendFailures()
					s.publish(Event{
						Type:    EventTypeError,
						Time:    time.Now().UTC(),
						Kind:    "encoder_delta",
						KnobID:  ev.KnobID,
						Message: "backend adjust failed",
						Error:   err.Error(),
					})
					if cfg.Debug {
						runtime.Log("backend_error", runtime.Fields{"error": err, "knob_id": ev.KnobID, "operation": "adjust"})
					}
				}
			case proto.EventButtonPress:
				if err := s.backend.ToggleMute(m.Target, backendTargetName(m)); err != nil {
					if errors.Is(err, audio.ErrTargetUnavailable) {
						if cfg.Debug {
							runtime.Log("target_unavailable", runtime.Fields{"knob_id": ev.KnobID, "operation": "toggle_mute"})
						}
						continue
					}
					s.metrics.IncBackendFailures()
					s.publish(Event{
						Type:    EventTypeError,
						Time:    time.Now().UTC(),
						Kind:    "button_press",
						KnobID:  ev.KnobID,
						Message: "backend mute toggle failed",
						Error:   err.Error(),
					})
					if cfg.Debug {
						runtime.Log("backend_error", runtime.Fields{"error": err, "knob_id": ev.KnobID, "operation": "toggle_mute"})
					}
				}
			default:
				s.metrics.IncDroppedEvents()
			}
		}
	}
}

func (s *Service) UpdateConfig(next config.Config) error {
	validated, err := config.Validate(next)
	if err != nil {
		return err
	}
	validated = cloneConfig(validated)

	current := s.configSnapshot()
	serialChanged := current.Serial.Port != validated.Serial.Port || current.Serial.Baud != validated.Serial.Baud

	s.cfgMu.Lock()
	s.cfg = validated
	s.cfgMu.Unlock()

	runtime.Log("config_reloaded", runtime.Fields{
		"port":           validated.Serial.Port,
		"baud":           validated.Serial.Baud,
		"serial_changed": serialChanged,
	})

	if serialChanged {
		s.setStatus(Status{
			State:     "reconnecting",
			Port:      validated.Serial.Port,
			Baud:      validated.Serial.Baud,
			Connected: false,
			Message:   "applying updated serial settings",
		})
		s.requestReconnect()
	}

	return nil
}

func (s *Service) Status() Status {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	return s.status
}

func (s *Service) MetricsSnapshot() runtime.MetricsSnapshot {
	return s.metrics.Snapshot()
}

func (s *Service) ConfigSnapshot() config.Config {
	return s.configSnapshot()
}

func (s *Service) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 64
	}
	ch := make(chan Event, buffer)

	s.subsMu.Lock()
	id := s.nextSubID
	s.nextSubID++
	s.subs[id] = ch
	s.subsMu.Unlock()

	// New subscribers immediately receive the latest runtime state snapshot.
	select {
	case ch <- statusEvent(s.Status()):
	default:
	}

	cancel := func() {
		s.subsMu.Lock()
		stream, ok := s.subs[id]
		if ok {
			delete(s.subs, id)
			close(stream)
		}
		s.subsMu.Unlock()
	}
	return ch, cancel
}

func (s *Service) requestReconnect() {
	s.sessionMu.Lock()
	cancel := s.sessionCancel
	closeFn := s.sessionClose
	s.sessionMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if closeFn != nil {
		closeFn()
	}
}

func (s *Service) setSession(cancel context.CancelFunc, closeFn func()) uint64 {
	s.sessionMu.Lock()
	s.sessionID++
	id := s.sessionID
	s.sessionCancel = cancel
	s.sessionClose = closeFn
	s.sessionMu.Unlock()
	return id
}

func (s *Service) clearSession(id uint64) {
	s.sessionMu.Lock()
	if s.sessionID == id {
		s.sessionCancel = nil
		s.sessionClose = nil
	}
	s.sessionMu.Unlock()
}

func (s *Service) configSnapshot() config.Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return cloneConfig(s.cfg)
}

func (s *Service) setStatus(next Status) {
	next.UpdatedAt = time.Now().UTC()
	s.statusMu.Lock()
	s.status = next
	s.statusMu.Unlock()
	s.publish(statusEvent(next))
}

func statusEvent(status Status) Event {
	return Event{
		Type:      EventTypeState,
		Time:      status.UpdatedAt,
		State:     status.State,
		Port:      status.Port,
		Baud:      status.Baud,
		Connected: status.Connected,
		Message:   status.Message,
		Error:     status.Error,
		RetryIn:   status.RetryIn,
	}
}

func (s *Service) publish(ev Event) {
	s.subsMu.RLock()
	defer s.subsMu.RUnlock()
	for _, ch := range s.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func backendTargetName(m config.Mapping) string {
	if m.Target == config.TargetApp && m.Selector != nil {
		return string(m.Selector.Kind) + ":" + m.Selector.Value
	}
	if m.Target == config.TargetGroup && len(m.Selectors) > 0 {
		if b, err := json.Marshal(m.Selectors); err == nil {
			return string(b)
		}
	}
	return m.Name
}

func cloneConfig(in config.Config) config.Config {
	out := in
	out.Mappings = cloneMappings(in.Mappings)
	out.Profiles = make([]config.Profile, len(in.Profiles))
	for i := range in.Profiles {
		out.Profiles[i] = config.Profile{
			Name:     in.Profiles[i].Name,
			Mappings: cloneMappings(in.Profiles[i].Mappings),
		}
	}
	return out
}

func cloneMappings(in []config.Mapping) []config.Mapping {
	if len(in) == 0 {
		return nil
	}
	out := make([]config.Mapping, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].Selector != nil {
			selector := *in[i].Selector
			out[i].Selector = &selector
		}
		if len(in[i].Selectors) > 0 {
			out[i].Selectors = append([]config.Selector(nil), in[i].Selectors...)
		} else {
			out[i].Selectors = nil
		}
	}
	return out
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
