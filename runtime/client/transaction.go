// Package client provides transaction support.
package client

import (
	"context"
	"database/sql"
	"fmt"
)

// TransactionFunc is a function that runs within a transaction
type TransactionFunc func(tx *sql.Tx) error

// Transaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (c *PrismaClient) Transaction(ctx context.Context, fn TransactionFunc) error {
	// Begin transaction
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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
	tx, err := c.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ReadOnlyTransaction executes a read-only transaction
func (c *PrismaClient) ReadOnlyTransaction(ctx context.Context, fn TransactionFunc) error {
	opts := &sql.TxOptions{
		ReadOnly: true,
	}
	return c.TransactionWithOptions(ctx, opts, fn)
}
