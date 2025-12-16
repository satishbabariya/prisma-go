// Package repository implements repository interfaces for data access.
package repository

import (
	"context"
	"fmt"
	"os"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/parser"
)

// SchemaRepositoryImpl implements the SchemaRepository interface.
type SchemaRepositoryImpl struct {
	parser domain.SchemaParser
}

// NewSchemaRepository creates a new schema repository.
func NewSchemaRepository(parser domain.SchemaParser) *SchemaRepositoryImpl {
	return &SchemaRepositoryImpl{
		parser: parser,
	}
}

// Load loads a schema from file.
func (r *SchemaRepositoryImpl) Load(ctx context.Context, path string) (*domain.Schema, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("schema file not found: %s", path)
	}

	// Use parser to load schema
	schema, err := r.parser.ParseFile(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	return schema, nil
}

// Save saves a schema to file.
func (r *SchemaRepositoryImpl) Save(ctx context.Context, path string, schema *domain.Schema) error {
	// This would use the formatter to convert schema to string
	// For now, we'll implement a basic version
	// TODO: Integrate with formatter

	return fmt.Errorf("not implemented")
}

// Validate validates a schema file exists.
func (r *SchemaRepositoryImpl) Validate(ctx context.Context, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", path)
	}
	return nil
}

// Ensure SchemaRepositoryImpl implements SchemaRepository interface.
var _ SchemaRepository = (*SchemaRepositoryImpl)(nil)

// Helper function to create a schema repository with default parser
func NewSchemaRepositoryWithDefaults() *SchemaRepositoryImpl {
	return NewSchemaRepository(parser.NewParser())
}
