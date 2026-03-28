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
	var selector string
	var duration time.Duration
	var volume int
	var muted bool

	flag.StringVar(&bundleID, "bundle", "", "bundle ID of the running app to tap")
	flag.StringVar(&selector, "selector", "", "selector token of the running app to tap, e.g. exe:spotify")
	flag.DurationVar(&duration, "duration", 5*time.Second, "how long to run the tap before reporting stats")
	flag.IntVar(&volume, "volume", 100, "software gain percent to apply to the tapped app")
	flag.BoolVar(&muted, "muted", false, "start the tap muted")
	flag.Parse()

	if bundleID == "" && selector == "" {
		flag.Usage()
		os.Exit(2)
	}

	result, err := audio.RunDarwinTapProbeSelector(selector, bundleID, duration, volume, muted)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("bundle=%s\n", result.BundleID)
	fmt.Printf("selector=%s\n", result.Selector)
	fmt.Printf("state_key=%s\n", result.StateKey)
	fmt.Printf("process_objects=%v\n", result.MatchedProcessObjects)
	fmt.Printf("output_device_uid=%s\n", result.OutputDeviceUID)
	fmt.Printf("restore_capable=%t\n", result.RestoreCapable)
	fmt.Printf("volume=%d\n", result.Volume)
	fmt.Printf("muted=%t\n", result.Muted)
	fmt.Printf("sample_rate=%.0f\n", result.SampleRate)
	fmt.Printf("channels=%d\n", result.Channels)
	fmt.Printf("bits_per_channel=%d\n", result.BitsPerChannel)
	fmt.Printf("bytes_per_frame=%d\n", result.BytesPerFrame)
	fmt.Printf("duration=%s\n", result.Duration)
	fmt.Printf("capture_callbacks=%d\n", result.Callbacks)
	fmt.Printf("capture_frames=%d\n", result.Frames)
	fmt.Printf("render_callbacks=%d\n", result.RenderCallbacks)
	fmt.Printf("render_frames=%d\n", result.RenderFrames)
	fmt.Printf("underruns=%d\n", result.Underruns)
	fmt.Printf("overflows=%d\n", result.Overflows)
}
