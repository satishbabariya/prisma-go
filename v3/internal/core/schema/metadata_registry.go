// Package schema provides a metadata registry for schema information.
package schema

import (
	"fmt"
	"sync"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// MetadataRegistry stores queryable schema metadata for use by compilers and generators.
// It provides fast lookup of models, fields, and relations.
type MetadataRegistry struct {
	mu        sync.RWMutex
	models    map[string]*domain.Model
	relations map[string][]domain.Relation // key: model name
	enums     map[string]*domain.Enum
}

// NewMetadataRegistry creates a new metadata registry.
func NewMetadataRegistry() *MetadataRegistry {
	return &MetadataRegistry{
		models:    make(map[string]*domain.Model),
		relations: make(map[string][]domain.Relation),
		enums:     make(map[string]*domain.Enum),
	}
}

// LoadFromSchema loads metadata from a parsed schema.
func (r *MetadataRegistry) LoadFromSchema(schema *domain.Schema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing data
	r.models = make(map[string]*domain.Model)
	r.relations = make(map[string][]domain.Relation)
	r.enums = make(map[string]*domain.Enum)

	// Index models
	for i := range schema.Models {
		model := &schema.Models[i]
		r.models[model.Name] = model
	}

	// Index enums
	for i := range schema.Enums {
		enum := &schema.Enums[i]
		r.enums[enum.Name] = enum
	}

	// Index relations by model
	// Relations in Prisma are typically defined via @relation attributes on fields
	// We'll build reverse lookups from relation fields
	for _, model := range schema.Models {
		for _, field := range model.Fields {
			// Check if this field is a relation field
			if r.isRelationType(field.Type.Name) {
				// Create relation metadata
				rel := r.buildRelationFromField(model, field)
				if rel != nil {
					r.relations[model.Name] = append(r.relations[model.Name], *rel)
				}
			}
		}
	}

	return nil
}

// GetModel retrieves a model by name.
func (r *MetadataRegistry) GetModel(name string) (*domain.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, exists := r.models[name]
	if !exists {
		return nil, fmt.Errorf("model %s not found", name)
	}
	return model, nil
}

// GetRelation retrieves relation metadata between two models.
func (r *MetadataRegistry) GetRelation(fromModel, relationName string) (*domain.Relation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	relations, exists := r.relations[fromModel]
	if !exists {
		return nil, fmt.Errorf("no relations found for model %s", fromModel)
	}

	for i := range relations {
		if relations[i].Name == relationName {
			return &relations[i], nil
		}
	}

	return nil, fmt.Errorf("relation %s not found for model %s", relationName, fromModel)
}

// GetAllRelations returns all relations for a model.
func (r *MetadataRegistry) GetAllRelations(modelName string) ([]domain.Relation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	relations, exists := r.relations[modelName]
	if !exists {
		return nil, nil // No relations is not an error
	}

	// Return a copy to prevent external modification
	result := make([]domain.Relation, len(relations))
	copy(result, relations)
	return result, nil
}

// GetField retrieves a field from a model.
func (r *MetadataRegistry) GetField(modelName, fieldName string) (*domain.Field, error) {
	model, err := r.GetModel(modelName)
	if err != nil {
		return nil, err
	}

	for i := range model.Fields {
		if model.Fields[i].Name == fieldName {
			return &model.Fields[i], nil
		}
	}

	return nil, fmt.Errorf("field %s not found in model %s", fieldName, modelName)
}

// IsEnum checks if a type name is an enum.
func (r *MetadataRegistry) IsEnum(typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.enums[typeName]
	return exists
}

// isRelationType checks if a type is a relation (not a scalar or enum).
func (r *MetadataRegistry) isRelationType(typeName string) bool {
	// Check if it's a scalar type
	scalars := map[string]bool{
		"String": true, "Boolean": true, "Int": true, "BigInt": true,
		"Float": true, "Decimal": true, "DateTime": true,
		"Json": true, "Bytes": true,
	}
	if scalars[typeName] {
		return false
	}

	// Check if it's an enum
	if r.IsEnum(typeName) {
		return false
	}

	// Otherwise, assume it's a model reference (relation)
	return true
}

// buildRelationFromField constructs relation metadata from a field.
// This is a simplified implementation - real Prisma schema has @relation attributes.
func (r *MetadataRegistry) buildRelationFromField(model domain.Model, field domain.Field) *domain.Relation {
	// Look for @relation attribute to get detailed info
	var relationAttr *domain.Attribute
	for i := range field.Attributes {
		if field.Attributes[i].Name == "relation" {
			relationAttr = &field.Attributes[i]
			break
		}
	}

	rel := &domain.Relation{
		Name:      field.Name,
		FromModel: model.Name,
		ToModel:   field.Type.Name,
	}

	// If we have @relation attribute, extract fields and references
	if relationAttr != nil {
		// Parse @relation(fields: [...], references: [...])
		rel.FromFields = r.extractRelationFields(relationAttr, "fields")
		rel.ToFields = r.extractRelationFields(relationAttr, "references")
	}

	// Determine relation type based on field  arity
	if field.IsList {
		// One-to-many (this side is "many")
		rel.RelationType = domain.OneToMany
	} else {
		// Could be one-to-one or many-to-one
		// Default to many-to-one for now
		rel.RelationType = domain.ManyToOne
	}

	return rel
}

// extractRelationFields extracts field names from @relation attribute.
func (r *MetadataRegistry) extractRelationFields(attr *domain.Attribute, argName string) []string {
	// This is a placeholder - real implementation would parse the attribute arguments
	// For now, return empty
	return []string{}
}

// GetTableName returns the database table name for a model (handles @@map).
func (r *MetadataRegistry) GetTableName(modelName string) (string, error) {
	model, err := r.GetModel(modelName)
	if err != nil {
		return "", err
	}

	// Check for @@map attribute
	for i := range model.Attributes {
		if model.Attributes[i].Name == "map" {
			// Extract the mapped name from attribute args
			// Simplified - real implementation would parse args properly
			if len(model.Attributes[i].Arguments) > 0 {
				return fmt.Sprint(model.Attributes[i].Arguments[0]), nil
			}
		}
	}

	// Default to model name
	return model.Name, nil
}

// GetColumnName returns the database column name for a field (handles @map).
func (r *MetadataRegistry) GetColumnName(modelName, fieldName string) (string, error) {
	field, err := r.GetField(modelName, fieldName)
	if err != nil {
		return "", err
	}

	// Check for @map attribute
	for i := range field.Attributes {
		if field.Attributes[i].Name == "map" {
			// Extract the mapped name
			if len(field.Attributes[i].Arguments) > 0 {
				return fmt.Sprint(field.Attributes[i].Arguments[0]), nil
			}
		}
	}

	// Default to field name
	return field.Name, nil
}
