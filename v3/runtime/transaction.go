// Package runtime provides transaction support for Prisma clients.
package runtime

import (
	"context"
	"database/sql"
	"fmt"
)

// Transaction represents a database transaction.
type Transaction interface {
	// Commit commits the transaction.
	Commit() error

	// Rollback rolls back the transaction.
	Rollback() error

	// QueryContext executes a query within the transaction.
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// QueryRowContext executes a query that returns a single row.
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row

	// ExecContext executes a statement within the transaction.
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Tx wraps sql.Tx to implement Transaction interface.
type Tx struct {
	tx     *sql.Tx
	client *Client
}

// Commit commits the transaction.
func (t *Tx) Commit() error {
	if t.tx == nil {
		return fmt.Errorf("%w: transaction is nil", ErrTransactionFailed)
	}
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *Tx) Rollback() error {
	if t.tx == nil {
		return nil // Already rolled back or committed
	}
	return t.tx.Rollback()
}

// QueryContext executes a query within the transaction.
func (t *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if t.tx == nil {
		return nil, fmt.Errorf("%w: transaction is nil", ErrTransactionFailed)
	}

	// Log query if enabled
	if t.client != nil && t.client.config.LogQueries && t.client.config.Logger != nil {
		t.client.config.Logger.Debug("Executing query (tx)", "query", query, "args", args)
	}

	return t.tx.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row.
func (t *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if t.tx == nil {
		return nil
	}

	// Log query if enabled
	if t.client != nil && t.client.config.LogQueries && t.client.config.Logger != nil {
		t.client.config.Logger.Debug("Executing query (tx)", "query", query, "args", args)
	}

	return t.tx.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a statement within the transaction.
func (t *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if t.tx == nil {
		return nil, fmt.Errorf("%w: transaction is nil", ErrTransactionFailed)
	}

	// Log query if enabled
	if t.client != nil && t.client.config.LogQueries && t.client.config.Logger != nil {
		t.client.config.Logger.Debug("Executing statement (tx)", "query", query, "args", args)
	}

	return t.tx.ExecContext(ctx, query, args...)
}

// TransactionOptions configures transaction behavior.
type TransactionOptions struct {
	// IsolationLevel sets the transaction isolation level.
	IsolationLevel sql.IsolationLevel

	// ReadOnly marks the transaction as read-only.
	ReadOnly bool

	// MaxRetries is the maximum number of retries on serialization failure.
	MaxRetries int
}

// DefaultTransactionOptions returns default transaction options.
func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		IsolationLevel: sql.LevelDefault,
		ReadOnly:       false,
		MaxRetries:     3,
	}
}

// BeginTx starts a new transaction with options.
func (c *Client) BeginTx(ctx context.Context, opts *TransactionOptions) (*Tx, error) {
	if !c.IsConnected() {
		return nil, ErrConnectionFailed
	}

	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	sqlOpts := &sql.TxOptions{
		Isolation: opts.IsolationLevel,
		ReadOnly:  opts.ReadOnly,
	}

	tx, err := c.db.BeginTx(ctx, sqlOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransactionFailed, err)
	}

	return &Tx{
		tx:     tx,
		client: c,
	}, nil
}

// Transaction executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (c *Client) Transaction(ctx context.Context, fn func(tx Transaction) error) error {
	return c.TransactionWithOptions(ctx, nil, fn)
}

// TransactionWithOptions executes a function within a transaction with options.
func (c *Client) TransactionWithOptions(ctx context.Context, opts *TransactionOptions, fn func(tx Transaction) error) error {
	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	var lastErr error
	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		tx, err := c.BeginTx(ctx, opts)
		if err != nil {
			return err
		}

		// Execute the function
		if err := fn(tx); err != nil {
			tx.Rollback()

			// Check for serialization failure - could retry
			if isSerializationError(err) && attempt < opts.MaxRetries {
				lastErr = err
				continue
			}

			return err
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			// Check for serialization failure during commit
			if isSerializationError(err) && attempt < opts.MaxRetries {
				lastErr = err
				continue
			}
			return fmt.Errorf("%w: commit failed: %v", ErrTransactionFailed, err)
		}

		return nil
	}

	return fmt.Errorf("%w: max retries exceeded: %v", ErrTransactionFailed, lastErr)
}

// isSerializationError checks if an error is a serialization failure.
func isSerializationError(err error) bool {
	// This would need to check for database-specific error codes
	// For PostgreSQL: error code 40001 (serialization_failure)
	// For MySQL: error 1213 (deadlock), 1205 (lock wait timeout)
	return false
}

// Ensure Tx implements Transaction interface.
var _ Transaction = (*Tx)(nil)
