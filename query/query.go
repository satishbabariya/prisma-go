// Package query provides query compilation and SQL generation.
package query

import (
	"database/sql"
)

// Compiler compiles high-level queries into SQL
type Compiler struct {
	provider string
	db       *sql.DB
}

// NewCompiler creates a new query compiler
func NewCompiler(provider string, db *sql.DB) *Compiler {
	return &Compiler{
		provider: provider,
		db:       db,
	}
}

// Query represents a compiled query
type Query struct {
	SQL  string
	Args []interface{}
}

// Result represents a query result
type Result struct {
	Rows     []map[string]interface{}
	RowCount int
}

