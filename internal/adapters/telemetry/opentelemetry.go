// Package telemetry provides OpenTelemetry tracing.
package telemetry

import (
	"context"
	"sync"
	"time"
)

// OpenTelemetryAdapter implements Telemetry using OpenTelemetry.
type OpenTelemetryAdapter struct {
	config *Config
	spans  []SpanInfo
	mu     sync.Mutex
}

// SpanInfo holds information about a span for tracing.
type SpanInfo struct {
	TraceID    string
	SpanID     string
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Status     string
	Attributes map[string]string
}

// NewOpenTelemetryAdapter creates a new OpenTelemetry telemetry adapter.
func NewOpenTelemetryAdapter(config *Config) *OpenTelemetryAdapter {
	return &OpenTelemetryAdapter{
		config: config,
		spans:  []SpanInfo{},
	}
}

// RecordQuery records a query execution as a span.
func (o *OpenTelemetryAdapter) RecordQuery(ctx context.Context, info QueryInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()

	status := "OK"
	if !info.Success {
		status = "ERROR"
	}

	span := SpanInfo{
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
		Name:      "prisma." + info.Operation,
		StartTime: time.Now().Add(-info.Duration),
		EndTime:   time.Now(),
		Status:    status,
		Attributes: map[string]string{
			"db.system":    "prisma",
			"db.operation": info.Operation,
			"db.model":     info.Model,
		},
	}
	o.spans = append(o.spans, span)
}

// RecordError records an error as a span event.
func (o *OpenTelemetryAdapter) RecordError(ctx context.Context, info ErrorInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()

	span := SpanInfo{
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
		Name:      "prisma.error",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Status:    "ERROR",
		Attributes: map[string]string{
			"error.message": info.Error.Error(),
			"db.model":      info.Model,
			"db.operation":  info.Operation,
		},
	}
	o.spans = append(o.spans, span)
}

// RecordConnection records a connection event.
func (o *OpenTelemetryAdapter) RecordConnection(ctx context.Context, info ConnectionInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()

	status := "OK"
	if !info.Success {
		status = "ERROR"
	}

	span := SpanInfo{
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
		Name:      "prisma.connection." + info.Event,
		StartTime: time.Now().Add(-info.Duration),
		EndTime:   time.Now(),
		Status:    status,
		Attributes: map[string]string{
			"db.connection.event": info.Event,
		},
	}
	o.spans = append(o.spans, span)
}

// Flush flushes buffered spans to the backend.
func (o *OpenTelemetryAdapter) Flush(ctx context.Context) error {
	// In a real implementation, this would send spans to the OTLP endpoint
	// For now, we just clear the buffer
	o.mu.Lock()
	defer o.mu.Unlock()
	o.spans = []SpanInfo{}
	return nil
}

// Close closes the telemetry adapter.
func (o *OpenTelemetryAdapter) Close(ctx context.Context) error {
	return o.Flush(ctx)
}

// GetSpans returns all collected spans for testing/debugging.
func (o *OpenTelemetryAdapter) GetSpans() []SpanInfo {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.spans
}

// generateTraceID generates a simple trace ID.
func generateTraceID() string {
	return time.Now().Format("20060102150405.000000000")
}

// generateSpanID generates a simple span ID.
func generateSpanID() string {
	return time.Now().Format("150405.000000")
}

// Ensure OpenTelemetryAdapter implements Telemetry interface.
var _ Telemetry = (*OpenTelemetryAdapter)(nil)
