// Package validator implements schema validation.
package validator

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// Validator implements the SchemaValidator interface.
type Validator struct{}

// NewValidator creates a new schema validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates the entire schema.
func (v *Validator) Validate(ctx context.Context, schema *domain.Schema) error {
	// Validate datasources
	if len(schema.Datasources) == 0 {
		return fmt.Errorf("schema must have at least one datasource")
	}

	// Validate models
	for _, model := range schema.Models {
		if err := v.ValidateModel(ctx, &model); err != nil {
			return fmt.Errorf("model %s: %w", model.Name, err)
		}
	}

	// Validate enums
	for _, enum := range schema.Enums {
		if len(enum.Values) == 0 {
			return fmt.Errorf("enum %s must have at least one value", enum.Name)
		}
	}

	return nil
}

// ValidateModel validates a single model.
func (v *Validator) ValidateModel(ctx context.Context, model *domain.Model) error {
	if model.Name == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	if len(model.Fields) == 0 {
		return fmt.Errorf("model must have at least one field")
	}

	// Check for at least one unique identifier
	hasID := false
	for _, field := range model.Fields {
		if err := v.ValidateField(ctx, &field); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		// Check if field has @id attribute
		for _, attr := range field.Attributes {
			if attr.Name == "id" {
				hasID = true
			}
		}
	}

	if !hasID {
		return fmt.Errorf("model must have at least one @id field")
	}

	return nil
}

// ValidateField validates a single field.
func (v *Validator) ValidateField(ctx context.Context, field *domain.Field) error {
	if field.Name == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	if field.Type.Name == "" {
		return fmt.Errorf("field type cannot be empty")
	}

	return nil
}

// ValidateRelation validates a relation.
func (v *Validator) ValidateRelation(ctx context.Context, relation *domain.Relation) error {
	if relation.FromModel == "" || relation.ToModel == "" {
		return fmt.Errorf("relation must specify both from and to models")
	}

	if relation.RelationType == "" {
		return fmt.Errorf("relation must specify a relation type")
	}

	return nil
}

// Ensure Validator implements SchemaValidator interface.
var _ domain.SchemaValidator = (*Validator)(nil)
