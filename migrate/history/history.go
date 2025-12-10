// Package history manages migration history tracking.
package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// MigrationRecord represents a migration in the history
type MigrationRecord struct {
	ID            string
	Name          string
	AppliedAt     time.Time
	ExecutionTime int64 // milliseconds
	Checksum      string
	RolledBack    bool
}

// Manager manages migration history
type Manager struct {
	db       *sql.DB
	provider string
}

// NewManager creates a new migration history manager
func NewManager(db *sql.DB, provider string) *Manager {
	return &Manager{
		db:       db,
		provider: provider,
	}
}

// InitTable creates the migrations history table
func (m *Manager) InitTable(ctx context.Context) error {
	createTableSQL := m.getMigrationTableSQL()
	_, err := m.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}
	return nil
}

// Record records a migration execution
func (m *Manager) Record(ctx context.Context, record *MigrationRecord) error {
	insertSQL := m.getInsertSQL()
	_, err := m.db.ExecContext(ctx, insertSQL,
		record.Name,
		record.AppliedAt,
		record.Checksum,
		record.ExecutionTime,
		record.RolledBack,
	)
	return err
}

// GetAll returns all migration records
func (m *Manager) GetAll(ctx context.Context) ([]MigrationRecord, error) {
	query := m.getSelectAllSQL()
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		var rolledBackInt int
		err := rows.Scan(
			&record.ID,
			&record.Name,
			&record.AppliedAt,
			&record.Checksum,
			&record.ExecutionTime,
			&rolledBackInt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		record.RolledBack = (rolledBackInt == 1)
		records = append(records, record)
	}

	return records, rows.Err()
}

// GetPending returns migrations that haven't been applied
func (m *Manager) GetPending(ctx context.Context, availableMigrations []string) ([]string, error) {
	applied, err := m.GetAppliedMigrationNames(ctx)
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]bool)
	for _, name := range applied {
		appliedMap[name] = true
	}

	var pending []string
	for _, name := range availableMigrations {
		if !appliedMap[name] {
			pending = append(pending, name)
		}
	}

	return pending, nil
}

// GetAppliedMigrationNames returns names of all applied migrations
func (m *Manager) GetAppliedMigrationNames(ctx context.Context) ([]string, error) {
	query := m.getSelectNamesSQL()
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan migration name: %w", err)
		}
		names = append(names, name)
	}

	return names, rows.Err()
}

// MarkRolledBack marks a migration as rolled back
func (m *Manager) MarkRolledBack(ctx context.Context, migrationName string) error {
	updateSQL := m.getUpdateRolledBackSQL()
	_, err := m.db.ExecContext(ctx, updateSQL, migrationName)
	return err
}

// CalculateChecksum calculates a checksum for migration SQL
func CalculateChecksum(migrationSQL string) string {
	hash := sha256.Sum256([]byte(migrationSQL))
	return hex.EncodeToString(hash[:])
}

// getMigrationTableSQL returns SQL to create migration history table
func (m *Manager) getMigrationTableSQL() string {
	switch m.provider {
	case "postgresql", "postgres":
		return `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id SERIAL PRIMARY KEY,
				migration_name VARCHAR(255) NOT NULL UNIQUE,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				checksum VARCHAR(64) NOT NULL,
				execution_time INTEGER,
				rolled_back BOOLEAN DEFAULT FALSE
			)
		`
	case "mysql":
		return `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id INT AUTO_INCREMENT PRIMARY KEY,
				migration_name VARCHAR(255) NOT NULL UNIQUE,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				checksum VARCHAR(64) NOT NULL,
				execution_time INT,
				rolled_back TINYINT(1) DEFAULT 0
			)
		`
	case "sqlite":
		return `
			CREATE TABLE IF NOT EXISTS _prisma_migrations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				migration_name TEXT NOT NULL UNIQUE,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				checksum TEXT NOT NULL,
				execution_time INTEGER,
				rolled_back INTEGER DEFAULT 0
			)
		`
	default:
		return ""
	}
}

// getInsertSQL returns SQL to insert a migration record
func (m *Manager) getInsertSQL() string {
	switch m.provider {
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

// getSelectAllSQL returns SQL to select all migrations
func (m *Manager) getSelectAllSQL() string {
	return `
		SELECT id, migration_name, applied_at, checksum, execution_time, rolled_back
		FROM _prisma_migrations
		ORDER BY applied_at ASC
	`
}

// getSelectNamesSQL returns SQL to select migration names
func (m *Manager) getSelectNamesSQL() string {
	return `
		SELECT migration_name
		FROM _prisma_migrations
		WHERE rolled_back = 0
		ORDER BY applied_at ASC
	`
}

// getUpdateRolledBackSQL returns SQL to mark migration as rolled back
func (m *Manager) getUpdateRolledBackSQL() string {
	switch m.provider {
	case "postgresql", "postgres":
		return `
			UPDATE _prisma_migrations
			SET rolled_back = TRUE
			WHERE migration_name = $1
		`
	case "mysql":
		return `
			UPDATE _prisma_migrations
			SET rolled_back = 1
			WHERE migration_name = ?
		`
	case "sqlite":
		return `
			UPDATE _prisma_migrations
			SET rolled_back = 1
			WHERE migration_name = ?
		`
	default:
		return ""
	}
}
