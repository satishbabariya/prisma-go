// Package repository implements repository interfaces for data access.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// HistoryRepositoryImpl implements the HistoryRepository interface using a database.
type HistoryRepositoryImpl struct {
	db database.Adapter
}

// NewHistoryRepository creates a new history repository.
func NewHistoryRepository(db database.Adapter) *HistoryRepositoryImpl {
	return &HistoryRepositoryImpl{
		db: db,
	}
}

// Record records a migration in the history table.
func (r *HistoryRepositoryImpl) Record(ctx context.Context, migration *domain.Migration) error {
	// Ensure the migrations table exists
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	query := `
		INSERT INTO _prisma_migrations (id, name, applied_at, checksum)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.Execute(ctx, query,
		migration.ID,
		migration.Name,
		migration.AppliedAt,
		migration.Checksum,
	)

	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// GetHistory retrieves the migration history from the database.
func (r *HistoryRepositoryImpl) GetHistory(ctx context.Context) ([]*domain.Migration, error) {
	// Ensure the migrations table exists
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	query := `
		SELECT id, name, applied_at, checksum
		FROM _prisma_migrations
		ORDER BY applied_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration history: %w", err)
	}
	defer rows.Close()

	var migrations []*domain.Migration
	for rows.Next() {
		var migration domain.Migration
		var appliedAt sql.NullTime

		err := rows.Scan(
			&migration.ID,
			&migration.Name,
			&appliedAt,
			&migration.Checksum,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}

		if appliedAt.Valid {
			migration.AppliedAt = &appliedAt.Time
			migration.Status = domain.Applied
		}

		migrations = append(migrations, &migration)
	}

	return migrations, nil
}

// GetLastApplied retrieves the last applied migration.
func (r *HistoryRepositoryImpl) GetLastApplied(ctx context.Context) (*domain.Migration, error) {
	// Ensure the migrations table exists
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	query := `
		SELECT id, name, applied_at, checksum
		FROM _prisma_migrations
		ORDER BY applied_at DESC
		LIMIT 1
	`

	row := r.db.QueryRow(ctx, query)

	var migration domain.Migration
	var appliedAt sql.NullTime

	err := row.Scan(
		&migration.ID,
		&migration.Name,
		&appliedAt,
		&migration.Checksum,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to scan migration row: %w", err)
	}

	if appliedAt.Valid {
		migration.AppliedAt = &appliedAt.Time
		migration.Status = domain.Applied
	}

	return &migration, nil
}

// ensureMigrationsTable creates the migrations table if it doesn't exist.
func (r *HistoryRepositoryImpl) ensureMigrationsTable(ctx context.Context) error {
	if r.db == nil {
		// If no database adapter, skip table creation
		return nil
	}

	query := `
		CREATE TABLE IF NOT EXISTS _prisma_migrations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP,
			checksum TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := r.db.Execute(ctx, query)
	return err
}

// Ensure HistoryRepositoryImpl implements HistoryRepository interface.
var _ HistoryRepository = (*HistoryRepositoryImpl)(nil)
