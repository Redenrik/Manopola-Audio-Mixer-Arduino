//go:build darwin

package audio

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type darwinMicVolumeController struct {
	run commandRunnerScript

	mu          sync.Mutex
	lastNonZero int
}

type commandRunnerScript func(script string) (string, error)

func newMicVolumeController() volumeController {
	if _, err := exec.LookPath("osascript"); err != nil {
		return nil
	}
	return &darwinMicVolumeController{
		run:         runAppleScript,
		lastNonZero: 50,
	}
}

func runAppleScript(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("osascript failed: %s", msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *darwinMicVolumeController) GetVolume() (int, error) {
	out, err := d.run("input volume of (get volume settings)")
	if err != nil {
		return 0, err
	}
	vol, err := parseBoundedPercent(out)
	if err != nil {
		return 0, fmt.Errorf("parse macOS input volume: %w", err)
	}
	if vol > 0 {
		d.mu.Lock()
		d.lastNonZero = vol
		d.mu.Unlock()
	}
	return vol, nil
}

func (d *darwinMicVolumeController) SetVolume(v int) error {
	if v < 0 || v > 100 {
		return fmt.Errorf("invalid mic volume: %d", v)
	}
	_, err := d.run(fmt.Sprintf("set volume input volume %d", v))
	if err != nil {
		return err
	}
	if v > 0 {
		d.mu.Lock()
		d.lastNonZero = v
		d.mu.Unlock()
	}
	return nil
}

func (d *darwinMicVolumeController) GetMuted() (bool, error) {
	vol, err := d.GetVolume()
	if err != nil {
		return false, err
	}
	return vol == 0, nil
}

func (d *darwinMicVolumeController) Mute() error {
	vol, err := d.GetVolume()
	if err != nil {
		return err
	}
	if vol > 0 {
		d.mu.Lock()
		d.lastNonZero = vol
		d.mu.Unlock()
	}
	_, err = d.run("set volume input volume 0")
	return err
}

func (d *darwinMicVolumeController) Unmute() error {
	restore := 50
	d.mu.Lock()
	if d.lastNonZero > 0 {
		restore = d.lastNonZero
	}
	d.mu.Unlock()
	_, err := d.run(fmt.Sprintf("set volume input volume %d", restore))
	return err
}

func parseBoundedPercent(raw string) (int, error) {
	vol, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, err
	}
	if vol < 0 {
		vol = 0
	}
	if vol > 100 {
		vol = 100
	}
	return vol, nil
}
