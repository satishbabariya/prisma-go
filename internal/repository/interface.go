// Package repository defines repository interfaces for data access.
package repository

import (
	"context"

	"github.com/satishbabariya/prisma-go/v3/internal/config"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
	schemadomain "github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// MigrationRepository defines the interface for migration data access.
type MigrationRepository interface {
	// Save saves a migration.
	Save(ctx context.Context, migration *domain.Migration) error

	// FindAll retrieves all migrations.
	FindAll(ctx context.Context) ([]*domain.Migration, error)

	// FindByID retrieves a migration by ID.
	FindByID(ctx context.Context, id string) (*domain.Migration, error)

	// FindPending retrieves pending migrations.
	FindPending(ctx context.Context) ([]*domain.Migration, error)

	// FindApplied retrieves applied migrations.
	FindApplied(ctx context.Context) ([]*domain.Migration, error)

	// MarkAsApplied marks a migration as applied.
	MarkAsApplied(ctx context.Context, id string) error

	// Delete deletes a migration.
	Delete(ctx context.Context, id string) error
}

// SchemaRepository defines the interface for schema data access.
type SchemaRepository interface {
	// Load loads a schema from file.
	Load(ctx context.Context, path string) (*schemadomain.Schema, error)

	// Save saves a schema to file.
	Save(ctx context.Context, path string, schema *schemadomain.Schema) error

	// Validate validates a schema file exists.
	Validate(ctx context.Context, path string) error
}

// HistoryRepository defines the interface for migration history data access.
type HistoryRepository interface {
	// Record records a migration in history.
	Record(ctx context.Context, migration *domain.Migration) error

	// GetHistory retrieves migration history.
	GetHistory(ctx context.Context) ([]*domain.Migration, error)

	// GetLastApplied retrieves the last applied migration.
	GetLastApplied(ctx context.Context) (*domain.Migration, error)
}

// ConfigRepository defines the interface for configuration data access.
type ConfigRepository interface {
	// Load loads configuration.
	Load(ctx context.Context) (*config.Config, error)

	// Save saves configuration.
	Save(ctx context.Context, cfg *config.Config) error
}
