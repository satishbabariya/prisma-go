// Package executor applies migration plans to databases.
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/satishbabariya/prisma-go/internal/debug"
	"github.com/satishbabariya/prisma-go/migrate/history"
	"github.com/satishbabariya/prisma-go/migrate/planner"
)

// Executor executes migration plans
type Executor struct {
	db       *sql.DB
	provider string
	history  *history.Manager
}

// NewExecutor creates a new migration executor
func NewExecutor(db *sql.DB, provider string) *Executor {
	debug.Debug("Creating new migration executor", "provider", provider)
	return &Executor{
		db:       db,
		provider: provider,
		history:  history.NewManager(db, provider),
	}
}

// Execute applies a migration plan to the database
func (e *Executor) Execute(ctx context.Context, plan *planner.MigrationPlan) error {
	debug.Debug("Executing migration plan", "plan", plan)
	// TODO: Implement migration execution
	// 1. Begin transaction
	// 2. Create migrations table if not exists
	// 3. Execute each step
	// 4. Record migration in history
	// 5. Commit transaction
	debug.Debug("Migration execution not yet implemented")
	return nil
}

// Rollback rolls back a migration
func (e *Executor) Rollback(ctx context.Context, migrationID string) error {
	debug.Debug("Starting migration rollback", "migrationID", migrationID)

	// Ensure migration table exists
	debug.Debug("Initializing migration table")
	if err := e.history.InitTable(ctx); err != nil {
		debug.Error("Failed to initialize migration table", "error", err)
		return fmt.Errorf("failed to ensure migration table exists: %w", err)
	}

	// Get all migration records to find the one to rollback
	debug.Debug("Fetching all migration records")
	records, err := e.history.GetAll(ctx)
	if err != nil {
		debug.Error("Failed to get migration records", "error", err)
		return fmt.Errorf("failed to get migration records: %w", err)
	}
	debug.Debug("Retrieved migration records", "count", len(records))

	// Find the migration record by ID or name
	var targetRecord *history.MigrationRecord
	for i := range records {
		if records[i].ID == migrationID || records[i].Name == migrationID {
			if records[i].RolledBack {
				debug.Error("Migration already rolled back", "migrationID", migrationID)
				return fmt.Errorf("migration '%s' has already been rolled back", migrationID)
			}
			targetRecord = &records[i]
			debug.Debug("Found target migration record", "name", targetRecord.Name, "id", targetRecord.ID)
			break
		}
	}

	if targetRecord == nil {
		debug.Error("Migration not found", "migrationID", migrationID)
		return fmt.Errorf("migration '%s' not found", migrationID)
	}

	// Read rollback SQL from disk
	// Migration files are stored in migrations/{migrationName}/rollback.sql
	migrationDir := filepath.Join("migrations", targetRecord.Name)
	rollbackPath := filepath.Join(migrationDir, "rollback.sql")
	debug.Debug("Reading rollback SQL file", "path", rollbackPath)

	rollbackSQL, err := os.ReadFile(rollbackPath)
	if err != nil {
		debug.Error("Failed to read rollback SQL file", "path", rollbackPath, "error", err)
		return fmt.Errorf("failed to read rollback SQL file '%s': %w", rollbackPath, err)
	}

	// If rollback SQL is empty or just comments, return an error
	rollbackSQLStr := strings.TrimSpace(string(rollbackSQL))
	if rollbackSQLStr == "" || strings.HasPrefix(rollbackSQLStr, "--") {
		debug.Error("Rollback SQL is empty or contains only comments", "migrationID", migrationID)
		return fmt.Errorf("rollback SQL is empty or contains only comments for migration '%s'", migrationID)
	}
	debug.Debug("Rollback SQL loaded", "length", len(rollbackSQLStr))

	// Start transaction
	debug.Debug("Starting database transaction")
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		debug.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// Execute rollback SQL
	// Split by semicolons and execute each statement
	debug.Debug("Splitting rollback SQL into statements")
	statements := splitSQLStatements(rollbackSQLStr)
	debug.Debug("Split rollback SQL", "statementCount", len(statements))

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			debug.Debug("Skipping empty or comment statement", "index", i)
			continue
		}

		debug.Debug("Executing rollback statement", "index", i+1, "statement", stmt)
		_, err = tx.ExecContext(ctx, stmt)
		if err != nil {
			debug.Error("Failed to execute rollback statement", "index", i+1, "statement", stmt, "error", err)
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute rollback statement %d: %w", i+1, err)
		}
		debug.Debug("Rollback statement executed successfully", "index", i+1)
	}

	// Mark migration as rolled back
	debug.Debug("Marking migration as rolled back", "name", targetRecord.Name)
	err = e.markRolledBackInTx(ctx, tx, targetRecord.Name)
	if err != nil {
		debug.Error("Failed to mark migration as rolled back", "error", err)
		_ = tx.Rollback()
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	// Commit transaction
	debug.Debug("Committing transaction")
	if err := tx.Commit(); err != nil {
		debug.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	debug.Info("Migration rollback completed successfully", "migrationID", migrationID)
	return nil
}

// markRolledBackInTx marks a migration as rolled back within a transaction
func (e *Executor) markRolledBackInTx(ctx context.Context, tx *sql.Tx, migrationName string) error {
	var updateSQL string
	switch e.provider {
	case "postgresql", "postgres":
		updateSQL = `UPDATE _prisma_migrations SET rolled_back = TRUE WHERE migration_name = $1`
		debug.Debug("Updating migration record (PostgreSQL)", "sql", updateSQL, "migrationName", migrationName)
		_, err := tx.ExecContext(ctx, updateSQL, migrationName)
		return err
	case "mysql", "sqlite":
		updateSQL = `UPDATE _prisma_migrations SET rolled_back = 1 WHERE migration_name = ?`
		debug.Debug("Updating migration record (MySQL/SQLite)", "sql", updateSQL, "migrationName", migrationName)
		_, err := tx.ExecContext(ctx, updateSQL, migrationName)
		return err
	default:
		debug.Error("Unsupported provider", "provider", e.provider)
		return fmt.Errorf("unsupported provider: %s", e.provider)
	}
}

// splitSQLStatements splits SQL string by semicolons, handling quoted strings
func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false

	for i, char := range sql {
		switch char {
		case '\'':
			if !inDoubleQuote && !inBacktick {
				inSingleQuote = !inSingleQuote
			}
			current.WriteRune(char)
		case '"':
			if !inSingleQuote && !inBacktick {
				inDoubleQuote = !inDoubleQuote
			}
			current.WriteRune(char)
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
			current.WriteRune(char)
		case ';':
			if !inSingleQuote && !inDoubleQuote && !inBacktick {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}

		// Handle last statement if no trailing semicolon
		if i == len(sql)-1 {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
		}
	}

	return statements
}
