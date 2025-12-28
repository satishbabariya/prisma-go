// Package sqlite implements SQLite database adapter.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
)

// SQLiteAdapter implements the database.Adapter interface for SQLite.
type SQLiteAdapter struct {
	db     *sql.DB
	config database.Config
}

// NewSQLiteAdapter creates a new SQLite adapter.
func NewSQLiteAdapter(config database.Config) (*SQLiteAdapter, error) {
	return &SQLiteAdapter{
		config: config,
	}, nil
}

// Connect establishes a connection to the SQLite database.
func (a *SQLiteAdapter) Connect(ctx context.Context) error {
	// SQLite URL is typically just a file path
	db, err := sql.Open("sqlite3", a.config.URL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite has different connection pool requirements
	// It's generally recommended to use a single connection for writes
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(time.Duration(a.config.MaxIdleTime) * time.Second)

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, time.Duration(a.config.ConnectTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys (disabled by default in SQLite)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	a.db = db
	return nil
}

// Disconnect closes the database connection.
func (a *SQLiteAdapter) Disconnect(ctx context.Context) error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Execute executes a query without returning rows.
func (a *SQLiteAdapter) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (a *SQLiteAdapter) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row.
func (a *SQLiteAdapter) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if a.db == nil {
		return nil
	}
	return a.db.QueryRowContext(ctx, query, args...)
}

// Begin starts a new transaction.
func (a *SQLiteAdapter) Begin(ctx context.Context) (database.Transaction, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &SQLiteTransaction{tx: tx}, nil
}

// Ping checks if the database connection is alive.
func (a *SQLiteAdapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("database not connected")
	}
	return a.db.PingContext(ctx)
}

// GetDialect returns the SQL dialect.
func (a *SQLiteAdapter) GetDialect() database.SQLDialect {
	return database.SQLite
}

// SQLiteTransaction implements the database.Transaction interface.
type SQLiteTransaction struct {
	tx *sql.Tx
}

// Commit commits the transaction.
func (t *SQLiteTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *SQLiteTransaction) Rollback() error {
	return t.tx.Rollback()
}

// Execute executes a query within the transaction.
func (t *SQLiteTransaction) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// Query executes a query within the transaction.
func (t *SQLiteTransaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// Ensure SQLiteAdapter implements Adapter interface.
var _ database.Adapter = (*SQLiteAdapter)(nil)

// Ensure SQLiteTransaction implements Transaction interface.
var _ database.Transaction = (*SQLiteTransaction)(nil)
