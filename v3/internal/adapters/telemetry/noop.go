// Package telemetry provides a no-op telemetry implementation.
package telemetry

import (
	"context"
)

// NoopTelemetry is a no-op implementation of the Telemetry interface.
// Use this when telemetry is disabled or not needed.
type NoopTelemetry struct{}

// NewNoopTelemetry creates a new no-op telemetry adapter.
func NewNoopTelemetry() *NoopTelemetry {
	return &NoopTelemetry{}
}

// RecordQuery does nothing.
func (n *NoopTelemetry) RecordQuery(ctx context.Context, info QueryInfo) {}

// RecordError does nothing.
func (n *NoopTelemetry) RecordError(ctx context.Context, info ErrorInfo) {}

// RecordConnection does nothing.
func (n *NoopTelemetry) RecordConnection(ctx context.Context, info ConnectionInfo) {}

// Flush does nothing.
func (n *NoopTelemetry) Flush(ctx context.Context) error {
	return nil
}

// Close does nothing.
func (n *NoopTelemetry) Close(ctx context.Context) error {
	return nil
}

// Ensure NoopTelemetry implements Telemetry interface.
var _ Telemetry = (*NoopTelemetry)(nil)
