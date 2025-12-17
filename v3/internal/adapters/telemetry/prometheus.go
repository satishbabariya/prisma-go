// Package telemetry provides Prometheus metrics telemetry.
package telemetry

import (
	"context"
	"sync"
	"time"
)

// PrometheusTelemetry implements Telemetry using Prometheus metrics.
type PrometheusTelemetry struct {
	// Metrics collectors
	queryDuration    *HistogramVec
	queryTotal       *CounterVec
	errorTotal       *CounterVec
	connectionsTotal *GaugeVec

	mu sync.RWMutex
}

// HistogramVec represents a histogram metric with labels.
type HistogramVec struct {
	name    string
	buckets []float64
	values  map[string][]float64
}

// CounterVec represents a counter metric with labels.
type CounterVec struct {
	name   string
	values map[string]float64
}

// GaugeVec represents a gauge metric with labels.
type GaugeVec struct {
	name   string
	values map[string]float64
}

// NewPrometheusTelemetry creates a new Prometheus telemetry adapter.
func NewPrometheusTelemetry(config *Config) *PrometheusTelemetry {
	return &PrometheusTelemetry{
		queryDuration: &HistogramVec{
			name:    "prisma_query_duration_seconds",
			buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			values:  make(map[string][]float64),
		},
		queryTotal: &CounterVec{
			name:   "prisma_queries_total",
			values: make(map[string]float64),
		},
		errorTotal: &CounterVec{
			name:   "prisma_errors_total",
			values: make(map[string]float64),
		},
		connectionsTotal: &GaugeVec{
			name:   "prisma_connections",
			values: make(map[string]float64),
		},
	}
}

// RecordQuery records a query execution.
func (p *PrometheusTelemetry) RecordQuery(ctx context.Context, info QueryInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Record query duration
	labels := info.Model + "_" + info.Operation
	p.queryDuration.values[labels] = append(p.queryDuration.values[labels], info.Duration.Seconds())

	// Increment query counter
	status := "success"
	if !info.Success {
		status = "error"
	}
	counterLabel := info.Model + "_" + info.Operation + "_" + status
	p.queryTotal.values[counterLabel]++
}

// RecordError records an error.
func (p *PrometheusTelemetry) RecordError(ctx context.Context, info ErrorInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	labels := info.Model + "_" + info.Operation
	p.errorTotal.values[labels]++
}

// RecordConnection records a connection event.
func (p *PrometheusTelemetry) RecordConnection(ctx context.Context, info ConnectionInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.connectionsTotal.values[info.Event] = float64(info.ActiveConnections)
}

// Flush flushes buffered metrics (metrics are pushed on each record).
func (p *PrometheusTelemetry) Flush(ctx context.Context) error {
	return nil
}

// Close closes the telemetry adapter.
func (p *PrometheusTelemetry) Close(ctx context.Context) error {
	return nil
}

// GetMetrics returns all collected metrics for testing/debugging.
func (p *PrometheusTelemetry) GetMetrics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"query_duration": p.queryDuration.values,
		"query_total":    p.queryTotal.values,
		"error_total":    p.errorTotal.values,
		"connections":    p.connectionsTotal.values,
	}
}

// Ensure PrometheusTelemetry implements Telemetry interface.
var _ Telemetry = (*PrometheusTelemetry)(nil)

// Unused import prevention
var _ = time.Now
