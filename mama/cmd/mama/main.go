package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/proto"
	"mama/internal/runtime"
	serialx "mama/internal/serial"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "", "path to config yaml (default: auto)")
	flag.Parse()

	if cfgPath == "" {
		cfgPath = config.ResolveDefaultPath()
	}
	if _, err := config.EnsureDefaultFile(cfgPath); err != nil {
		runtime.Log("startup_error", runtime.Fields{"error": err, "step": "ensure_config"})
		os.Exit(1)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		runtime.Log("startup_error", runtime.Fields{"error": err, "step": "load_config"})
		os.Exit(1)
	}

	b := audio.NewBackend()

	fmt.Printf("MAMA started. Serial=%s @ %d\n", cfg.Serial.Port, cfg.Serial.Baud)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ctrl+C handling
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigC)
	go func() {
		<-sigC
		fmt.Println("\nStopping...")
		cancel()
	}()

	backoff := runtime.NewBackoff(500*time.Millisecond, 10*time.Second)
	metrics := runtime.NewMetrics()

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		runtime.Log("serial_state", runtime.Fields{"baud": cfg.Serial.Baud, "port": cfg.Serial.Port, "state": "connecting"})
		r, err := serialx.Open(cfg.Serial.Port, cfg.Serial.Baud)
		if err != nil {
			metrics.IncReconnectCount()
			delay := backoff.Next()
			runtime.Log("serial_state", runtime.Fields{"error": err, "retry_in": delay, "state": "reconnecting"})
			runtime.Log("runtime_metrics", runtime.Fields{"snapshot": metrics.Snapshot()})
			if !sleepWithContext(ctx, delay) {
				return
			}
			continue
		}

		backoff.Reset()
		runtime.Log("serial_state", runtime.Fields{"port": cfg.Serial.Port, "state": "connected"})
		if err := runSession(ctx, r, &cfg, b, metrics); err != nil && !errors.Is(err, context.Canceled) {
			metrics.IncReconnectCount()
			delay := backoff.Next()
			runtime.Log("serial_state", runtime.Fields{"error": err, "state": "disconnected"})
			runtime.Log("serial_state", runtime.Fields{"retry_in": delay, "state": "reconnecting"})
			runtime.Log("runtime_metrics", runtime.Fields{"snapshot": metrics.Snapshot()})
			if !sleepWithContext(ctx, delay) {
				return
			}
			continue
		}
		return
	}
}

func runSession(ctx context.Context, r *serialx.Reader, cfg *config.Config, b audio.Backend, metrics *runtime.Metrics) error {
	lines := make(chan string, 128)
	readErrC := make(chan error, 1)
	go func() {
		readErrC <- r.ReadLines(ctx, lines)
	}()

	defer func() { _ = r.Close() }()
	return runSessionFromChannels(ctx, cfg, b, metrics, lines, readErrC)
}

func runSessionFromChannels(ctx context.Context, cfg *config.Config, b audio.Backend, metrics *runtime.Metrics, lines <-chan string, readErrC <-chan error) error {
	defer runtime.Log("runtime_metrics", runtime.Fields{"snapshot": metrics.Snapshot()})

	protocolAnnounced := false
	protocolCompatible := true

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
				protocolAnnounced = true
				protocolCompatible = proto.IsProtocolCompatible(ev.ProtocolVersion)
				if protocolCompatible {
					runtime.Log("protocol_negotiated", runtime.Fields{"firmware": ev.ProtocolVersion, "host": proto.HostProtocolVersion})
				} else {
					runtime.Log("protocol_mismatch", runtime.Fields{"firmware": ev.ProtocolVersion, "host": proto.HostProtocolVersion, "note": "dropping_control_events"})
				}
				continue
			}

			if protocolAnnounced && !protocolCompatible {
				metrics.IncDroppedEvents()
				if cfg.Debug {
					runtime.Log("event_dropped", runtime.Fields{"line": line, "reason": "incompatible_protocol"})
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
					metrics.IncBackendFailures()
					if cfg.Debug {
						runtime.Log("backend_error", runtime.Fields{"error": err, "knob_id": ev.KnobID, "operation": "adjust"})
					}
				}
			case proto.EventButtonPress:
				if err := b.ToggleMute(m.Target, backendTargetName(m)); err != nil {
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
