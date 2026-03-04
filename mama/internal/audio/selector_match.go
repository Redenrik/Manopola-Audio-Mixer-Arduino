package audio

import (
	"path/filepath"
	"strings"

	"mama/internal/config"
)

func normalizeMatchValues(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.ToLower(strings.TrimSpace(value))
		if v == "" {
			continue
		}
		if _, exists := seen[v]; exists {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func selectorMatchesAnyValue(selector config.Selector, values []string) bool {
	needle := strings.ToLower(strings.TrimSpace(selector.Value))
	if needle == "" || len(values) == 0 {
		return false
	}

	switch selector.Kind {
	case config.SelectorExact, config.SelectorExe:
		for _, candidate := range values {
			if candidate == needle {
				return true
			}
		}
		return false
	case config.SelectorContains:
		for _, candidate := range values {
			if strings.Contains(candidate, needle) {
				return true
			}
		}
		return false
	case config.SelectorPrefix:
		for _, candidate := range values {
			if strings.HasPrefix(candidate, needle) {
				return true
			}
		}
		return false
	case config.SelectorSuffix:
		for _, candidate := range values {
			if strings.HasSuffix(candidate, needle) {
				return true
			}
		}
		return false
	case config.SelectorGlob:
		for _, candidate := range values {
			ok, err := filepath.Match(needle, candidate)
			if err == nil && ok {
				return true
			}
		}
		return false
	default:
		return false
	}
}
