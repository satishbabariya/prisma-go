// Package client provides the public Prisma client API.
package client

import "context"

// Client is the main Prisma client interface.
type Client interface {
	// Connect establishes a connection to the database.
	Connect(ctx context.Context) error

	// Disconnect closes the database connection.
	Disconnect(ctx context.Context) error

	// Transaction executes a function within a transaction.
	Transaction(fn func(tx Transaction) error) error

	// Use adds a middleware to the client.
	Use(middleware Middleware)

	// Raw executes raw SQL.
	Raw(ctx context.Context, query string, args ...interface{}) (interface{}, error)
}

// Transaction represents a database transaction.
type Transaction interface {
	// Commit commits the transaction.
	Commit() error

	// Rollback rolls back the transaction.
	Rollback() error
}

// Middleware defines the middleware function signature.
type Middleware func(next QueryFunc) QueryFunc

// QueryFunc is the function signature for query execution.
type QueryFunc func(ctx context.Context, query string, args ...interface{}) (interface{}, error)

// Options defines client configuration options.
type Options struct {
	DatabaseURL    string
	MaxConnections int
	Middleware     []Middleware
	Logger         Logger
}

// Logger defines the logging interface.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// Field represents a log field.
type Field struct {
	Key   string
	Value interface{}
}
