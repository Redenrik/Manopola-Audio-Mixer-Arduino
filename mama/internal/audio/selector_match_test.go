package audio

import (
	"testing"

	"mama/internal/config"
)

func TestSelectorMatchesAnyValue_NormalizesPathSeparators(t *testing.T) {
	values := normalizeMatchValues("/Users/tester/Library/Application Support/Steam/steamapps/common/RimWorld/RimWorldMac.app/Contents/MacOS/RimWorld")
	selector := config.Selector{Kind: config.SelectorContains, Value: `steamapps\common`}
	if !selectorMatchesAnyValue(selector, values) {
		t.Fatalf("selector %+v did not match values %v", selector, values)
	}
}

func TestSelectorMatchesAnyValue_DerivesBundleAndAppAliases(t *testing.T) {
	values := normalizeMatchValues(
		"Google Chrome Helper (Renderer)",
		"com.google.Chrome.helper",
		"/Applications/Google Chrome.app/Contents/Frameworks/Google Chrome Framework.framework/Helpers/Google Chrome Helper (Renderer).app/Contents/MacOS/Google Chrome Helper (Renderer)",
	)

	cases := []config.Selector{
		{Kind: config.SelectorExe, Value: "chrome"},
		{Kind: config.SelectorExact, Value: "google chrome"},
		{Kind: config.SelectorContains, Value: "google chrome helper"},
	}
	for _, selector := range cases {
		if !selectorMatchesAnyValue(selector, values) {
			t.Fatalf("selector %+v did not match values %v", selector, values)
		}
	}
}
