// Package validator provides schema validation for Prisma schemas.
package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// Validator implements schema validation.
type Validator struct {
	schema *domain.Schema
	errors []string
}

// NewValidator creates a new schema validator.
func NewValidator() *Validator {
	return &Validator{
		errors: make([]string, 0),
	}
}

// Validate validates the entire schema.
func (v *Validator) Validate(ctx context.Context, schema *domain.Schema) error {
	v.schema = schema
	v.errors = make([]string, 0)

	// Validate datasources
	if len(schema.Datasources) == 0 {
		v.addError("schema must have at least one datasource")
	}

	for _, ds := range schema.Datasources {
		v.validateDatasource(ds)
	}

	// Validate models
	for _, model := range schema.Models {
		v.validateModel(model)
	}

	// Validate enums
	for _, enum := range schema.Enums {
		v.validateEnum(enum)
	}

	if len(v.errors) > 0 {
		return fmt.Errorf("schema validation failed:\n  - %s", strings.Join(v.errors, "\n  - "))
	}

	return nil
}

// ValidateModel validates a single model.
func (v *Validator) ValidateModel(ctx context.Context, model *domain.Model) error {
	v.errors = make([]string, 0)
	v.validateModel(*model)

	if len(v.errors) > 0 {
		return fmt.Errorf("model %s validation failed:\n  - %s", model.Name, strings.Join(v.errors, "\n  - "))
	}

	return nil
}

