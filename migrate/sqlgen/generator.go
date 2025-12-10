// Package sqlgen generates migration SQL from schema changes.
package sqlgen

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// MigrationGenerator is the interface for generating migration SQL
type MigrationGenerator interface {
	GenerateMigrationSQL(diffResult *diff.DiffResult, dbSchema *introspect.DatabaseSchema) (string, error)
	GenerateRollbackSQL(diffResult *diff.DiffResult, dbSchema *introspect.DatabaseSchema) (string, error)
}

// NewMigrationGenerator creates a new migration generator for the given provider
func NewMigrationGenerator(provider string) (MigrationGenerator, error) {
	switch provider {
	case "postgresql", "postgres":
		return NewPostgresMigrationGenerator(), nil
	case "mysql":
		return NewMySQLMigrationGenerator(), nil
	case "sqlite":
		return NewSQLiteMigrationGenerator(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
