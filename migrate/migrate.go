// Package migrate provides database migration capabilities for Prisma Go.
// It handles schema comparison, migration generation, and migration execution.
package migrate

import (
	"database/sql"
)

// Engine is the main migration engine
type Engine struct {
	db *sql.DB
}

// NewEngine creates a new migration engine
func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// Migration represents a database migration
type Migration struct {
	ID        string
	Name      string
	Applied   bool
	Timestamp int64
	SQL       string
}

// MigrationPlan represents a planned migration
type MigrationPlan struct {
	Migrations []Migration
	IsSafe     bool
	Warnings   []string
}
