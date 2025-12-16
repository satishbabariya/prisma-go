// Package repository implements repository interfaces for data access.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// MigrationRepositoryImpl implements the MigrationRepository interface.
type MigrationRepositoryImpl struct {
	migrationsDir string
}

// NewMigrationRepository creates a new migration repository.
func NewMigrationRepository(migrationsDir string) *MigrationRepositoryImpl {
	return &MigrationRepositoryImpl{
		migrationsDir: migrationsDir,
	}
}

// Save saves a migration to the migrations directory.
func (r *MigrationRepositoryImpl) Save(ctx context.Context, migration *domain.Migration) error {
	// Ensure migrations directory exists
	if err := os.MkdirAll(r.migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Create migration file
	filename := fmt.Sprintf("%s_%s.json", migration.CreatedAt.Format("20060102150405"), migration.Name)
	path := filepath.Join(r.migrationsDir, filename)

	// Marshal migration to JSON
	data, err := json.MarshalIndent(migration, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	return nil
}

// FindAll retrieves all migrations.
func (r *MigrationRepositoryImpl) FindAll(ctx context.Context) ([]*domain.Migration, error) {
	// Ensure directory exists
	if _, err := os.Stat(r.migrationsDir); os.IsNotExist(err) {
		return []*domain.Migration{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(r.migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []*domain.Migration
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(r.migrationsDir, entry.Name())
		migration, err := r.loadMigration(path)
		if err != nil {
			// Skip invalid migrations
			continue
		}

		migrations = append(migrations, migration)
	}

	// Sort by created time
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].CreatedAt.Before(migrations[j].CreatedAt)
	})

	return migrations, nil
}

// FindByID retrieves a migration by ID.
func (r *MigrationRepositoryImpl) FindByID(ctx context.Context, id string) (*domain.Migration, error) {
	migrations, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, migration := range migrations {
		if migration.ID == id {
			return migration, nil
		}
	}

	return nil, fmt.Errorf("migration not found: %s", id)
}

// FindPending retrieves pending migrations.
func (r *MigrationRepositoryImpl) FindPending(ctx context.Context) ([]*domain.Migration, error) {
	migrations, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var pending []*domain.Migration
	for _, migration := range migrations {
		if migration.Status == domain.Pending {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// FindApplied retrieves applied migrations.
func (r *MigrationRepositoryImpl) FindApplied(ctx context.Context) ([]*domain.Migration, error) {
	migrations, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var applied []*domain.Migration
	for _, migration := range migrations {
		if migration.Status == domain.Applied {
			applied = append(applied, migration)
		}
	}

	return applied, nil
}

// MarkAsApplied marks a migration as applied.
func (r *MigrationRepositoryImpl) MarkAsApplied(ctx context.Context, id string) error {
	migration, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	migration.AppliedAt = &now
	migration.Status = domain.Applied

	return r.Save(ctx, migration)
}

// Delete deletes a migration.
func (r *MigrationRepositoryImpl) Delete(ctx context.Context, id string) error {
	migrations, err := r.FindAll(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.ID == id {
			// Find the file and delete it
			entries, err := os.ReadDir(r.migrationsDir)
			if err != nil {
				return fmt.Errorf("failed to read migrations directory: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				path := filepath.Join(r.migrationsDir, entry.Name())
				m, err := r.loadMigration(path)
				if err != nil {
					continue
				}

				if m.ID == id {
					return os.Remove(path)
				}
			}
		}
	}

	return fmt.Errorf("migration not found: %s", id)
}

// loadMigration loads a migration from a file.
func (r *MigrationRepositoryImpl) loadMigration(path string) (*domain.Migration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	var migration domain.Migration
	if err := json.Unmarshal(data, &migration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal migration: %w", err)
	}

	return &migration, nil
}

// Ensure MigrationRepositoryImpl implements MigrationRepository interface.
var _ MigrationRepository = (*MigrationRepositoryImpl)(nil)
