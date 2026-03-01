package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
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
		log.Fatalf("failed to ensure config file: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
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

		log.Printf("serial state: connecting port=%s baud=%d", cfg.Serial.Port, cfg.Serial.Baud)
		r, err := serialx.Open(cfg.Serial.Port, cfg.Serial.Baud)
		if err != nil {
			metrics.IncReconnectCount()
			delay := backoff.Next()
			log.Printf("serial state: reconnecting open_failed=%v retry_in=%s", err, delay)
			log.Printf("runtime metrics: %s", metrics.Snapshot())
			if !sleepWithContext(ctx, delay) {
				return
			}
			continue
		}

		backoff.Reset()
		log.Printf("serial state: connected port=%s", cfg.Serial.Port)
		if err := runSession(ctx, r, &cfg, b, metrics); err != nil && !errors.Is(err, context.Canceled) {
			metrics.IncReconnectCount()
			delay := backoff.Next()
			log.Printf("serial state: disconnected err=%v", err)
			log.Printf("serial state: reconnecting retry_in=%s", delay)
			log.Printf("runtime metrics: %s", metrics.Snapshot())
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
	defer log.Printf("runtime metrics: %s", metrics.Snapshot())

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
				return fmt.Errorf("serial reader stopped")
			}
			if cfg.Debug {
				log.Printf("RX: %s", line)
			}

			ev, err := proto.ParseLine(line)
			if err != nil {
				metrics.IncParseErrors()
				if cfg.Debug {
					log.Printf("parse error: %v", err)
				}
				continue
			}

			if ev.Kind == proto.EventProtocolHello {
				protocolAnnounced = true
				protocolCompatible = proto.IsProtocolCompatible(ev.ProtocolVersion)
				if protocolCompatible {
					log.Printf("protocol version negotiated: firmware=%d host=%d", ev.ProtocolVersion, proto.HostProtocolVersion)
				} else {
					log.Printf("protocol mismatch: firmware=%d host=%d (dropping control events)", ev.ProtocolVersion, proto.HostProtocolVersion)
				}
				continue
			}

			if protocolAnnounced && !protocolCompatible {
				metrics.IncDroppedEvents()
				if cfg.Debug {
					log.Printf("dropping event due to incompatible protocol: %q", line)
				}
				continue
			}

			m, ok := cfg.MappingForKnob(ev.KnobID)
			if !ok {
				metrics.IncDroppedEvents()
				if cfg.Debug {
					log.Printf("unmapped knob: %d", ev.KnobID)
				}
				continue
			}

			switch ev.Kind {
			case proto.EventEncoderDelta:
				if err := b.Adjust(m.Target, m.Name, m.Step, ev.Delta); err != nil {
					metrics.IncBackendFailures()
					if cfg.Debug {
						log.Printf("adjust error knob %d: %v", ev.KnobID, err)
					}
				}
			case proto.EventButtonPress:
				if err := b.ToggleMute(m.Target, m.Name); err != nil {
					metrics.IncBackendFailures()
					if cfg.Debug {
						log.Printf("mute error knob %d: %v", ev.KnobID, err)
					}
				}
			default:
				metrics.IncDroppedEvents()
			}
		}
	}
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
