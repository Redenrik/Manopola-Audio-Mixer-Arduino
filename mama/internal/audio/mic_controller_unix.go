//go:build !windows

package audio

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var percentPattern = regexp.MustCompile(`(\d+)%`)

type unixMicVolumeController struct {
	useAmixer bool
}

func newMicVolumeController() volumeController {
	if _, err := exec.LookPath("pactl"); err == nil {
		return &unixMicVolumeController{}
	}
	if _, err := exec.LookPath("amixer"); err == nil {
		return &unixMicVolumeController{useAmixer: true}
	}
	return nil
}

func (u *unixMicVolumeController) GetVolume() (int, error) {
	out, err := u.execGet()
	if err != nil {
		return 0, err
	}
	m := percentPattern.FindStringSubmatch(out)
	if len(m) < 2 {
		return 0, errors.New("could not parse microphone volume")
	}
	vol, err := strconv.Atoi(m[1])
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

func (u *unixMicVolumeController) SetVolume(v int) error {
	if v < 0 || v > 100 {
		return fmt.Errorf("invalid mic volume: %d", v)
	}
	var cmd *exec.Cmd
	if u.useAmixer {
		cmd = exec.Command("amixer", "set", "Capture", fmt.Sprintf("%d%%", v))
	} else {
		cmd = exec.Command("pactl", "set-source-volume", "@DEFAULT_SOURCE@", fmt.Sprintf("%d%%", v))
	}
	_, err := cmd.CombinedOutput()
	return err
}

func (u *unixMicVolumeController) GetMuted() (bool, error) {
	out, err := u.execGet()
	if err != nil {
		return false, err
	}
	if strings.Contains(out, "[off]") || strings.Contains(out, "Mute: yes") {
		return true, nil
	}
	if strings.Contains(out, "[on]") || strings.Contains(out, "Mute: no") {
		return false, nil
	}
	return false, errors.New("could not parse microphone mute state")
}

func (u *unixMicVolumeController) Mute() error {
	var cmd *exec.Cmd
	if u.useAmixer {
		cmd = exec.Command("amixer", "set", "Capture", "nocap")
	} else {
		cmd = exec.Command("pactl", "set-source-mute", "@DEFAULT_SOURCE@", "1")
	}
	_, err := cmd.CombinedOutput()
	return err
}

func (u *unixMicVolumeController) Unmute() error {
	var cmd *exec.Cmd
	if u.useAmixer {
		cmd = exec.Command("amixer", "set", "Capture", "cap")
	} else {
		cmd = exec.Command("pactl", "set-source-mute", "@DEFAULT_SOURCE@", "0")
	}
	_, err := cmd.CombinedOutput()
	return err
}

func (u *unixMicVolumeController) execGet() (string, error) {
	var cmd *exec.Cmd
	if u.useAmixer {
		cmd = exec.Command("amixer", "get", "Capture")
	} else {
		cmd = exec.Command("pactl", "list", "sources")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
