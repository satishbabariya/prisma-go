// Package runtime provides middleware support for Prisma clients.
package runtime

import (
	"context"
	"time"
)

// QueryInfo contains information about a query being executed.
type QueryInfo struct {
	// Model is the model being queried.
	Model string

	// Operation is the operation type (findMany, create, update, etc.).
	Operation string

	// Args are the query arguments.
	Args map[string]interface{}

	// Timestamp is when the query started.
	Timestamp time.Time

	// Duration is how long the query took (set after execution).
	Duration time.Duration
}

// QueryResult contains the result of a query.
type QueryResult struct {
	// Data is the result data.
	Data interface{}

	// Error is any error that occurred.
	Error error

	// RowsAffected is the number of rows affected (for mutations).
	RowsAffected int64
}

// Next is the function to call to continue the middleware chain.
type Next func(ctx context.Context, info QueryInfo) QueryResult

// Middleware is a function that can intercept query execution.
type Middleware func(ctx context.Context, info QueryInfo, next Next) QueryResult

// MiddlewareChain manages a chain of middleware.
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain creates a new middleware chain.
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: []Middleware{},
	}
}

// Use adds middleware to the chain.
func (mc *MiddlewareChain) Use(mw Middleware) {
	mc.middlewares = append(mc.middlewares, mw)
}

// Execute runs the middleware chain and the final handler.
func (mc *MiddlewareChain) Execute(ctx context.Context, info QueryInfo, handler func(ctx context.Context, info QueryInfo) QueryResult) QueryResult {
	info.Timestamp = time.Now()

	// If no middlewares, execute handler directly
	if len(mc.middlewares) == 0 {
		result := handler(ctx, info)
		info.Duration = time.Since(info.Timestamp)
		return result
	}

	// Build the chain
	index := 0
	var next Next
	next = func(ctx context.Context, info QueryInfo) QueryResult {
		if index < len(mc.middlewares) {
			mw := mc.middlewares[index]
			index++
			return mw(ctx, info, next)
		}
		result := handler(ctx, info)
		info.Duration = time.Since(info.Timestamp)
		return result
	}

	return next(ctx, info)
}

// LoggingMiddleware creates a middleware that logs queries.
func LoggingMiddleware(logger Logger) Middleware {
	return func(ctx context.Context, info QueryInfo, next Next) QueryResult {
		logger.Info("Query started",
			"model", info.Model,
			"operation", info.Operation,
		)

		result := next(ctx, info)

		if result.Error != nil {
			logger.Error("Query failed",
				"model", info.Model,
				"operation", info.Operation,
				"duration", info.Duration,
				"error", result.Error,
			)
		} else {
			logger.Info("Query completed",
				"model", info.Model,
				"operation", info.Operation,
				"duration", info.Duration,
			)
		}

		return result
	}
}

// MetricsMiddleware creates a middleware that records query metrics.
func MetricsMiddleware(recorder MetricsRecorder) Middleware {
	return func(ctx context.Context, info QueryInfo, next Next) QueryResult {
		result := next(ctx, info)

		recorder.RecordQuery(info.Model, info.Operation, info.Duration, result.Error == nil)

		return result
	}
}

// MetricsRecorder records query metrics.
type MetricsRecorder interface {
	// RecordQuery records a query execution.
	RecordQuery(model, operation string, duration time.Duration, success bool)
}

// TimeoutMiddleware creates a middleware that enforces query timeouts.
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(ctx context.Context, info QueryInfo, next Next) QueryResult {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resultCh := make(chan QueryResult, 1)
		go func() {
			resultCh <- next(ctx, info)
		}()

		select {
		case result := <-resultCh:
			return result
		case <-ctx.Done():
			return QueryResult{Error: ErrTimeout}
		}
	}
}

// RetryMiddleware creates a middleware that retries failed queries.
func RetryMiddleware(maxRetries int, retryDelay time.Duration) Middleware {
	return func(ctx context.Context, info QueryInfo, next Next) QueryResult {
		var result QueryResult
		for attempt := 0; attempt <= maxRetries; attempt++ {
			result = next(ctx, info)
			if result.Error == nil {
				return result
			}

			// Don't retry on certain errors
			if IsNotFound(result.Error) || IsUniqueConstraint(result.Error) {
				return result
			}

			// Wait before retrying
			if attempt < maxRetries {
				time.Sleep(retryDelay)
			}
		}
		return result
	}
}
