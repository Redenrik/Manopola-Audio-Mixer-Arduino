package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/proto"
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

	r, err := serialx.Open(cfg.Serial.Port, cfg.Serial.Baud)
	if err != nil {
		log.Fatalf("serial open error: %v", err)
	}
	var closeOnce sync.Once
	closeReader := func() {
		closeOnce.Do(func() { _ = r.Close() })
	}
	defer closeReader()

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
		// Closing the serial port unblocks a pending read so shutdown can complete.
		closeReader()
	}()

	lines := make(chan string, 128)
	go func() {
		if err := r.ReadLines(ctx, lines); err != nil && cfg.Debug && !errors.Is(err, context.Canceled) {
			log.Printf("serial read ended: %v", err)
		}
	}()

	for line := range lines {
		if cfg.Debug {
			log.Printf("RX: %s", line)
		}

		ev, err := proto.ParseLine(line)
		if err != nil {
			if cfg.Debug {
				log.Printf("parse error: %v", err)
			}
			continue
		}

		m, ok := cfg.MappingForKnob(ev.KnobID)
		if !ok {
			if cfg.Debug {
				log.Printf("unmapped knob: %d", ev.KnobID)
			}
			continue
		}

		switch ev.Kind {
		case proto.EventEncoderDelta:
			if err := b.Adjust(m.Target, m.Name, m.Step, ev.Delta); err != nil && cfg.Debug {
				log.Printf("adjust error knob %d: %v", ev.KnobID, err)
			}
		case proto.EventButtonPress:
			if err := b.ToggleMute(m.Target, m.Name); err != nil && cfg.Debug {
				log.Printf("mute error knob %d: %v", ev.KnobID, err)
			}
		}
	}
}
