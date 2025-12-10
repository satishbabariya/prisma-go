// Package client provides middleware support for query hooks.
package client

import (
	"context"
	"time"
)

// QueryEvent represents a query execution event
type QueryEvent struct {
	Query    string
	Args     []interface{}
	Duration time.Duration
	Error    error
	Start    time.Time
	End      time.Time
}

// Middleware is a function that intercepts queries
type Middleware func(ctx context.Context, event *QueryEvent, next func() error) error

// PrismaClientWithMiddleware wraps PrismaClient with middleware support
type PrismaClientWithMiddleware struct {
	*PrismaClient
	middlewares []Middleware
}

// NewPrismaClientWithMiddleware creates a client with middleware support
func NewPrismaClientWithMiddleware(client *PrismaClient) *PrismaClientWithMiddleware {
	return &PrismaClientWithMiddleware{
		PrismaClient: client,
		middlewares:  []Middleware{},
	}
}

// Use adds a middleware to the chain
func (c *PrismaClientWithMiddleware) Use(middleware Middleware) {
	c.middlewares = append(c.middlewares, middleware)
}

// executeWithMiddleware executes a query with middleware chain
func (c *PrismaClientWithMiddleware) executeWithMiddleware(ctx context.Context, query string, args []interface{}, exec func() error) error {
	if len(c.middlewares) == 0 {
		return exec()
	}

	event := &QueryEvent{
		Query: query,
		Args:  args,
		Start: time.Now(),
	}

	var next func() error
	index := 0

	next = func() error {
		if index >= len(c.middlewares) {
			// Last middleware, execute the actual query
			err := exec()
			event.End = time.Now()
			event.Duration = event.End.Sub(event.Start)
			event.Error = err
			return err
		}

		middleware := c.middlewares[index]
		index++
		return middleware(ctx, event, next)
	}

	return next()
}

// LoggingMiddleware creates a middleware that logs queries
func LoggingMiddleware(logger func(format string, args ...interface{})) Middleware {
	return func(ctx context.Context, event *QueryEvent, next func() error) error {
		logger("Executing query: %s with args: %v", event.Query, event.Args)
		err := next()
		if err != nil {
			logger("Query failed: %v", err)
		} else {
			logger("Query completed in %v", event.Duration)
		}
		return err
	}
}

// TimingMiddleware creates a middleware that measures query execution time
func TimingMiddleware(onTiming func(query string, duration time.Duration)) Middleware {
	return func(ctx context.Context, event *QueryEvent, next func() error) error {
		err := next()
		if onTiming != nil {
			onTiming(event.Query, event.Duration)
		}
		return err
	}
}

// ErrorMiddleware creates a middleware that handles errors
func ErrorMiddleware(onError func(query string, err error)) Middleware {
	return func(ctx context.Context, event *QueryEvent, next func() error) error {
		err := next()
		if err != nil && onError != nil {
			onError(event.Query, err)
		}
		return err
	}
}
