// Package client provides middleware support.
package client

import (
	"context"
	"time"
)

// MiddlewareClient provides middleware registration.
type MiddlewareClient interface {
	// Use adds middleware to the client's middleware chain.
	Use(mw MiddlewareFunc)
}

// MiddlewareFunc is the middleware function signature.
type MiddlewareFunc func(ctx context.Context, params MiddlewareParams, next MiddlewareNext) MiddlewareResult

// MiddlewareNext is the function to call to continue the middleware chain.
type MiddlewareNext func(ctx context.Context) MiddlewareResult

// MiddlewareParams contains information about the current operation.
type MiddlewareParams struct {
	// Model is the model being operated on.
	Model string

	// Operation is the operation type (findMany, create, update, etc.).
	Operation string

	// Args are the operation arguments.
	Args map[string]interface{}

	// StartTime is when the operation started.
	StartTime time.Time
}

// MiddlewareResult contains the result of a middleware operation.
type MiddlewareResult struct {
	// Data is the result data.
	Data interface{}

	// Error is any error that occurred.
	Error error

	// Duration is how long the operation took.
	Duration time.Duration
}

// LogMiddleware creates a middleware that logs operations.
func LogMiddleware(logger Logger) MiddlewareFunc {
	return func(ctx context.Context, params MiddlewareParams, next MiddlewareNext) MiddlewareResult {
		logger.Info("Operation started", Field{Key: "model", Value: params.Model}, Field{Key: "operation", Value: params.Operation})

		result := next(ctx)

		if result.Error != nil {
			logger.Error("Operation failed",
				Field{Key: "model", Value: params.Model},
				Field{Key: "operation", Value: params.Operation},
				Field{Key: "duration", Value: result.Duration},
				Field{Key: "error", Value: result.Error},
			)
		} else {
			logger.Info("Operation completed",
				Field{Key: "model", Value: params.Model},
				Field{Key: "operation", Value: params.Operation},
				Field{Key: "duration", Value: result.Duration},
			)
		}

		return result
	}
}

// TimeoutMiddleware creates a middleware that enforces timeouts.
func TimeoutMiddleware(timeout time.Duration) MiddlewareFunc {
	return func(ctx context.Context, params MiddlewareParams, next MiddlewareNext) MiddlewareResult {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return next(ctx)
	}
}

// RetryMiddleware creates a middleware that retries on failure.
func RetryMiddleware(maxRetries int, delay time.Duration) MiddlewareFunc {
	return func(ctx context.Context, params MiddlewareParams, next MiddlewareNext) MiddlewareResult {
		var result MiddlewareResult
		for attempt := 0; attempt <= maxRetries; attempt++ {
			result = next(ctx)
			if result.Error == nil {
				return result
			}

			// Don't retry on certain errors
			if IsNotFound(result.Error) || IsUniqueConstraint(result.Error) {
				return result
			}

			if attempt < maxRetries {
				time.Sleep(delay)
			}
		}
		return result
	}
}
