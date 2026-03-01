package runtime

import "testing"

func TestMetricsSnapshot(t *testing.T) {
	m := NewMetrics()

	m.IncParseErrors()
	m.IncDroppedEvents()
	m.IncDroppedEvents()
	m.IncReconnectCount()
	m.IncReconnectCount()
	m.IncReconnectCount()
	m.IncBackendFailures()

	s := m.Snapshot()
	if s.ParseErrors != 1 {
		t.Fatalf("ParseErrors = %d, want 1", s.ParseErrors)
	}
	if s.DroppedEvents != 2 {
		t.Fatalf("DroppedEvents = %d, want 2", s.DroppedEvents)
	}
	if s.ReconnectCount != 3 {
		t.Fatalf("ReconnectCount = %d, want 3", s.ReconnectCount)
	}
	if s.BackendFailures != 1 {
		t.Fatalf("BackendFailures = %d, want 1", s.BackendFailures)
	}
}

func TestMetricsSnapshotString(t *testing.T) {
	s := MetricsSnapshot{
		ParseErrors:     4,
		DroppedEvents:   5,
		ReconnectCount:  6,
		BackendFailures: 7,
	}

	const want = "parse_errors=4 dropped_events=5 reconnect_count=6 backend_failures=7"
	if got := s.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
