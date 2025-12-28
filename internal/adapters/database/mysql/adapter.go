// Package mysql implements MySQL database adapter.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
)

// MySQLAdapter implements the database.Adapter interface for MySQL.
type MySQLAdapter struct {
	db     *sql.DB
	config database.Config
}

// NewMySQLAdapter creates a new MySQL adapter.
func NewMySQLAdapter(config database.Config) (*MySQLAdapter, error) {
	return &MySQLAdapter{
		config: config,
	}, nil
}

// Connect establishes a connection to the MySQL database.
func (a *MySQLAdapter) Connect(ctx context.Context) error {
	db, err := sql.Open("mysql", a.config.URL)
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
func (a *MySQLAdapter) Disconnect(ctx context.Context) error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Execute executes a query without returning rows.
func (a *MySQLAdapter) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (a *MySQLAdapter) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row.
func (a *MySQLAdapter) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if a.db == nil {
		return nil
	}
	return a.db.QueryRowContext(ctx, query, args...)
}

// Begin starts a new transaction.
func (a *MySQLAdapter) Begin(ctx context.Context) (database.Transaction, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &MySQLTransaction{tx: tx}, nil
}

// Ping checks if the database connection is alive.
func (a *MySQLAdapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("database not connected")
	}
	return a.db.PingContext(ctx)
}

// GetDialect returns the SQL dialect.
func (a *MySQLAdapter) GetDialect() database.SQLDialect {
	return database.MySQL
}

// MySQLTransaction implements the database.Transaction interface.
type MySQLTransaction struct {
	tx *sql.Tx
}

// Commit commits the transaction.
func (t *MySQLTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *MySQLTransaction) Rollback() error {
	return t.tx.Rollback()
}

// Execute executes a query within the transaction.
func (t *MySQLTransaction) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// Query executes a query within the transaction.
func (t *MySQLTransaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// Ensure MySQLAdapter implements Adapter interface.
var _ database.Adapter = (*MySQLAdapter)(nil)

// Ensure MySQLTransaction implements Transaction interface.
var _ database.Transaction = (*MySQLTransaction)(nil)
