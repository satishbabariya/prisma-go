// Package service implements application services (use cases).
package service

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/repository"
)

// SchemaService orchestrates schema operations.
type SchemaService struct {
	schemaRepo repository.SchemaRepository
	validator  domain.SchemaValidator
	formatter  domain.SchemaFormatter
}

// NewSchemaService creates a new schema service.
func NewSchemaService(
	schemaRepo repository.SchemaRepository,
	validator domain.SchemaValidator,
	formatter domain.SchemaFormatter,
) *SchemaService {
	return &SchemaService{
		schemaRepo: schemaRepo,
		validator:  validator,
		formatter:  formatter,
	}
}

// ParseAndValidate parses and validates a schema file.
func (s *SchemaService) ParseAndValidate(ctx context.Context, schemaPath string) (*domain.Schema, error) {
	// Load schema
	schema, err := s.schemaRepo.Load(ctx, schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	// Validate schema
	if err := s.validator.Validate(ctx, schema); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return schema, nil
}

// FormatSchema formats a schema file.
func (s *SchemaService) FormatSchema(ctx context.Context, schemaPath string) error {
	// Validate file exists
	if err := s.schemaRepo.Validate(ctx, schemaPath); err != nil {
		return err
	}

	// Format the file
	if err := s.formatter.FormatFile(ctx, schemaPath); err != nil {
		return fmt.Errorf("failed to format schema: %w", err)
	}

	return nil
}

// ValidateSchema validates a schema file without loading it fully.
func (s *SchemaService) ValidateSchema(ctx context.Context, schemaPath string) error {
	schema, err := s.schemaRepo.Load(ctx, schemaPath)
	if err != nil {
		return err
	}

	return s.validator.Validate(ctx, schema)
}
