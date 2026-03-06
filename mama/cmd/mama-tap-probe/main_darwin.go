//go:build darwin && cgo

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"mama/internal/audio"
)

func main() {
	var bundleID string
	var duration time.Duration
	var volume int
	var muted bool

	flag.StringVar(&bundleID, "bundle", "", "bundle ID of the running app to tap")
	flag.DurationVar(&duration, "duration", 5*time.Second, "how long to run the tap before reporting stats")
	flag.IntVar(&volume, "volume", 100, "software gain percent to apply to the tapped app")
	flag.BoolVar(&muted, "muted", false, "start the tap muted")
	flag.Parse()

	if bundleID == "" {
		flag.Usage()
		os.Exit(2)
	}

	result, err := audio.RunDarwinTapProbe(bundleID, duration, volume, muted)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("bundle=%s\n", result.BundleID)
	fmt.Printf("process_object=%d\n", result.ProcessObject)
	fmt.Printf("session_key=%s\n", result.SessionKey)
	fmt.Printf("output_device_uid=%s\n", result.OutputDeviceUID)
	fmt.Printf("restore_capable=%t\n", result.RestoreCapable)
	fmt.Printf("volume=%d\n", result.Volume)
	fmt.Printf("muted=%t\n", result.Muted)
	fmt.Printf("duration=%s\n", result.Duration)
	fmt.Printf("callbacks=%d\n", result.Callbacks)
	fmt.Printf("frames=%d\n", result.Frames)
}
