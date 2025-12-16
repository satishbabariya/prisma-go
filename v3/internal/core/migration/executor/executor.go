// Package executor implements migration execution.
package executor

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// MigrationExecutor implements the Executor interface.
type MigrationExecutor struct {
	db database.Adapter
}

// NewMigrationExecutor creates a new migration executor.
func NewMigrationExecutor(db database.Adapter) *MigrationExecutor {
	return &MigrationExecutor{
		db: db,
	}
}

// Execute executes a migration.
func (e *MigrationExecutor) Execute(ctx context.Context, migration *domain.Migration) error {
	if e.db == nil {
		return fmt.Errorf("database adapter not initialized")
	}

	// Start transaction
	tx, err := e.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute each SQL statement
	for i, sql := range migration.SQL {
		_, err := tx.Execute(ctx, sql)
		if err != nil {
			// Rollback on error
			tx.Rollback()
			return fmt.Errorf("failed to execute SQL statement %d: %w\nSQL: %s", i, err, sql)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back a migration.
func (e *MigrationExecutor) Rollback(ctx context.Context, migration *domain.Migration) error {
	if e.db == nil {
		return fmt.Errorf("database adapter not initialized")
	}

	// For now, rollback is not implemented
	// A full implementation would require storing reverse migrations
	// or generating rollback SQL from the changes
	return fmt.Errorf("rollback not implemented: would need reverse migration logic")
}

// ExecuteSQL executes raw SQL statements.
func (e *MigrationExecutor) ExecuteSQL(ctx context.Context, sql []string) error {
	if e.db == nil {
		return fmt.Errorf("database adapter not initialized")
	}

	// Start transaction
	tx, err := e.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute each SQL statement
	for i, stmt := range sql {
		_, err := tx.Execute(ctx, stmt)
		if err != nil {
			// Rollback on error
			tx.Rollback()
			return fmt.Errorf("failed to execute SQL statement %d: %w\nSQL: %s", i, err, stmt)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Ensure MigrationExecutor implements Executor interface.
var _ domain.Executor = (*MigrationExecutor)(nil)
