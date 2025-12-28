// Package domain defines interfaces for database introspection.
package domain

import (
	"context"
	"database/sql"
)

// Introspector defines the interface for database introspection.
type Introspector interface {
	// IntrospectDatabase introspects the entire database schema.
	IntrospectDatabase(ctx context.Context, db *sql.DB, schemas []string) (*IntrospectedDatabase, error)

	// IntrospectTable introspects a single table.
	IntrospectTable(ctx context.Context, db *sql.DB, tableName string) (*IntrospectedTable, error)

	// GetDatabaseVersion returns the database version.
	GetDatabaseVersion(ctx context.Context, db *sql.DB) (string, error)

	// GetSchemas returns all available schemas.
	GetSchemas(ctx context.Context, db *sql.DB) ([]string, error)
}

// TypeMapper maps database types to Prisma types.
type TypeMapper interface {
	// MapToPrismaType converts a database type to Prisma type.
	MapToPrismaType(dbType string, isNullable bool) string

	// MapToGoType converts a database type to Go type.
	MapToGoType(dbType string, isNullable bool) string
}

// RelationInferrer infers relations from foreign keys.
type RelationInferrer interface {
	// InferRelations analyzes foreign keys and infers Prisma relations.
	InferRelations(db *IntrospectedDatabase) ([]InferredRelation, error)
}

// InferredRelation represents a relation inferred from database structure.
type InferredRelation struct {
	FromTable    string
	ToTable      string
	FromFields   []string
	ToFields     []string
	RelationType RelationType // OneToOne, OneToMany, ManyToOne, ManyToMany
	RelationName string       // Generated relation name
	OnDelete     ReferentialAction
	OnUpdate     ReferentialAction
}

// RelationType represents the type of relation.
type RelationType string

const (
	// OneToOne represents a one-to-one relation.
	OneToOne RelationType = "OneToOne"
	// OneToMany represents a one-to-many relation.
	OneToMany RelationType = "OneToMany"
	// ManyToOne represents a many-to-one relation.
	ManyToOne RelationType = "ManyToOne"
	// ManyToMany represents a many-to-many relation.
	ManyToMany RelationType = "ManyToMany"
)
