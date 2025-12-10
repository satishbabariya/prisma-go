// Package executor executes database migrations.
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/satishbabariya/prisma-go/migrate/history"
)

// MigrationExecutor executes migrations on a database
type MigrationExecutor struct {
	db       *sql.DB
	provider string
	history  *history.Manager
}

// NewMigrationExecutor creates a new migration executor
func NewMigrationExecutor(db *sql.DB, provider string) *MigrationExecutor {
	return &MigrationExecutor{
		db:       db,
		provider: provider,
		history:  history.NewManager(db, provider),
	}
}

// ExecuteMigration executes a migration SQL string
func (e *MigrationExecutor) ExecuteMigration(ctx context.Context, migrationSQL string, migrationName string) error {
	// Ensure migration table exists before starting transaction
	if err := e.EnsureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	startTime := time.Now()

	// Start transaction
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// Execute migration SQL
	_, err = tx.ExecContext(ctx, migrationSQL)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Calculate checksum
	checksum := history.CalculateChecksum(migrationSQL)
	executionTime := time.Since(startTime).Milliseconds()

	// Record migration in history
	err = e.recordMigration(ctx, tx, migrationName, checksum, executionTime)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// ExecuteMigrationStatements executes multiple SQL statements
func (e *MigrationExecutor) ExecuteMigrationStatements(ctx context.Context, statements []string, migrationName string) error {
	// Ensure migration table exists before starting transaction
	if err := e.EnsureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	startTime := time.Now()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// Execute each statement
	for i, stmt := range statements {
		if stmt == "" {
			continue
		}

		_, err = tx.ExecContext(ctx, stmt)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
	}

	// Calculate checksum from all statements
	combinedSQL := ""
	for _, stmt := range statements {
		combinedSQL += stmt + "\n"
	}
	checksum := history.CalculateChecksum(combinedSQL)
	executionTime := time.Since(startTime).Milliseconds()

	// Record migration
	err = e.recordMigration(ctx, tx, migrationName, checksum, executionTime)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// EnsureMigrationTable ensures the migration history table exists
func (e *MigrationExecutor) EnsureMigrationTable(ctx context.Context) error {
	return e.history.InitTable(ctx)
}

// GetAppliedMigrations returns list of applied migrations
func (e *MigrationExecutor) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	records, err := e.history.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	migrations := make([]Migration, len(records))
	for i, record := range records {
		// Convert string ID to int (assuming it's numeric)
		var id int
		fmt.Sscanf(record.ID, "%d", &id)
		migrations[i] = Migration{
			ID:        id,
			Name:      record.Name,
			AppliedAt: record.AppliedAt,
			Checksum:  record.Checksum,
		}
	}

	return migrations, nil
}

// GetPendingMigrations returns list of pending migrations
func (e *MigrationExecutor) GetPendingMigrations(ctx context.Context, availableMigrations []string) ([]string, error) {
	return e.history.GetPending(ctx, availableMigrations)
}

// MarkMigrationRolledBack marks a migration as rolled back
func (e *MigrationExecutor) MarkMigrationRolledBack(ctx context.Context, migrationName string) error {
	return e.history.MarkRolledBack(ctx, migrationName)
}

// recordMigration records a migration in the history table
func (e *MigrationExecutor) recordMigration(ctx context.Context, tx *sql.Tx, migrationName string, checksum string, executionTime int64) error {
	// Use a separate connection for history recording since we're in a transaction
	// We'll record it after commit, or use the same transaction
	record := &history.MigrationRecord{
		Name:          migrationName,
		AppliedAt:     time.Now(),
		Checksum:      checksum,
		ExecutionTime: executionTime,
		RolledBack:    false,
	}

	// Insert using the transaction
	insertSQL := e.getInsertSQL()
	_, err := tx.ExecContext(ctx, insertSQL,
		record.Name,
		record.AppliedAt,
		record.Checksum,
		record.ExecutionTime,
		false, // rolled_back
	)
	return err
}

// getInsertSQL returns SQL to insert a migration record
func (e *MigrationExecutor) getInsertSQL() string {
	switch e.provider {
	case "postgresql", "postgres":
		return `
			INSERT INTO _prisma_migrations (migration_name, applied_at, checksum, execution_time, rolled_back)
			VALUES ($1, $2, $3, $4, $5)
		`
	case "mysql", "sqlite":
		return `
			INSERT INTO _prisma_migrations (migration_name, applied_at, checksum, execution_time, rolled_back)
			VALUES (?, ?, ?, ?, ?)
		`
	default:
		return ""
	}
}

// Migration represents a migration record
type Migration struct {
	ID        int
	Name      string
	AppliedAt time.Time
	Checksum  string
}
