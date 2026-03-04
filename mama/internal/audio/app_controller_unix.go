//go:build !windows

package audio

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mama/internal/config"
)

type commandRunner func(name string, args ...string) (string, error)

type appSession struct {
	id         string
	name       string
	exe        string
	aliases    []string
	exeAliases []string
	volume     int
	isMuted    bool
}

type unixAppSessionController struct {
	run commandRunner
}

var (
	sinkInputIDPattern = regexp.MustCompile(`^Sink Input #(\d+)`)
	volumePattern      = regexp.MustCompile(`(\d+)%`)
)

func newAppSessionController() appSessionController {
	if _, err := exec.LookPath("pactl"); err != nil {
		return nil
	}
	return &unixAppSessionController{run: runCommandOutput}
}

func runCommandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return string(out), nil
}

func (u *unixAppSessionController) Adjust(selectorToken string, step float64, deltaSteps int) error {
	target, err := u.selectSession(selectorToken)
	if err != nil {
		return err
	}
	next := target.volume + int(step*100*float64(deltaSteps))
	if next < 0 {
		next = 0
	}
	if next > 100 {
		next = 100
	}
	_, err = u.run("pactl", "set-sink-input-volume", target.id, fmt.Sprintf("%d%%", next))
	return err
}

func (u *unixAppSessionController) ToggleMute(selectorToken string) error {
	target, err := u.selectSession(selectorToken)
	if err != nil {
		return err
	}
	next := "1"
	if target.isMuted {
		next = "0"
	}
	_, err = u.run("pactl", "set-sink-input-mute", target.id, next)
	return err
}

func (u *unixAppSessionController) AdjustGroup(selectors []config.Selector, step float64, deltaSteps int) error {
	targets, err := u.selectSessions(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		next := target.volume + int(step*100*float64(deltaSteps))
		if next < 0 {
			next = 0
		}
		if next > 100 {
			next = 100
		}
		if _, err := u.run("pactl", "set-sink-input-volume", target.id, fmt.Sprintf("%d%%", next)); err != nil {
			return err
		}
	}
	return nil
}

func (u *unixAppSessionController) ToggleMuteGroup(selectors []config.Selector) error {
	targets, err := u.selectSessions(selectors)
	if err != nil {
		return err
	}
	for _, target := range targets {
		next := "1"
		if target.isMuted {
			next = "0"
		}
		if _, err := u.run("pactl", "set-sink-input-mute", target.id, next); err != nil {
			return err
		}
	}
	return nil
}

func (u *unixAppSessionController) ListTargets() ([]DiscoveredTarget, error) {
	sessions, err := u.listSessions()
	if err != nil {
		return nil, err
	}
	targets := make([]DiscoveredTarget, 0, len(sessions))
	for _, s := range sessions {
		name := s.name
		if name == "" {
			name = s.exe
		}
		target := DiscoveredTarget{
			ID:       "app:" + s.id,
			Type:     config.TargetApp,
			Name:     name,
			Selector: name,
		}
		if len(s.aliases) > 0 {
			target.Aliases = append(target.Aliases, s.aliases...)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func (u *unixAppSessionController) selectSession(selectorToken string) (appSession, error) {
	sel := parseSelectorToken(selectorToken)
	sessions, err := u.listSessions()
	if err != nil {
		return appSession{}, err
	}
	for _, s := range sessions {
		if sessionMatchesSelector(s, sel) {
			return s, nil
		}
	}
	return appSession{}, fmt.Errorf("%w: no app session matched selector %q", ErrTargetUnavailable, selectorToken)
}

func (u *unixAppSessionController) selectSessions(selectors []config.Selector) ([]appSession, error) {
	sessions, err := u.listSessions()
	if err != nil {
		return nil, err
	}
	matched := make([]appSession, 0, len(sessions))
	seen := make(map[string]struct{}, len(sessions))
	for _, selector := range selectors {
		for _, s := range sessions {
			if !sessionMatchesSelector(s, selector) {
				continue
			}
			if _, exists := seen[s.id]; exists {
				continue
			}
			seen[s.id] = struct{}{}
			matched = append(matched, s)
		}
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w: no app sessions matched group selectors", ErrTargetUnavailable)
	}
	return matched, nil
}

func (u *unixAppSessionController) listSessions() ([]appSession, error) {
	out, err := u.run("pactl", "list", "sink-inputs")
	if err != nil {
		return nil, err
	}
	sessions := parsePactlSinkInputs(out)
	for i := range sessions {
		exeNoExt := strings.TrimSuffix(strings.TrimSpace(sessions[i].exe), filepath.Ext(strings.TrimSpace(sessions[i].exe)))
		sessions[i].aliases = normalizeMatchValues(sessions[i].name, sessions[i].exe, exeNoExt)
		sessions[i].exeAliases = normalizeMatchValues(sessions[i].exe, exeNoExt)
		if len(sessions[i].aliases) == 0 && len(sessions[i].exeAliases) > 0 {
			sessions[i].aliases = append(sessions[i].aliases, sessions[i].exeAliases...)
		}
	}
	return sessions, nil
}

func parsePactlSinkInputs(out string) []appSession {
	lines := strings.Split(out, "\n")
	var sessions []appSession
	var current *appSession
	flush := func() {
		if current != nil && current.id != "" {
			sessions = append(sessions, *current)
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if matches := sinkInputIDPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
			flush()
			current = &appSession{id: matches[1]}
			continue
		}
		if current == nil {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "Mute:"):
			current.isMuted = strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(trimmed, "Mute:")), "yes")
		case strings.HasPrefix(trimmed, "Volume:") && current.volume == 0:
			if m := volumePattern.FindStringSubmatch(trimmed); len(m) == 2 {
				if vol, err := strconv.Atoi(m[1]); err == nil {
					if vol < 0 {
						vol = 0
					}
					if vol > 100 {
						vol = 100
					}
					current.volume = vol
				}
			}
		case strings.HasPrefix(trimmed, "application.name ="):
			current.name = trimPactlValue(trimmed)
		case strings.HasPrefix(trimmed, "application.process.binary ="):
			current.exe = trimPactlValue(trimmed)
		}
	}
	flush()
	return sessions
}

func trimPactlValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(parts[1]), `"`)
}

func sessionMatchesSelector(s appSession, selector config.Selector) bool {
	aliases := s.aliases
	exeAliases := s.exeAliases
	if len(aliases) == 0 || len(exeAliases) == 0 {
		exeNoExt := strings.TrimSuffix(strings.TrimSpace(s.exe), filepath.Ext(strings.TrimSpace(s.exe)))
		if len(aliases) == 0 {
			aliases = normalizeMatchValues(s.name, s.exe, exeNoExt)
		}
		if len(exeAliases) == 0 {
			exeAliases = normalizeMatchValues(s.exe, exeNoExt)
		}
	}

	if selector.Kind == config.SelectorExe {
		return selectorMatchesAnyValue(selector, exeAliases)
	}
	return selectorMatchesAnyValue(selector, aliases)
}
