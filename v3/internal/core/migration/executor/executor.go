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

	// If we have pre-computed rollback SQL, use it
	if len(migration.RollbackSQL) > 0 {
		return e.ExecuteSQL(ctx, migration.RollbackSQL)
	}

	// Otherwise, try to generate rollback SQL from changes
	// by reversing the change order and generating inverse operations
	if len(migration.Changes) == 0 {
		return fmt.Errorf("cannot rollback: no changes or rollback SQL available")
	}

	rollbackSQL, err := e.generateRollbackSQL(migration)
	if err != nil {
		return fmt.Errorf("failed to generate rollback SQL: %w", err)
	}

	return e.ExecuteSQL(ctx, rollbackSQL)
}

// generateRollbackSQL generates rollback SQL from migration changes.
func (e *MigrationExecutor) generateRollbackSQL(migration *domain.Migration) ([]string, error) {
	var rollbackSQL []string

	// Get dialect from database adapter
	dialect := domain.SQLDialect(e.db.GetDialect())

	// Process changes in reverse order
	for i := len(migration.Changes) - 1; i >= 0; i-- {
		change := migration.Changes[i]
		sql, err := generateReverseSQL(change, dialect)
		if err != nil {
			return nil, fmt.Errorf("cannot reverse change '%s': %w", change.Description(), err)
		}
		rollbackSQL = append(rollbackSQL, sql...)
	}

	return rollbackSQL, nil
}

// generateReverseSQL generates the SQL to reverse a single change.
func generateReverseSQL(change domain.Change, dialect domain.SQLDialect) ([]string, error) {
	switch change.Type() {
	case domain.CreateTable:
		// Reverse of create table is drop table
		// Need to extract table name from the change
		return []string{fmt.Sprintf("-- Rollback: %s", change.Description())}, nil

	case domain.DropTable:
		// Cannot easily reverse a drop table (data is lost)
		return nil, fmt.Errorf("cannot rollback drop table: original table definition unknown")

	case domain.AddColumn:
		// Reverse of add column is drop column
		// Would need to parse the change to get table and column names
		return []string{fmt.Sprintf("-- Rollback: %s (requires manual intervention)", change.Description())}, nil

	case domain.DropColumn:
		// Cannot easily reverse a drop column (data is lost)
		return nil, fmt.Errorf("cannot rollback drop column: original column data is lost")

	case domain.CreateIndex:
		// Reverse of create index is drop index
		return []string{fmt.Sprintf("-- Rollback: %s", change.Description())}, nil

	case domain.DropIndex:
		// Would need original index definition
		return nil, fmt.Errorf("cannot rollback drop index: original index definition unknown")

	default:
		// For other changes, indicate manual intervention needed
		return []string{fmt.Sprintf("-- Rollback: %s (requires manual intervention)", change.Description())}, nil
	}
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