// ValidateField validates a single field.
func (v *Validator) ValidateField(ctx context.Context, field *domain.Field) error {
	v.errors = make([]string, 0)
	v.validateField(*field, "unknown")

	if len(v.errors) > 0 {
		return fmt.Errorf("field %s validation failed:\n  - %s", field.Name, strings.Join(v.errors, "\n  - "))
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

	// Validate referential actions
	if relation.OnDelete != "" && !v.isValidReferentialAction(relation.OnDelete) {
		return fmt.Errorf("invalid onDelete action: %s", relation.OnDelete)
	}

	if relation.OnUpdate != "" && !v.isValidReferentialAction(relation.OnUpdate) {
		return fmt.Errorf("invalid onUpdate action: %s", relation.OnUpdate)
	}

	return nil
}

// Internal validation methods

func (v *Validator) validateDatasource(ds domain.Datasource) {
	if ds.Name == "" {
		v.addError("datasource must have a name")
	}

	if ds.Provider == "" {
		v.addError(fmt.Sprintf("datasource %s must have a provider", ds.Name))
	}

	// Validate provider
	validProviders := map[string]bool{
		"postgresql":  true,
		"mysql":       true,
		"sqlite":      true,
		"sqlserver":   true,
		"mongodb":     true,
		"cockroachdb": true,
	}

	if !validProviders[ds.Provider] {
		v.addError(fmt.Sprintf("datasource %s has invalid provider: %s", ds.Name, ds.Provider))
	}

	if ds.URL == "" {
		v.addError(fmt.Sprintf("datasource %s must have a URL", ds.Name))
	}
}

func (v *Validator) validateModel(model domain.Model) {
	if model.Name == "" {
		v.addError("model must have a name")
		return
	}

	// Model name must be PascalCase
	if !v.isPascalCase(model.Name) {
		v.addError(fmt.Sprintf("model %s must be in PascalCase", model.Name))
	}

	if len(model.Fields) == 0 {
		v.addError(fmt.Sprintf("model %s must have at least one field", model.Name))
		return
	}

	// Must have at least one ID field
	hasID := false
	uniqueFields := make(map[string]bool)

	for _, field := range model.Fields {
		v.validateField(field, model.Name)

		// Check for @id attribute
		for _, attr := range field.Attributes {
			if attr.Name == "id" {
				hasID = true
			}
		}

		// Track unique field names
		if uniqueFields[field.Name] {
			v.addError(fmt.Sprintf("model %s has duplicate field: %s", model.Name, field.Name))
		}
		uniqueFields[field.Name] = true
	}

	if !hasID {
		v.addError(fmt.Sprintf("model %s must have an @id field", model.Name))
	}

	// Validate indexes
	for i, index := range model.Indexes {
		if len(index.Fields) == 0 {
			v.addError(fmt.Sprintf("model %s index %d has no fields", model.Name, i))
		}

		// Check that indexed fields exist
		for _, fieldName := range index.Fields {
			found := false
			for _, field := range model.Fields {
				if field.Name == fieldName {
					found = true
					break
				}
			}
			if !found {
				v.addError(fmt.Sprintf("model %s index references unknown field: %s", model.Name, fieldName))
			}
		}
	}
}

func (v *Validator) validateField(field domain.Field, modelName string) {
	if field.Name == "" {
		v.addError(fmt.Sprintf("field in model %s must have a name", modelName))
		return
	}

	// Field name must be camelCase
	if !v.isCamelCase(field.Name) {
		v.addError(fmt.Sprintf("field %s in model %s should be camelCase", field.Name, modelName))
	}

	// Validate field type
	if field.Type.Name == "" {
		v.addError(fmt.Sprintf("field %s in model %s must have a type", field.Name, modelName))
	}

	// List fields cannot have default values
	if field.IsList && field.DefaultValue != nil {
		v.addError(fmt.Sprintf("field %s in model %s: list fields cannot have default values", field.Name, modelName))
	}

	// Validate conflicting attributes
	hasDefault := false
	hasUpdatedAt := false

	for _, attr := range field.Attributes {
		if attr.Name == "default" {
			hasDefault = true
		}
		if attr.Name == "updatedAt" {
			hasUpdatedAt = true
		}
	}

	if hasDefault && hasUpdatedAt {
		v.addError(fmt.Sprintf("field %s in model %s cannot have both @default and @updatedAt", field.Name, modelName))
	}

	// If field is required and has no default, that's ok
	// If field is optional and required, that's a problem
	if !field.IsRequired && field.DefaultValue == nil {
		// Optional field without default is fine
	}
}

func (v *Validator) validateEnum(enum domain.Enum) {
	if enum.Name == "" {
		v.addError("enum must have a name")
		return
	}

	// Enum name must be PascalCase
	if !v.isPascalCase(enum.Name) {
		v.addError(fmt.Sprintf("enum %s must be in PascalCase", enum.Name))
	}

	if len(enum.Values) == 0 {
		v.addError(fmt.Sprintf("enum %s must have at least one value", enum.Name))
		return
	}

	// Check for duplicate values
	valueSet := make(map[string]bool)
	for _, value := range enum.Values {
		if valueSet[value] {
			v.addError(fmt.Sprintf("enum %s has duplicate value: %s", enum.Name, value))
		}
		valueSet[value] = true

		// Enum values should be UPPER_CASE
		if !v.isUpperSnakeCase(value) {
			v.addError(fmt.Sprintf("enum %s value %s should be UPPER_SNAKE_CASE", enum.Name, value))
		}
	}
}

// Helper methods

func (v *Validator) addError(msg string) {
	v.errors = append(v.errors, msg)
}

func (v *Validator) isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Must start with uppercase letter
	return s[0] >= 'A' && s[0] <= 'Z'
}

func (v *Validator) isCamelCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Must start with lowercase letter
	return s[0] >= 'a' && s[0] <= 'z'
}

func (v *Validator) isUpperSnakeCase(s string) bool {
	for _, ch := range s {
		if !(ch >= 'A' && ch <= 'Z' || ch == '_' || ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

func (v *Validator) isValidReferentialAction(action domain.ReferentialAction) bool {
	validActions := map[domain.ReferentialAction]bool{
		domain.Cascade:    true,
		domain.Restrict:   true,
		domain.NoAction:   true,
		domain.SetNull:    true,
		domain.SetDefault: true,
	}
	return validActions[action]
}

// Ensure Validator implements SchemaValidator interface.
var _ domain.SchemaValidator = (*Validator)(nil)
