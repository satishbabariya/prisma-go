// Package runtime provides the base runtime for generated Prisma clients.
package runtime

import (
	"context"
)

// contextKey is a type for context keys.
type contextKey string

const (
	// txKey is the context key for transaction.
	txKey contextKey = "prisma_tx"

	// clientKey is the context key for client.
	clientKey contextKey = "prisma_client"

	// loggerKey is the context key for logger.
	loggerKey contextKey = "prisma_logger"

	// traceKey is the context key for trace ID.
	traceKey contextKey = "prisma_trace"
)

// WithTransaction stores a transaction in the context.
func WithTransaction(ctx context.Context, tx Transaction) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// TransactionFromContext retrieves a transaction from the context.
func TransactionFromContext(ctx context.Context) (Transaction, bool) {
	tx, ok := ctx.Value(txKey).(Transaction)
	return tx, ok
}

// WithClient stores a client in the context.
func WithClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, clientKey, client)
}

// ClientFromContext retrieves a client from the context.
func ClientFromContext(ctx context.Context) (*Client, bool) {
	c, ok := ctx.Value(clientKey).(*Client)
	return c, ok
}

// WithTraceID stores a trace ID in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceKey, traceID)
}

// TraceIDFromContext retrieves a trace ID from the context.
func TraceIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(traceKey).(string)
	return id, ok
}

// Logger interface for runtime logging.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// ContextWithLogger stores a logger in the context.
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext retrieves a logger from the context.
func LoggerFromContext(ctx context.Context) (Logger, bool) {
	l, ok := ctx.Value(loggerKey).(Logger)
	return l, ok
}
