// Package client provides transaction support.
package client

import (
	"context"
	"database/sql"
	"fmt"
)

// IsolationLevel represents transaction isolation levels
type IsolationLevel int

const (
	// ReadUncommitted allows dirty reads
	ReadUncommitted IsolationLevel = iota
	// ReadCommitted prevents dirty reads (default)
	ReadCommitted
	// RepeatableRead prevents dirty reads and non-repeatable reads
	RepeatableRead
	// Serializable prevents dirty reads, non-repeatable reads, and phantom reads
	Serializable
)

// ToSQLIsolationLevel converts IsolationLevel to sql.IsolationLevel
func (level IsolationLevel) ToSQLIsolationLevel() sql.IsolationLevel {
	switch level {
	case ReadUncommitted:
		return sql.LevelReadUncommitted
	case ReadCommitted:
		return sql.LevelReadCommitted
	case RepeatableRead:
		return sql.LevelRepeatableRead
	case Serializable:
		return sql.LevelSerializable
	default:
		return sql.LevelReadCommitted
	}
}

// NewTxOptions creates sql.TxOptions from isolation level
func NewTxOptions(isolation IsolationLevel, readOnly bool) *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: isolation.ToSQLIsolationLevel(),
		ReadOnly:  readOnly,
	}
}

// Tx wraps sql.Tx and provides Prisma-specific transaction methods
type Tx struct {
	*sql.Tx
	db       *sql.DB
	provider string
	depth    int // Track nesting depth for savepoints
}

// TransactionFunc is a function that runs within a transaction
type TransactionFunc func(tx *Tx) error

// Transaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
// Supports nested transactions using savepoints
func (c *PrismaClient) Transaction(ctx context.Context, fn TransactionFunc) error {
	return c.TransactionWithOptions(ctx, nil, fn)
}

// TransactionWithTxAndOptions executes a function with a Tx wrapper and custom options
func (c *PrismaClient) TransactionWithTxAndOptions(ctx context.Context, opts *sql.TxOptions, fn TransactionFunc) error {
	// Begin transaction
	sqlTx, err := c.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	tx := &Tx{
		Tx:       sqlTx,
		db:       c.db,
		provider: c.provider,
		depth:    0,
	}

	// Defer rollback in case of panic
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// TransactionWithOptions executes a transaction with custom options
func (c *PrismaClient) TransactionWithOptions(ctx context.Context, opts *sql.TxOptions, fn TransactionFunc) error {
	return c.TransactionWithTxAndOptions(ctx, opts, fn)
}

// NestedTransaction executes a nested transaction using savepoints
// This allows transactions within transactions
func (tx *Tx) NestedTransaction(ctx context.Context, fn TransactionFunc) error {
	tx.depth++
	savepointName := fmt.Sprintf("sp_%d", tx.depth)

	// Create savepoint
	_, err := tx.ExecContext(ctx, fmt.Sprintf("SAVEPOINT %s", savepointName))
	if err != nil {
		tx.depth--
		return fmt.Errorf("failed to create savepoint: %w", err)
	}

	// Defer rollback to savepoint in case of panic
	defer func() {
		if p := recover(); p != nil {
			_, _ = tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName))
			tx.depth--
			panic(p)
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		// Rollback to savepoint on error
		if _, rbErr := tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName)); rbErr != nil {
			tx.depth--
			return fmt.Errorf("nested transaction error: %v, rollback error: %w", err, rbErr)
		}
		tx.depth--
		return err
	}

	// Release savepoint (commit nested transaction)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", savepointName)); err != nil {
		tx.depth--
		return fmt.Errorf("failed to release savepoint: %w", err)
	}

	tx.depth--
	return nil
}

// TransactionWithIsolation executes a transaction with a specific isolation level
func (c *PrismaClient) TransactionWithIsolation(ctx context.Context, isolation IsolationLevel, fn TransactionFunc) error {
	opts := NewTxOptions(isolation, false)
	return c.TransactionWithOptions(ctx, opts, fn)
}

// ReadOnlyTransaction executes a read-only transaction
func (c *PrismaClient) ReadOnlyTransaction(ctx context.Context, fn TransactionFunc) error {
	opts := &sql.TxOptions{
		ReadOnly: true,
	}
	return c.TransactionWithOptions(ctx, opts, fn)
}

// ReadOnlyTransactionWithIsolation executes a read-only transaction with a specific isolation level
func (c *PrismaClient) ReadOnlyTransactionWithIsolation(ctx context.Context, isolation IsolationLevel, fn TransactionFunc) error {
	opts := NewTxOptions(isolation, true)
	return c.TransactionWithOptions(ctx, opts, fn)
}
