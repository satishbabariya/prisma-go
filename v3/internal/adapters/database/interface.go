// Package database defines database adapter interfaces.
package database

import (
	"context"
	"database/sql"
)

// Adapter defines the database adapter interface.
type Adapter interface {
	// Connect establishes a database connection.
	Connect(ctx context.Context) error

	// Disconnect closes the database connection.
	Disconnect(ctx context.Context) error

	// Execute executes a SQL statement.
	Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Query executes a query that returns rows.
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow executes a query that returns a single row.
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

	// Begin starts a transaction.
	Begin(ctx context.Context) (Transaction, error)

	// Ping checks the database connection.
	Ping(ctx context.Context) error

	// GetDialect returns the SQL dialect.
	GetDialect() SQLDialect
}

// Transaction defines the transaction interface.
type Transaction interface {
	// Commit commits the transaction.
	Commit() error

	// Rollback rolls back the transaction.
	Rollback() error

	// Execute executes a statement within the transaction.
	Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Query executes a query within the transaction.
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// SQLDialect represents a SQL dialect.
type SQLDialect string

const (
	// PostgreSQL dialect.
	PostgreSQL SQLDialect = "postgres"
	// MySQL dialect.
	MySQL SQLDialect = "mysql"
	// SQLite dialect.
	SQLite SQLDialect = "sqlite"
)

// Config holds database connection configuration.
type Config struct {
	Provider       string
	URL            string
	MaxConnections int
	MaxIdleTime    int // seconds
	ConnectTimeout int // seconds
}
