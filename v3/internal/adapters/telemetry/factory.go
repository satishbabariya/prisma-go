// Package telemetry provides a factory for creating telemetry adapters.
package telemetry

import (
	"fmt"
)

// TelemetryType represents the type of telemetry.
type TelemetryType string

const (
	// TypeNoop is the no-op telemetry type.
	TypeNoop TelemetryType = "noop"

	// TypePrometheus is the Prometheus telemetry type.
	TypePrometheus TelemetryType = "prometheus"

	// TypeOpenTelemetry is the OpenTelemetry type.
	TypeOpenTelemetry TelemetryType = "opentelemetry"
)

// NewTelemetry creates a new telemetry adapter based on configuration.
func NewTelemetry(config *Config) (Telemetry, error) {
	if config == nil {
		return NewNoopTelemetry(), nil
	}

	switch TelemetryType(config.Type) {
	case TypeNoop, "":
		return NewNoopTelemetry(), nil

	case TypePrometheus:
		return NewPrometheusTelemetry(config), nil

	case TypeOpenTelemetry:
		return NewOpenTelemetryAdapter(config), nil

	default:
		return nil, fmt.Errorf("unknown telemetry type: %s", config.Type)
	}
}
