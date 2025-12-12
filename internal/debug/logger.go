// Package debug provides debug logging functionality using log/slog
package debug

import (
	"log/slog"
	"os"
	"sync"
)

var (
	// logger is the global debug logger instance
	logger *slog.Logger
	// enabled indicates if debug logging is enabled
	enabled bool
	// mu protects the logger and enabled flag
	mu sync.RWMutex
)

// Init initializes the debug logger
// If enable is true, debug logs will be written to os.Stderr
// If enable is false, debug logs will be silently discarded
func Init(enable bool) {
	mu.Lock()
	defer mu.Unlock()

	enabled = enable

	if enable {
		// Create a text handler that writes to stderr
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		handler := slog.NewTextHandler(os.Stderr, opts)
		logger = slog.New(handler)
	} else {
		// Create a no-op logger that discards all logs
		opts := &slog.HandlerOptions{
			Level: slog.LevelError + 1, // Set to a level higher than any actual level
		}
		handler := slog.NewTextHandler(os.Stderr, opts)
		logger = slog.New(handler)
	}
}

// Enabled returns whether debug logging is enabled
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Error(msg, args...)
}

// With returns a logger with the given attributes
func With(args ...any) *slog.Logger {
	mu.RLock()
	l := logger
	mu.RUnlock()
	return l.With(args...)
}

// Logger returns the underlying slog.Logger instance
func Logger() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger
}
