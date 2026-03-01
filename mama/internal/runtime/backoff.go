package runtime

import "time"

// Backoff provides bounded exponential retry delays.
type Backoff struct {
	initial time.Duration
	max     time.Duration
	next    time.Duration
}

func NewBackoff(initial, max time.Duration) Backoff {
	if initial <= 0 {
		initial = 500 * time.Millisecond
	}
	if max < initial {
		max = initial
	}
	return Backoff{initial: initial, max: max, next: initial}
}

func (b *Backoff) Next() time.Duration {
	d := b.next
	if b.next >= b.max {
		b.next = b.max
		return d
	}
	b.next *= 2
	if b.next > b.max {
		b.next = b.max
	}
	return d
}

func (b *Backoff) Reset() {
	b.next = b.initial
}
