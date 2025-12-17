// Package telemetry provides telemetry adapter interfaces.
package telemetry

import (
	"context"
	"time"
)

// Telemetry defines the telemetry adapter interface.
type Telemetry interface {
	// RecordQuery records a query execution.
	RecordQuery(ctx context.Context, info QueryInfo)

	// RecordError records an error.
	RecordError(ctx context.Context, info ErrorInfo)

	// RecordConnection records a connection event.
	RecordConnection(ctx context.Context, info ConnectionInfo)

	// Flush flushes any buffered telemetry data.
	Flush(ctx context.Context) error

	// Close closes the telemetry adapter.
	Close(ctx context.Context) error
}

// QueryInfo contains information about a query.
type QueryInfo struct {
	// Model is the model being queried.
	Model string

	// Operation is the operation type (findMany, create, update, etc.).
	Operation string

	// Duration is how long the query took.
	Duration time.Duration

	// Success indicates if the query succeeded.
	Success bool

	// RowsAffected is the number of rows affected.
	RowsAffected int64
}

// ErrorInfo contains information about an error.
type ErrorInfo struct {
	// Error is the error that occurred.
	Error error

	// Model is the model involved (if applicable).
	Model string

	// Operation is the operation that failed.
	Operation string

	// Query is the SQL query (if applicable).
	Query string
}

// ConnectionInfo contains information about a connection event.
type ConnectionInfo struct {
	// Event is the event type (connect, disconnect, error).
	Event string

	// Duration is how long the operation took.
	Duration time.Duration

	// Success indicates if the operation succeeded.
	Success bool

	// ActiveConnections is the number of active connections.
	ActiveConnections int
}

// Config holds telemetry configuration.
type Config struct {
	// Type is the telemetry type (noop, prometheus, opentelemetry).
	Type string

	// ServiceName is the name of the service for tracing.
	ServiceName string

	// Endpoint is the telemetry endpoint URL.
	Endpoint string

	// SampleRate is the sampling rate for traces (0.0-1.0).
	SampleRate float64

	// EnableMetrics enables metrics collection.
	EnableMetrics bool

	// EnableTracing enables distributed tracing.
	EnableTracing bool
}
