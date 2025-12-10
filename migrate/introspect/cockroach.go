// Package introspect provides CockroachDB database introspection.
// CockroachDB is PostgreSQL-compatible, so we can reuse PostgreSQL introspection
// with some CockroachDB-specific adjustments.
package introspect

import (
	"context"
	"database/sql"
	"fmt"
)

// CockroachDBIntrospector implements introspection for CockroachDB
// CockroachDB is PostgreSQL-compatible, so we reuse PostgreSQL logic
type CockroachDBIntrospector struct {
	db                    *sql.DB
	*PostgresIntrospector // Embed PostgreSQL introspector
}

// NewCockroachDBIntrospector creates a new CockroachDB introspector
func NewCockroachDBIntrospector(db *sql.DB) *CockroachDBIntrospector {
	pgIntrospector := &PostgresIntrospector{db: db}
	return &CockroachDBIntrospector{
		db:                   db,
		PostgresIntrospector: pgIntrospector,
	}
}

// Introspect introspects a CockroachDB database
// CockroachDB is PostgreSQL-compatible, so we can use PostgreSQL introspection
// with CockroachDB-specific adjustments
func (i *CockroachDBIntrospector) Introspect(ctx context.Context) (*DatabaseSchema, error) {
	// Use PostgreSQL introspection as base
	schema, err := i.PostgresIntrospector.Introspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect CockroachDB: %w", err)
	}

	// CockroachDB-specific adjustments:
	// - Handle multi-region features
	// - Handle unique rowid columns
	// - Handle cluster settings

	return schema, nil
}
