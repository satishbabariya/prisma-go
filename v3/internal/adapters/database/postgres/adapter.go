// Package postgres implements PostgreSQL database adapter.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
)

// PostgresAdapter implements the database.Adapter interface for PostgreSQL.
type PostgresAdapter struct {
	db     *sql.DB
	config database.Config
}

// NewPostgresAdapter creates a new PostgreSQL adapter.
func NewPostgresAdapter(config database.Config) (*PostgresAdapter, error) {
	return &PostgresAdapter{
		config: config,
	}, nil
}

// Connect establishes a connection to the PostgreSQL database.
func (a *PostgresAdapter) Connect(ctx context.Context) error {
	db, err := sql.Open("postgres", a.config.URL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(a.config.MaxConnections)
	db.SetMaxIdleConns(a.config.MaxConnections / 2)
	db.SetConnMaxIdleTime(time.Duration(a.config.MaxIdleTime) * time.Second)

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, time.Duration(a.config.ConnectTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = db
	return nil
}

// Disconnect closes the database connection.
func (a *PostgresAdapter) Disconnect(ctx context.Context) error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Execute executes a query without returning rows.
func (a *PostgresAdapter) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (a *PostgresAdapter) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row.
func (a *PostgresAdapter) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if a.db == nil {
		return nil
	}
	return a.db.QueryRowContext(ctx, query, args...)
}

// Begin starts a new transaction.
func (a *PostgresAdapter) Begin(ctx context.Context) (database.Transaction, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgresTransaction{tx: tx}, nil
}

// Ping checks if the database connection is alive.
func (a *PostgresAdapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("database not connected")
	}
	return a.db.PingContext(ctx)
}

// GetDialect returns the SQL dialect.
func (a *PostgresAdapter) GetDialect() database.SQLDialect {
	return database.PostgreSQL
}

// PostgresTransaction implements the database.Transaction interface.
type PostgresTransaction struct {
	tx *sql.Tx
}

// Commit commits the transaction.
func (t *PostgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *PostgresTransaction) Rollback() error {
	return t.tx.Rollback()
}

// Execute executes a query within the transaction.
func (t *PostgresTransaction) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// Query executes a query within the transaction.
func (t *PostgresTransaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// Ensure PostgresAdapter implements Adapter interface.
var _ database.Adapter = (*PostgresAdapter)(nil)

// Ensure PostgresTransaction implements Transaction interface.
var _ database.Transaction = (*PostgresTransaction)(nil)
