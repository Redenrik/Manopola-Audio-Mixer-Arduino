package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/proto"
	"mama/internal/runtime"
	serialx "mama/internal/serial"
)

func runSession(ctx context.Context, r *serialx.Reader, cfg *config.Config, b audio.Backend, metrics *runtime.Metrics) error {
	lines := make(chan string, 128)
	readErrC := make(chan error, 1)
	stopClose := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = r.Close()
		case <-stopClose:
		}
	}()
	go func() {
		readErrC <- r.ReadLines(ctx, lines)
	}()

	defer close(stopClose)
	defer func() { _ = r.Close() }()
	return runSessionFromChannels(ctx, cfg, b, metrics, lines, readErrC)
}

func runSessionFromChannels(ctx context.Context, cfg *config.Config, b audio.Backend, metrics *runtime.Metrics, lines <-chan string, readErrC <-chan error) error {
	defer runtime.Log("runtime_metrics", runtime.Fields{"snapshot": metrics.Snapshot()})

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
			if cfg.Debug {
				runtime.Log("serial_rx", runtime.Fields{"line": line})
			}

			ev, err := proto.ParseLine(line)
			if err != nil {
				metrics.IncParseErrors()
				if cfg.Debug {
					runtime.Log("parse_error", runtime.Fields{"error": err})
				}
				continue
			}

			if ev.Kind == proto.EventProtocolHello {
				if proto.IsProtocolCompatible(ev.ProtocolVersion) {
					runtime.Log("protocol_negotiated", runtime.Fields{"firmware": ev.ProtocolVersion, "host": proto.HostProtocolVersion})
				} else {
					runtime.Log("protocol_mismatch", runtime.Fields{"firmware": ev.ProtocolVersion, "host": proto.HostProtocolVersion, "note": "processing_parseable_events_best_effort"})
				}
				continue
			}

			m, ok := cfg.MappingForKnob(ev.KnobID)
			if !ok {
				metrics.IncDroppedEvents()
				if cfg.Debug {
					runtime.Log("event_dropped", runtime.Fields{"knob_id": ev.KnobID, "reason": "unmapped_knob"})
				}
				continue
			}

			switch ev.Kind {
			case proto.EventEncoderDelta:
				if err := b.Adjust(m.Target, backendTargetName(m), m.Step, ev.Delta); err != nil {
					if errors.Is(err, audio.ErrTargetUnavailable) {
						if cfg.Debug {
							runtime.Log("target_unavailable", runtime.Fields{"knob_id": ev.KnobID, "operation": "adjust"})
						}
						continue
					}
					metrics.IncBackendFailures()
					if cfg.Debug {
						runtime.Log("backend_error", runtime.Fields{"error": err, "knob_id": ev.KnobID, "operation": "adjust"})
					}
				}
			case proto.EventButtonPress:
				if err := b.ToggleMute(m.Target, backendTargetName(m)); err != nil {
					if errors.Is(err, audio.ErrTargetUnavailable) {
						if cfg.Debug {
							runtime.Log("target_unavailable", runtime.Fields{"knob_id": ev.KnobID, "operation": "toggle_mute"})
						}
						continue
					}
					metrics.IncBackendFailures()
					if cfg.Debug {
						runtime.Log("backend_error", runtime.Fields{"error": err, "knob_id": ev.KnobID, "operation": "toggle_mute"})
					}
				}
			default:
				metrics.IncDroppedEvents()
			}
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
