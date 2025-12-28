// Package client provides transaction support.
package client

import (
	"context"
	"database/sql"
)

// TransactionClient provides transaction methods.
type TransactionClient interface {
	// Transaction executes a function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, the transaction is committed.
	Transaction(ctx context.Context, fn func(tx Tx) error) error

	// TransactionWithOptions executes a function within a transaction with options.
	TransactionWithOptions(ctx context.Context, opts TxOptions, fn func(tx Tx) error) error
}

// Tx represents an active database transaction.
type Tx interface {
	// Commit commits the transaction.
	Commit() error

	// Rollback rolls back the transaction.
	Rollback() error

	// Query executes a query within the transaction.
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// Exec executes a statement within the transaction.
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// TxOptions configures transaction behavior.
type TxOptions struct {
	// IsolationLevel sets the transaction isolation level.
	IsolationLevel IsolationLevel

	// ReadOnly marks the transaction as read-only.
	ReadOnly bool

	// MaxRetries is the maximum number of retries on serialization failure.
	MaxRetries int
}

// DefaultTxOptions returns the default transaction options.
func DefaultTxOptions() TxOptions {
	return TxOptions{
		IsolationLevel: IsolationLevelDefault,
		ReadOnly:       false,
		MaxRetries:     3,
	}
}

// IsolationLevel represents the transaction isolation level.
type IsolationLevel int

const (
	// IsolationLevelDefault uses the database default isolation level.
	IsolationLevelDefault IsolationLevel = iota

	// IsolationLevelReadUncommitted allows dirty reads.
	IsolationLevelReadUncommitted

	// IsolationLevelReadCommitted prevents dirty reads.
	IsolationLevelReadCommitted

	// IsolationLevelRepeatableRead prevents dirty and non-repeatable reads.
	IsolationLevelRepeatableRead

	// IsolationLevelSerializable provides full serializability.
	IsolationLevelSerializable
)

// ToSQLIsolationLevel converts to the standard library isolation level.
func (l IsolationLevel) ToSQLIsolationLevel() sql.IsolationLevel {
	switch l {
	case IsolationLevelReadUncommitted:
		return sql.LevelReadUncommitted
	case IsolationLevelReadCommitted:
		return sql.LevelReadCommitted
	case IsolationLevelRepeatableRead:
		return sql.LevelRepeatableRead
	case IsolationLevelSerializable:
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}
