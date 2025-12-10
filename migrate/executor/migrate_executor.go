// Package executor executes database migrations.
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// MigrationExecutor executes migrations on a database
type MigrationExecutor struct {
	db       *sql.DB
	provider string
}

// NewMigrationExecutor creates a new migration executor
func NewMigrationExecutor(db *sql.DB, provider string) *MigrationExecutor {
	return &MigrationExecutor{
		db:       db,
		provider: provider,
	}
}

// ExecuteMigration executes a migration SQL string
func (e *MigrationExecutor) ExecuteMigration(ctx context.Context, migrationSQL string, migrationName string) error {
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

	// Record migration in history
	err = e.recordMigration(ctx, tx, migrationName)
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

	// Record migration
	err = e.recordMigration(ctx, tx, migrationName)
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
	createTableSQL := e.getMigrationTableSQL()
	
	_, err := e.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns list of applied migrations
func (e *MigrationExecutor) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `
		SELECT id, migration_name, applied_at, checksum
		FROM _prisma_migrations
		ORDER BY applied_at ASC
	`

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var m Migration
		err := rows.Scan(&m.ID, &m.Name, &m.AppliedAt, &m.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		migrations = append(migrations, m)
	}

	return migrations, rows.Err()
}

// recordMigration records a migration in the history table
func (e *MigrationExecutor) recordMigration(ctx context.Context, tx *sql.Tx, migrationName string) error {
	insertSQL := `
		INSERT INTO _prisma_migrations (migration_name, applied_at, checksum)
		VALUES ($1, $2, $3)
	`

	if e.provider == "mysql" {
		insertSQL = `
			INSERT INTO _prisma_migrations (migration_name, applied_at, checksum)
			VALUES (?, ?, ?)
		`
	} else if e.provider == "sqlite" {
		insertSQL = `
			INSERT INTO _prisma_migrations (migration_name, applied_at, checksum)
			VALUES (?, ?, ?)
		`
	}

	checksum := fmt.Sprintf("%x", time.Now().UnixNano()) // Simple checksum
	_, err := tx.ExecContext(ctx, insertSQL, migrationName, time.Now(), checksum)
	return err
}

// getMigrationTableSQL returns SQL to create migration history table
func (e *MigrationExecutor) getMigrationTableSQL() string {
	switch e.provider {
	case "postgresql", "postgres":
		return `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id SERIAL PRIMARY KEY,
				migration_name VARCHAR(255) NOT NULL UNIQUE,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				checksum VARCHAR(64) NOT NULL
			)
		`
	case "mysql":
		return `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id INT AUTO_INCREMENT PRIMARY KEY,
				migration_name VARCHAR(255) NOT NULL UNIQUE,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				checksum VARCHAR(64) NOT NULL
			)
		`
	case "sqlite":
		return `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				migration_name TEXT NOT NULL UNIQUE,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				checksum TEXT NOT NULL
			)
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

