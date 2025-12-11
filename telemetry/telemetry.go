// Package telemetry provides opt-in telemetry collection for prisma-go.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// TelemetryEvent represents a telemetry event
type TelemetryEvent struct {
	EventType    string                 `json:"event_type"`
	Command      string                 `json:"command,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
	Duration     *time.Duration         `json:"duration,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Version      string                 `json:"version"`
	OS           string                 `json:"os"`
	Architecture string                 `json:"architecture"`
}

// TelemetryCollector manages telemetry collection
type TelemetryCollector struct {
	enabled       bool
	endpoint      string
	events        []TelemetryEvent
	mu            sync.Mutex
	httpClient    *http.Client
	version       string
	batchSize     int
	flushInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

var (
	globalCollector *TelemetryCollector
	once            sync.Once
)

// InitTelemetry initializes the global telemetry collector
func InitTelemetry(version string, enabled bool) {
	once.Do(func() {
		globalCollector = &TelemetryCollector{
			enabled:       enabled && !isTelemetryDisabled(),
			endpoint:      getTelemetryEndpoint(),
			events:        make([]TelemetryEvent, 0, 100),
			httpClient:    &http.Client{Timeout: 5 * time.Second},
			version:       version,
			batchSize:     10,
			flushInterval: 30 * time.Second,
			stopChan:      make(chan struct{}),
		}

		if globalCollector.enabled {
			globalCollector.startBackgroundFlush()
		}
	})
}

// RecordCommand records a command execution event
func RecordCommand(command string, provider string, duration time.Duration, err error) {
	if globalCollector == nil || !globalCollector.enabled {
		return
	}

	event := TelemetryEvent{
		EventType:    "command",
		Command:      command,
		Provider:     provider,
		Duration:     &duration,
		Timestamp:    time.Now(),
		Version:      globalCollector.version,
		OS:           getOS(),
		Architecture: getArchitecture(),
	}

	if err != nil {
		event.Error = err.Error()
	}

	globalCollector.recordEvent(event)
}

// RecordError records an error event
func RecordError(errorType string, err error, metadata map[string]interface{}) {
	if globalCollector == nil || !globalCollector.enabled {
		return
	}

	event := TelemetryEvent{
		EventType:    "error",
		Error:        err.Error(),
		Metadata:     metadata,
		Timestamp:    time.Now(),
		Version:      globalCollector.version,
		OS:           getOS(),
		Architecture: getArchitecture(),
	}

	if metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["error_type"] = errorType

	globalCollector.recordEvent(event)
}

// RecordPerformance records a performance metric
func RecordPerformance(metric string, duration time.Duration, metadata map[string]interface{}) {
	if globalCollector == nil || !globalCollector.enabled {
		return
	}

	event := TelemetryEvent{
		EventType:    "performance",
		Duration:     &duration,
		Metadata:     metadata,
		Timestamp:    time.Now(),
		Version:      globalCollector.version,
		OS:           getOS(),
		Architecture: getArchitecture(),
	}

	if metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["metric"] = metric

	globalCollector.recordEvent(event)
}

// recordEvent adds an event to the collector
func (tc *TelemetryCollector) recordEvent(event TelemetryEvent) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.events = append(tc.events, event)

	// Flush if batch size reached
	if len(tc.events) >= tc.batchSize {
		go tc.flush()
	}
}

// flush sends collected events to the telemetry endpoint
func (tc *TelemetryCollector) flush() {
	tc.mu.Lock()
	if len(tc.events) == 0 {
		tc.mu.Unlock()
		return
	}

	events := make([]TelemetryEvent, len(tc.events))
	copy(events, tc.events)
	tc.events = tc.events[:0]
	tc.mu.Unlock()

	// Send events asynchronously
	go tc.sendEvents(events)
}

// sendEvents sends events to the telemetry endpoint
func (tc *TelemetryCollector) sendEvents(events []TelemetryEvent) {
	if len(events) == 0 {
		return
	}

	payload := map[string]interface{}{
		"events": events,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		// Silently fail - telemetry should never break the application
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", tc.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("prisma-go/%s", tc.version))

	// Send request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req = req.WithContext(ctx)
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Read response to completion (but ignore it)
	io.Copy(io.Discard, resp.Body)
}

// startBackgroundFlush starts a background goroutine to flush events periodically
func (tc *TelemetryCollector) startBackgroundFlush() {
	tc.wg.Add(1)
	go func() {
		defer tc.wg.Done()
		ticker := time.NewTicker(tc.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				tc.flush()
			case <-tc.stopChan:
				// Final flush before stopping
				tc.flush()
				return
			}
		}
	}()
}

// Shutdown stops the telemetry collector and flushes remaining events
func Shutdown() {
	if globalCollector == nil {
		return
	}

	close(globalCollector.stopChan)
	globalCollector.wg.Wait()
	globalCollector.flush()
}

// isTelemetryDisabled checks if telemetry is disabled via environment variable or flag
func isTelemetryDisabled() bool {
	// Check environment variable
	if os.Getenv("PRISMA_TELEMETRY_DISABLED") == "1" || os.Getenv("PRISMA_TELEMETRY_DISABLED") == "true" {
		return true
	}

	// Check for --no-telemetry flag in command line args
	for _, arg := range os.Args {
		if arg == "--no-telemetry" {
			return true
		}
	}

	return false
}

// getTelemetryEndpoint returns the telemetry endpoint URL
func getTelemetryEndpoint() string {
	endpoint := os.Getenv("PRISMA_TELEMETRY_ENDPOINT")
	if endpoint == "" {
		// Default endpoint (can be configured)
		return "https://telemetry.prisma-go.dev/events"
	}
	return endpoint
}

// getOS returns the operating system name
func getOS() string {
	return os.Getenv("GOOS")
}

// getArchitecture returns the architecture
func getArchitecture() string {
	return os.Getenv("GOARCH")
}

// IsEnabled returns whether telemetry is enabled
func IsEnabled() bool {
	return globalCollector != nil && globalCollector.enabled
}
