// Package service implements application services (use cases).
package service

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
	schemadomain "github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/repository"
)

// IntrospectService handles database introspection operations.
type IntrospectService struct {
	schemaRepo   repository.SchemaRepository
	introspector domain.Introspector
}

// NewIntrospectService creates a new introspect service.
func NewIntrospectService(
	schemaRepo repository.SchemaRepository,
	introspector domain.Introspector,
) *IntrospectService {
	return &IntrospectService{
		schemaRepo:   schemaRepo,
		introspector: introspector,
	}
}

// IntrospectInput represents input for database introspection.
type IntrospectInput struct {
	SchemaPath string
	OutputPath string
}

// Introspect introspects the database and generates a schema.
func (s *IntrospectService) Introspect(ctx context.Context, input IntrospectInput) error {
	if s.introspector == nil {
		return fmt.Errorf("introspector not initialized")
	}

	// 1. Introspect database
	dbState, err := s.introspector.IntrospectDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect database: %w", err)
	}

	// 2. Convert database state to Prisma schema
	// This is a placeholder - actual implementation would convert Tables to Schema
	schema := s.convertDatabaseStateToSchema(dbState)

	// 3. Save schema to file
	if err := s.schemaRepo.Save(ctx, input.OutputPath, schema); err != nil {
		return fmt.Errorf("failed to save schema: %w", err)
	}

	return nil
}

// IntrospectTable introspects a specific table.
func (s *IntrospectService) IntrospectTable(ctx context.Context, tableName string) (*domain.Table, error) {
	if s.introspector == nil {
		return nil, fmt.Errorf("introspector not initialized")
	}

	table, err := s.introspector.IntrospectTable(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect table %s: %w", tableName, err)
	}

	return table, nil
}

// ListTables lists all tables in the database.
func (s *IntrospectService) ListTables(ctx context.Context) ([]string, error) {
	if s.introspector == nil {
		return nil, fmt.Errorf("introspector not initialized")
	}

	tables, err := s.introspector.ListTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	return tables, nil
}

// convertDatabaseStateToSchema converts a DatabaseState to a Prisma Schema.
// This is a simplified placeholder implementation.
func (s *IntrospectService) convertDatabaseStateToSchema(dbState *domain.DatabaseState) *schemadomain.Schema {
	// TODO: Implement actual conversion from database tables to Prisma models
	// For now, return an empty schema
	return &schemadomain.Schema{
		Datasources: []schemadomain.Datasource{},
		Generators:  []schemadomain.Generator{},
		Models:      []schemadomain.Model{},
		Enums:       []schemadomain.Enum{},
	}
}
