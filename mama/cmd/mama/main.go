package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/mixer"
	"mama/internal/ui"
)

func main() {
	var cfgPath string
	var listenAddr string
	var openBrowser bool

	flag.StringVar(&cfgPath, "config", "", "path to config yaml (default: auto)")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:18765", "HTTP listen address")
	flag.BoolVar(&openBrowser, "open", true, "open web UI in default browser")
	flag.Parse()

	if cfgPath == "" {
		cfgPath = config.ResolveDefaultPath()
	}
	if _, err := config.EnsureDefaultFile(cfgPath); err != nil {
		log.Printf("startup error (ensure config): %v", err)
		os.Exit(1)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("startup error (load config): %v", err)
		os.Exit(1)
	}

	mixerService, err := mixer.NewService(cfg, audio.NewBackend())
	if err != nil {
		log.Printf("startup error (init mixer): %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigC)
	go func() {
		<-sigC
		fmt.Println("\nStopping...")
		cancel()
	}()

	srv := ui.New(cfgPath)
	srv.SetMixerService(mixerService)

	mixerErrC := make(chan error, 1)
	go func() {
		mixerErrC <- mixerService.Run(ctx)
	}()

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: srv.Handler(),
	}
	httpErrC := make(chan error, 1)
	go func() {
		httpErrC <- httpServer.ListenAndServe()
	}()

	uiURL := fmt.Sprintf("http://%s", listenAddr)
	fmt.Printf("MAMA running. UI=%s\n", uiURL)
	fmt.Printf("Config file: %s\n", cfgPath)
	fmt.Printf("Serial target: %s @ %d\n", cfg.Serial.Port, cfg.Serial.Baud)

	if openBrowser {
		go openURL(uiURL)
	}

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-mixerErrC:
		if err != nil && !errors.Is(err, context.Canceled) {
			runErr = fmt.Errorf("mixer service error: %w", err)
		}
		cancel()
	case err := <-httpErrC:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			runErr = fmt.Errorf("ui server error: %w", err)
		}
		cancel()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) && runErr == nil {
		runErr = fmt.Errorf("ui shutdown error: %w", err)
	}

	select {
	case err := <-mixerErrC:
		if err != nil && !errors.Is(err, context.Canceled) && runErr == nil {
			runErr = fmt.Errorf("mixer service error: %w", err)
		}
	case <-time.After(1500 * time.Millisecond):
	}

	select {
	case err := <-httpErrC:
		if err != nil && !errors.Is(err, http.ErrServerClosed) && runErr == nil {
			runErr = fmt.Errorf("ui server error: %w", err)
		}
	case <-time.After(1500 * time.Millisecond):
	}

	if runErr != nil {
		log.Print(runErr)
		os.Exit(1)
	}
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
