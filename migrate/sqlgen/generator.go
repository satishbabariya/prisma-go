// Package sqlgen generates migration SQL from schema changes.
package sqlgen

import (
	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// MigrationGenerator is the interface for generating migration SQL
type MigrationGenerator interface {
	GenerateMigrationSQL(diffResult *diff.DiffResult, dbSchema *introspect.DatabaseSchema) (string, error)
}

