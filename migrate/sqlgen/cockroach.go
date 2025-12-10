// Package sqlgen generates migration SQL for CockroachDB.
// CockroachDB is PostgreSQL-compatible, so we can reuse PostgreSQL SQL generation
// with CockroachDB-specific adjustments.
package sqlgen

import (
	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// CockroachDBMigrationGenerator generates CockroachDB migration SQL
// CockroachDB is PostgreSQL-compatible, so we reuse PostgreSQL generator
type CockroachDBMigrationGenerator struct {
	*PostgresMigrationGenerator // Embed PostgreSQL generator
}

// NewCockroachDBMigrationGenerator creates a new CockroachDB migration generator
func NewCockroachDBMigrationGenerator() *CockroachDBMigrationGenerator {
	return &CockroachDBMigrationGenerator{
		PostgresMigrationGenerator: NewPostgresMigrationGenerator(),
	}
}

// GenerateMigrationSQL generates SQL for a diff result
// CockroachDB is PostgreSQL-compatible, so we can use PostgreSQL SQL
// with CockroachDB-specific adjustments
func (g *CockroachDBMigrationGenerator) GenerateMigrationSQL(diffResult *diff.DiffResult, dbSchema *introspect.DatabaseSchema) (string, error) {
	// Use PostgreSQL SQL generation as base
	sql, err := g.PostgresMigrationGenerator.GenerateMigrationSQL(diffResult, dbSchema)
	if err != nil {
		return "", err
	}

	// CockroachDB-specific adjustments:
	// - Handle multi-region syntax
	// - Handle unique rowid
	// - Handle cluster settings

	return sql, nil
}

// GenerateRollbackSQL generates rollback SQL for a diff result
func (g *CockroachDBMigrationGenerator) GenerateRollbackSQL(diffResult *diff.DiffResult, dbSchema *introspect.DatabaseSchema) (string, error) {
	return g.PostgresMigrationGenerator.GenerateRollbackSQL(diffResult, dbSchema)
}
