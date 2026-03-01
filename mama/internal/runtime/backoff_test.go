package runtime

import (
	"testing"
	"time"
)

func TestBackoffCapsAtMax(t *testing.T) {
	b := NewBackoff(200*time.Millisecond, 1200*time.Millisecond)

	got := []time.Duration{b.Next(), b.Next(), b.Next(), b.Next(), b.Next()}
	want := []time.Duration{
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1200 * time.Millisecond,
		1200 * time.Millisecond,
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("delay[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestBackoffReset(t *testing.T) {
	b := NewBackoff(time.Second, 10*time.Second)
	_ = b.Next()
	_ = b.Next()

	b.Reset()
	if got := b.Next(); got != time.Second {
		t.Fatalf("Next after reset = %s, want %s", got, time.Second)
	}
}

func TestBackoffUsesDefaultsForInvalidValues(t *testing.T) {
	b := NewBackoff(0, 100*time.Millisecond)
	if got := b.Next(); got != 500*time.Millisecond {
		t.Fatalf("Next() = %s, want default %s", got, 500*time.Millisecond)
	}
}
