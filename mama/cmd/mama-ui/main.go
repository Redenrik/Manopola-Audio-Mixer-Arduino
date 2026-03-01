package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"mama/internal/config"
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
		log.Fatalf("failed to ensure config file: %v", err)
	}

	url := fmt.Sprintf("http://%s", listenAddr)
	if openBrowser {
		go openURL(url)
	}

	fmt.Printf("MAMA UI listening on %s\n", url)
	fmt.Printf("Config file: %s\n", cfgPath)

	srv := ui.New(cfgPath)
	if err := srv.Run(listenAddr); err != nil {
		log.Fatalf("ui server error: %v", err)
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
