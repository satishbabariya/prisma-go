// Package telemetry provides tests for the telemetry adapters.
package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopTelemetry(t *testing.T) {
	ctx := context.Background()
	telemetry := NewNoopTelemetry()

	// All methods should work without error
	telemetry.RecordQuery(ctx, QueryInfo{
		Model:     "User",
		Operation: "findMany",
		Duration:  100 * time.Millisecond,
		Success:   true,
	})

	telemetry.RecordError(ctx, ErrorInfo{
		Error:     errors.New("test error"),
		Model:     "User",
		Operation: "create",
	})

	telemetry.RecordConnection(ctx, ConnectionInfo{
		Event:    "connect",
		Duration: 10 * time.Millisecond,
		Success:  true,
	})

	err := telemetry.Flush(ctx)
	if err != nil {
		t.Errorf("Flush should not return error, got: %v", err)
	}

	err = telemetry.Close(ctx)
	if err != nil {
		t.Errorf("Close should not return error, got: %v", err)
	}
}

func TestPrometheusTelemetry(t *testing.T) {
	ctx := context.Background()
	telemetry := NewPrometheusTelemetry(&Config{
		Type: string(TypePrometheus),
	})

	// Record some queries
	telemetry.RecordQuery(ctx, QueryInfo{
		Model:     "User",
		Operation: "findMany",
		Duration:  100 * time.Millisecond,
		Success:   true,
	})

	telemetry.RecordQuery(ctx, QueryInfo{
		Model:     "User",
		Operation: "findMany",
		Duration:  200 * time.Millisecond,
		Success:   false,
	})

	// Check metrics
	metrics := telemetry.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics should return metrics")
	}

	queryTotal, ok := metrics["query_total"].(map[string]float64)
	if !ok {
		t.Fatal("query_total should be map[string]float64")
	}

	if queryTotal["User_findMany_success"] != 1 {
		t.Errorf("Expected 1 success query, got %v", queryTotal["User_findMany_success"])
	}

	if queryTotal["User_findMany_error"] != 1 {
		t.Errorf("Expected 1 error query, got %v", queryTotal["User_findMany_error"])
	}
}

func TestOpenTelemetryAdapter(t *testing.T) {
	ctx := context.Background()
	adapter := NewOpenTelemetryAdapter(&Config{
		Type:        string(TypeOpenTelemetry),
		ServiceName: "test-service",
	})

	// Record a query
	adapter.RecordQuery(ctx, QueryInfo{
		Model:     "User",
		Operation: "create",
		Duration:  50 * time.Millisecond,
		Success:   true,
	})

	// Record an error
	adapter.RecordError(ctx, ErrorInfo{
		Error:     errors.New("test error"),
		Model:     "Post",
		Operation: "delete",
	})

	// Check spans
	spans := adapter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("Expected 2 spans, got %d", len(spans))
	}

	// Flush and check
	adapter.Flush(ctx)
	spans = adapter.GetSpans()
	if len(spans) != 0 {
		t.Errorf("Expected 0 spans after flush, got %d", len(spans))
	}
}

func TestTelemetryFactory(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantType string
		wantErr  bool
	}{
		{
			name:     "nil config returns noop",
			config:   nil,
			wantType: "*telemetry.NoopTelemetry",
		},
		{
			name:     "empty type returns noop",
			config:   &Config{Type: ""},
			wantType: "*telemetry.NoopTelemetry",
		},
		{
			name:     "noop type",
			config:   &Config{Type: "noop"},
			wantType: "*telemetry.NoopTelemetry",
		},
		{
			name:     "prometheus type",
			config:   &Config{Type: "prometheus"},
			wantType: "*telemetry.PrometheusTelemetry",
		},
		{
			name:     "opentelemetry type",
			config:   &Config{Type: "opentelemetry"},
			wantType: "*telemetry.OpenTelemetryAdapter",
		},
		{
			name:    "unknown type",
			config:  &Config{Type: "unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetry, err := NewTelemetry(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if telemetry == nil {
				t.Fatal("Expected telemetry, got nil")
			}
		})
	}
}
