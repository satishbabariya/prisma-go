// Package service implements application services (use cases).
package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	migrationdomain "github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/repository"
)

// MigrationService orchestrates migration operations.
type MigrationService struct {
	schemaRepo    repository.SchemaRepository
	migrationRepo repository.MigrationRepository
	historyRepo   repository.HistoryRepository
	introspector  migrationdomain.Introspector
	differ        migrationdomain.Differ
	planner       migrationdomain.Planner
	executor      migrationdomain.Executor
}

// NewMigrationService creates a new migration service.
func NewMigrationService(
	schemaRepo repository.SchemaRepository,
	migrationRepo repository.MigrationRepository,
	historyRepo repository.HistoryRepository,
	introspector migrationdomain.Introspector,
	differ migrationdomain.Differ,
	planner migrationdomain.Planner,
	executor migrationdomain.Executor,
) *MigrationService {
	return &MigrationService{
		schemaRepo:    schemaRepo,
		migrationRepo: migrationRepo,
		historyRepo:   historyRepo,
		introspector:  introspector,
		differ:        differ,
		planner:       planner,
		executor:      executor,
	}
}

// CreateMigrationInput represents input for creating a migration.
type CreateMigrationInput struct {
	Name       string
	SchemaPath string
}

// CreateMigration creates a new migration based on schema changes.
func (s *MigrationService) CreateMigration(ctx context.Context, input CreateMigrationInput) (*migrationdomain.Migration, error) {
	// 1. Load schema
	_, err := s.schemaRepo.Load(ctx, input.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	// 2. Introspect current database state (if introspector available)
	var dbState *migrationdomain.DatabaseState
	if s.introspector != nil {
		dbState, err = s.introspector.IntrospectDatabase(ctx)
		if err != nil {
			// If introspection fails, assume empty database
			dbState = &migrationdomain.DatabaseState{Tables: []migrationdomain.Table{}}
		}
	} else {
		// No introspector, assume empty database
		dbState = &migrationdomain.DatabaseState{Tables: []migrationdomain.Table{}}
	}

	// 3. Convert schema to desired database state
	// This is a simplified version - in reality, you'd convert schema.Models to Tables
	desiredState := &migrationdomain.DatabaseState{
		Tables: []migrationdomain.Table{},
	}

	// 4. Compare states and generate changes
	var changes []migrationdomain.Change
	if s.differ != nil {
		changes, err = s.differ.Compare(ctx, dbState, desiredState)
		if err != nil {
			return nil, fmt.Errorf("failed to compare database states: %w", err)
		}
	}

	// 5. Create migration plan
	var plan *migrationdomain.MigrationPlan
	if s.planner != nil && len(changes) > 0 {
		plan, err = s.planner.CreatePlan(ctx, changes)
		if err != nil {
			return nil, fmt.Errorf("failed to create migration plan: %w", err)
		}
	} else {
		plan = &migrationdomain.MigrationPlan{
			Changes: changes,
			SQL:     []string{},
		}
	}

	// 6. Generate migration ID and checksum
	now := time.Now()
	migrationID := fmt.Sprintf("%d_%s", now.Unix(), input.Name)
	checksum := generateChecksum(plan.SQL)

	// 7. Create migration object
	migration := &migrationdomain.Migration{
		ID:        migrationID,
		Name:      input.Name,
		CreatedAt: now,
		Changes:   changes,
		SQL:       plan.SQL,
		Checksum:  checksum,
		Status:    migrationdomain.Pending,
	}

	// 8. Save migration
	if err := s.migrationRepo.Save(ctx, migration); err != nil {
		return nil, fmt.Errorf("failed to save migration: %w", err)
	}

	return migration, nil
}

// ApplyMigration applies a specific migration.
func (s *MigrationService) ApplyMigration(ctx context.Context, migrationID string) error {
	// 1. Load migration
	migration, err := s.migrationRepo.FindByID(ctx, migrationID)
	if err != nil {
		return fmt.Errorf("migration not found: %w", err)
	}

	// 2. Check if already applied
	if migration.Status == migrationdomain.Applied {
		return fmt.Errorf("migration already applied: %s", migrationID)
	}

	// 3. Execute migration
	if s.executor != nil {
		if err := s.executor.Execute(ctx, migration); err != nil {
			migration.Status = migrationdomain.Failed
			s.migrationRepo.Save(ctx, migration)
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	// 4. Mark as applied
	if err := s.migrationRepo.MarkAsApplied(ctx, migrationID); err != nil {
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	// 5. Record in history
	if s.historyRepo != nil {
		if err := s.historyRepo.Record(ctx, migration); err != nil {
			return fmt.Errorf("failed to record migration in history: %w", err)
		}
	}

	return nil
}

// RollbackMigration rolls back a migration.
func (s *MigrationService) RollbackMigration(ctx context.Context, migrationID string) error {
	migration, err := s.migrationRepo.FindByID(ctx, migrationID)
	if err != nil {
		return fmt.Errorf("migration not found: %w", err)
	}

	if migration.Status != migrationdomain.Applied {
		return fmt.Errorf("cannot rollback migration that is not applied: %s", migrationID)
	}

	if s.executor != nil {
		if err := s.executor.Rollback(ctx, migration); err != nil {
			return fmt.Errorf("failed to rollback migration: %w", err)
		}
	}

	migration.Status = migrationdomain.RolledBack
	migration.AppliedAt = nil

	return s.migrationRepo.Save(ctx, migration)
}

// GetMigrationStatus gets the current migration status.
func (s *MigrationService) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	pending, err := s.migrationRepo.FindPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending migrations: %w", err)
	}

	applied, err := s.migrationRepo.FindApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find applied migrations: %w", err)
	}

	return &MigrationStatus{
		Pending: pending,
		Applied: applied,
		Total:   len(pending) + len(applied),
	}, nil
}

// MigrationStatus represents the current migration status.
type MigrationStatus struct {
	Pending []*migrationdomain.Migration
	Applied []*migrationdomain.Migration
	Total   int
}

// generateChecksum generates an MD5 checksum for SQL statements.
func generateChecksum(sql []string) string {
	combined := ""
	for _, statement := range sql {
		combined += statement
	}
	hash := md5.Sum([]byte(combined))
	return fmt.Sprintf("%x", hash)
}
