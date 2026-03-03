package serialx

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

type readResult struct {
	data []byte
	err  error
}

type fakePort struct {
	mu         sync.Mutex
	results    []readResult
	readCall   int
	defaultErr error
}

func (f *fakePort) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.results) == 0 {
		return 0, f.defaultErr
	}
	res := f.results[0]
	f.results = f.results[1:]
	f.readCall++
	if len(res.data) > 0 {
		copy(p, res.data)
	}
	return len(res.data), res.err
}

func (f *fakePort) Close() error { return nil }

func TestReadLinesToleratesErrNoProgressAndEmitsLines(t *testing.T) {
	port := &fakePort{results: []readResult{
		{err: io.ErrNoProgress},
		{err: io.ErrNoProgress},
		{data: []byte("E1:+1\n")},
		{err: io.ErrNoProgress},
		{data: []byte("B1:1\n")},
	}, defaultErr: io.ErrNoProgress}

	var sleepCalls int
	r := &Reader{
		port: port,
		sleep: func(time.Duration) {
			sleepCalls++
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan string, 8)
	errC := make(chan error, 1)
	go func() {
		errC <- r.ReadLines(ctx, out)
	}()

	var got []string
	for line := range out {
		got = append(got, line)
		if len(got) == 2 {
			cancel()
		}
	}

	err := <-errC
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	want := []string{"E1:+1", "B1:1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lines: got %v, want %v", got, want)
	}
	if sleepCalls < 3 {
		t.Fatalf("expected idle backoff sleeps for no-progress reads, got %d", sleepCalls)
	}
}

func TestReadLinesBacksOffOnZeroByteReads(t *testing.T) {
	port := &fakePort{results: []readResult{
		{},
		{},
		{data: []byte("V:1\n")},
	}, defaultErr: io.ErrNoProgress}

	var sleepCalls int
	r := &Reader{
		port: port,
		sleep: func(time.Duration) {
			sleepCalls++
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan string, 4)
	errC := make(chan error, 1)
	go func() {
		errC <- r.ReadLines(ctx, out)
	}()

	line := <-out
	if line != "V:1" {
		t.Fatalf("unexpected line: %q", line)
	}
	cancel()

	for range out {
	}
	if !errors.Is(<-errC, context.Canceled) {
		t.Fatalf("expected context cancellation")
	}
	if sleepCalls < 2 {
		t.Fatalf("expected at least 2 sleep calls for zero-byte reads, got %d", sleepCalls)
	}
}

func TestReadLinesSupportsCRAndNulDelimiters(t *testing.T) {
	port := &fakePort{results: []readResult{
		{data: []byte("E1:+1\rB1:1\x00V:1\r\n")},
	}, defaultErr: io.ErrNoProgress}

	r := &Reader{
		port: port,
		sleep: func(time.Duration) {
			// no-op
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan string, 8)
	errC := make(chan error, 1)
	go func() {
		errC <- r.ReadLines(ctx, out)
	}()

	var got []string
	for line := range out {
		got = append(got, line)
		if len(got) == 3 {
			cancel()
		}
	}

	if !errors.Is(<-errC, context.Canceled) {
		t.Fatalf("expected context cancellation")
	}

	want := []string{"E1:+1", "B1:1", "V:1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lines: got %v, want %v", got, want)
	}
}

func TestIsIdleReadError(t *testing.T) {
	timeoutErr := &net.DNSError{IsTimeout: true}
	temporaryErr := temporaryReadError{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "io err no progress", err: io.ErrNoProgress, want: true},
		{name: "network timeout", err: timeoutErr, want: true},
		{name: "temporary", err: temporaryErr, want: true},
		{name: "eof", err: io.EOF, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIdleReadError(tt.err); got != tt.want {
				t.Fatalf("isIdleReadError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type temporaryReadError struct{}

func (temporaryReadError) Error() string   { return "temporary read error" }
func (temporaryReadError) Temporary() bool { return true }
