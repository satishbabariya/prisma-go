// Package history manages migration history tracking.
package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// MigrationRecord represents a migration in the history
type MigrationRecord struct {
	ID            string
	Name          string
	AppliedAt     time.Time
	ExecutionTime int64 // milliseconds
	Checksum      string
	RolledBack    bool
	// SchemaSnapshot stores the database schema state before this migration was applied
	// Serialized as JSON string
	SchemaSnapshot string
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
		record.SchemaSnapshot,
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
		var schemaSnapshot sql.NullString
		err := rows.Scan(
			&record.ID,
			&record.Name,
			&record.AppliedAt,
			&record.Checksum,
			&record.ExecutionTime,
			&rolledBackInt,
			&schemaSnapshot,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		record.RolledBack = (rolledBackInt == 1)
		if schemaSnapshot.Valid {
			record.SchemaSnapshot = schemaSnapshot.String
		}
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

// GetSchemaSnapshot retrieves the schema snapshot for a specific migration
func (m *Manager) GetSchemaSnapshot(ctx context.Context, migrationName string) (*introspect.DatabaseSchema, error) {
	query := m.getSelectSchemaSnapshotSQL()
	var schemaSnapshot sql.NullString
	err := m.db.QueryRowContext(ctx, query, migrationName).Scan(&schemaSnapshot)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("migration '%s' not found", migrationName)
		}
		return nil, fmt.Errorf("failed to query schema snapshot: %w", err)
	}

	if !schemaSnapshot.Valid || schemaSnapshot.String == "" {
		return nil, nil // No schema snapshot stored
	}

	return DeserializeSchema(schemaSnapshot.String)
}

// RecordWithSchema records a migration execution with schema snapshot
func (m *Manager) RecordWithSchema(ctx context.Context, record *MigrationRecord, schema *introspect.DatabaseSchema) error {
	if schema != nil {
		schemaJSON, err := SerializeSchema(schema)
		if err != nil {
			return fmt.Errorf("failed to serialize schema: %w", err)
		}
		record.SchemaSnapshot = schemaJSON
	}
	return m.Record(ctx, record)
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
				rolled_back BOOLEAN DEFAULT FALSE,
				schema_snapshot TEXT
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
				rolled_back TINYINT(1) DEFAULT 0,
				schema_snapshot TEXT
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
				rolled_back INTEGER DEFAULT 0,
				schema_snapshot TEXT
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
			INSERT INTO _prisma_migrations (migration_name, applied_at, checksum, execution_time, rolled_back, schema_snapshot)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
	case "mysql", "sqlite":
		return `
			INSERT INTO _prisma_migrations (migration_name, applied_at, checksum, execution_time, rolled_back, schema_snapshot)
			VALUES (?, ?, ?, ?, ?, ?)
		`
	default:
		return ""
	}
}

// getSelectAllSQL returns SQL to select all migrations
func (m *Manager) getSelectAllSQL() string {
	return `
		SELECT id, migration_name, applied_at, checksum, execution_time, rolled_back, schema_snapshot
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

// getSelectSchemaSnapshotSQL returns SQL to select schema snapshot for a migration
func (m *Manager) getSelectSchemaSnapshotSQL() string {
	return `
		SELECT schema_snapshot
		FROM _prisma_migrations
		WHERE migration_name = ?
	`
}
