package runtime

import (
	"fmt"
	"sync/atomic"
)

// Metrics captures host runtime counters for troubleshooting and health monitoring.
type Metrics struct {
	parseErrors     atomic.Uint64
	droppedEvents   atomic.Uint64
	reconnectCount  atomic.Uint64
	backendFailures atomic.Uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncParseErrors() {
	m.parseErrors.Add(1)
}

func (m *Metrics) IncDroppedEvents() {
	m.droppedEvents.Add(1)
}

func (m *Metrics) IncReconnectCount() {
	m.reconnectCount.Add(1)
}

func (m *Metrics) IncBackendFailures() {
	m.backendFailures.Add(1)
}

type MetricsSnapshot struct {
	ParseErrors     uint64
	DroppedEvents   uint64
	ReconnectCount  uint64
	BackendFailures uint64
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		ParseErrors:     m.parseErrors.Load(),
		DroppedEvents:   m.droppedEvents.Load(),
		ReconnectCount:  m.reconnectCount.Load(),
		BackendFailures: m.backendFailures.Load(),
	}
}

func (m MetricsSnapshot) String() string {
	return fmt.Sprintf(
		"parse_errors=%d dropped_events=%d reconnect_count=%d backend_failures=%d",
		m.ParseErrors,
		m.DroppedEvents,
		m.ReconnectCount,
		m.BackendFailures,
	)
}
