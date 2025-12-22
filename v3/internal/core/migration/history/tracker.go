// Package history provides migration history tracking.
package history

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// MigrationRecord represents a migration in the history table.
type MigrationRecord struct {
	ID              int
	MigrationName   string
	Checksum        string
	AppliedAt       time.Time
	ExecutionTimeMs int64
	Success         bool
	ErrorMessage    string
	RolledBack      bool
	RolledBackAt    *time.Time
}

// Tracker manages migration history in the database.
type Tracker struct {
	db *sql.DB
}

// NewTracker creates a new migration history tracker.
func NewTracker(db *sql.DB) *Tracker {
	return &Tracker{db: db}
}

// EnsureTable creates the _prisma_migrations table if it doesn't exist.
func (t *Tracker) EnsureTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS _prisma_migrations (
			id SERIAL PRIMARY KEY,
			migration_name VARCHAR(255) NOT NULL UNIQUE,
			checksum VARCHAR(64) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			execution_time_ms INTEGER NOT NULL,
			success BOOLEAN NOT NULL DEFAULT TRUE,
			error_message TEXT,
			rolled_back BOOLEAN NOT NULL DEFAULT FALSE,
			rolled_back_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_migration_name ON _prisma_migrations(migration_name);
		CREATE INDEX IF NOT EXISTS idx_applied_at ON _prisma_migrations(applied_at);
		CREATE INDEX IF NOT EXISTS idx_success ON _prisma_migrations(success);
	`

	_, err := t.db.Exec(query)
	return err
}

// RecordSuccess records a successful migration execution.
func (t *Tracker) RecordSuccess(name, checksum string, executionTimeMs int64) error {
	query := `
		INSERT INTO _prisma_migrations (migration_name, checksum, execution_time_ms, success)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (migration_name) DO UPDATE
		SET checksum = EXCLUDED.checksum,
		    applied_at = CURRENT_TIMESTAMP,
		    execution_time_ms = EXCLUDED.execution_time_ms,
		    success = TRUE,
		    error_message = NULL,
		    rolled_back = FALSE
	`

	_, err := t.db.Exec(query, name, checksum, executionTimeMs)
	return err
}

// RecordFailure records a failed migration attempt.
func (t *Tracker) RecordFailure(name, checksum, errorMsg string, executionTimeMs int64) error {
	query := `
		INSERT INTO _prisma_migrations (migration_name, checksum, execution_time_ms, success, error_message)
		VALUES ($1, $2, $3, FALSE, $4)
		ON CONFLICT (migration_name) DO UPDATE
		SET checksum = EXCLUDED.checksum,
		    applied_at = CURRENT_TIMESTAMP,
		    execution_time_ms = EXCLUDED.execution_time_ms,
		    success = FALSE,
		    error_message = EXCLUDED.error_message
	`

	_, err := t.db.Exec(query, name, checksum, executionTimeMs, errorMsg)
	return err
}

// GetApplied returns all successfully applied migrations.
func (t *Tracker) GetApplied() ([]MigrationRecord, error) {
	query := `
		SELECT id, migration_name, checksum, applied_at, execution_time_ms, 
		       success, COALESCE(error_message, ''), rolled_back, rolled_back_at
		FROM _prisma_migrations
		WHERE success = TRUE AND rolled_back = FALSE
		ORDER BY applied_at ASC
	`

	rows, err := t.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []MigrationRecord
	for rows.Next() {
		var m MigrationRecord
		if err := rows.Scan(&m.ID, &m.MigrationName, &m.Checksum, &m.AppliedAt,
			&m.ExecutionTimeMs, &m.Success, &m.ErrorMessage, &m.RolledBack, &m.RolledBackAt); err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	return migrations, rows.Err()
}

// GetStatus returns the status of a specific migration.
func (t *Tracker) GetStatus(name string) (*MigrationRecord, error) {
	query := `
		SELECT id, migration_name, checksum, applied_at, execution_time_ms,
		       success, COALESCE(error_message, ''), rolled_back, rolled_back_at
		FROM _prisma_migrations
		WHERE migration_name = $1
		ORDER BY applied_at DESC
		LIMIT 1
	`

	var m MigrationRecord
	err := t.db.QueryRow(query, name).Scan(
		&m.ID, &m.MigrationName, &m.Checksum, &m.AppliedAt,
		&m.ExecutionTimeMs, &m.Success, &m.ErrorMessage, &m.RolledBack, &m.RolledBackAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not applied
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ValidateChecksum verifies that a migration hasn't been modified.
func (t *Tracker) ValidateChecksum(name, expectedChecksum string) error {
	var storedChecksum string
	query := `SELECT checksum FROM _prisma_migrations WHERE migration_name = $1 AND success = TRUE`

	err := t.db.QueryRow(query, name).Scan(&storedChecksum)
	if err == sql.ErrNoRows {
		return nil // Migration not applied yet, no validation needed
	}
	if err != nil {
		return fmt.Errorf("failed to get checksum: %w", err)
	}

	if storedChecksum != expectedChecksum {
		return fmt.Errorf("migration %s has been modified (checksum mismatch: expected %s, got %s)",
			name, expectedChecksum, storedChecksum)
	}

	return nil
}

// RecordRollback marks a migration as rolled back.
func (t *Tracker) RecordRollback(name string) error {
	query := `
		UPDATE _prisma_migrations
		SET rolled_back = TRUE,
		    rolled_back_at = CURRENT_TIMESTAMP
		WHERE migration_name = $1 AND success = TRUE
	`

	result, err := t.db.Exec(query, name)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("migration %s not found or already rolled back", name)
	}

	return nil
}

// GetPending compares available migrations with applied migrations.
func (t *Tracker) GetPending(allMigrations []string) ([]string, error) {
	applied, err := t.GetApplied()
	if err != nil {
		return nil, err
	}

	// Create map of applied migrations
	appliedMap := make(map[string]bool)
	for _, m := range applied {
		appliedMap[m.MigrationName] = true
	}

	// Find pending migrations
	var pending []string
	for _, migration := range allMigrations {
		if !appliedMap[migration] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// GetAll returns all migration records (applied, failed, and rolled back).
func (t *Tracker) GetAll() ([]MigrationRecord, error) {
	query := `
		SELECT id, migration_name, checksum, applied_at, execution_time_ms,
		       success, COALESCE(error_message, ''), rolled_back, rolled_back_at
		FROM _prisma_migrations
		ORDER BY applied_at DESC
	`

	rows, err := t.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []MigrationRecord
	for rows.Next() {
		var m MigrationRecord
		if err := rows.Scan(&m.ID, &m.MigrationName, &m.Checksum, &m.AppliedAt,
			&m.ExecutionTimeMs, &m.Success, &m.ErrorMessage, &m.RolledBack, &m.RolledBackAt); err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	return migrations, rows.Err()
}

// CalculateChecksum computes SHA256 checksum of migration content.
func CalculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
