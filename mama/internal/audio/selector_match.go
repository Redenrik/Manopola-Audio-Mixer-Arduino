package audio

import (
	"path/filepath"
	"strings"

	"mama/internal/config"
)

func normalizeMatchValues(values ...string) []string {
	seen := make(map[string]struct{}, len(values)*3)
	out := make([]string, 0, len(values)*3)
	appendValue := func(value string) {
		v := canonicalizeMatchValue(value)
		if v == "" {
			return
		}
		if _, exists := seen[v]; exists {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, value := range values {
		base := canonicalizeMatchValue(value)
		if base == "" {
			continue
		}
		appendValue(base)
		for _, alias := range derivedMatchAliases(base) {
			appendValue(alias)
		}
	}
	return out
}

func selectorMatchesAnyValue(selector config.Selector, values []string) bool {
	needle := canonicalizeMatchValue(selector.Value)
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

func canonicalizeMatchValue(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return ""
	}
	v = strings.ReplaceAll(v, "\\", "/")
	v = strings.ReplaceAll(v, "//", "/")
	return v
}

func derivedMatchAliases(value string) []string {
	if value == "" {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, 8)
	appendAlias := func(alias string) {
		alias = canonicalizeMatchValue(alias)
		if alias == "" || !shouldIncludeDerivedAlias(alias) {
			return
		}
		if _, exists := seen[alias]; exists {
			return
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}

	base := filepath.Base(value)
	appendAlias(strings.TrimSuffix(base, filepath.Ext(base)))
	if strings.Contains(value, "/") {
		parts := strings.Split(value, "/")
		for _, part := range parts {
			part = strings.TrimSuffix(part, filepath.Ext(part))
			appendAlias(part)
		}
	}
	if strings.Contains(value, ".") {
		for _, part := range strings.Split(value, ".") {
			appendAlias(part)
		}
	}
	return out
}

func shouldIncludeDerivedAlias(alias string) bool {
	if len(alias) < 2 {
		return false
	}
	switch alias {
	case "com", "org", "net", "io", "app", "apps", "contents", "macos", "frameworks", "versions", "current", "helper":
		return false
	default:
		return true
	}
}
