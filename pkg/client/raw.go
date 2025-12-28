// Package client provides raw SQL execution support.
package client

import (
	"context"
	"database/sql"
)

// RawClient provides raw SQL execution methods.
type RawClient interface {
	// QueryRaw executes a raw SQL query and returns rows.
	QueryRaw(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// ExecuteRaw executes a raw SQL statement.
	ExecuteRaw(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// QueryRawUnsafe executes a raw SQL query without escaping (use with caution).
	QueryRawUnsafe(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// ExecuteRawUnsafe executes a raw SQL statement without escaping (use with caution).
	ExecuteRawUnsafe(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// RawResult represents the result of a raw SQL query.
type RawResult struct {
	// Columns are the column names.
	Columns []string

	// Rows are the query result rows.
	Rows []map[string]interface{}

	// RowsAffected is the number of affected rows (for mutations).
	RowsAffected int64

	// LastInsertID is the last inserted ID (if applicable).
	LastInsertID int64
}

// RawQuery represents a raw SQL query with parameters.
type RawQuery struct {
	// SQL is the query string.
	SQL string

	// Args are the query arguments.
	Args []interface{}
}

// NewRawQuery creates a new raw query.
func NewRawQuery(sql string, args ...interface{}) *RawQuery {
	return &RawQuery{
		SQL:  sql,
		Args: args,
	}
}
