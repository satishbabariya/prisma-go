// Package executor provides relation loading functionality.
package executor

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// loadRelations loads related data for included relations
func (e *Executor) loadRelations(ctx context.Context, table string, include map[string]bool, relations map[string]RelationMetadata, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	// Handle slice of structs
	if destValue.Elem().Kind() == reflect.Slice {
		sliceValue := destValue.Elem()
		for i := 0; i < sliceValue.Len(); i++ {
			elem := sliceValue.Index(i)
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if err := e.loadRelationsForStruct(ctx, table, include, relations, elem); err != nil {
				return err
			}
		}
		return nil
	}

	// Handle single struct
	structValue := destValue.Elem()
	return e.loadRelationsForStruct(ctx, table, include, relations, structValue)
}

// loadRelationsForStruct loads relations for a single struct
func (e *Executor) loadRelationsForStruct(ctx context.Context, table string, include map[string]bool, relations map[string]RelationMetadata, structValue reflect.Value) error {
	if structValue.Kind() != reflect.Struct {
		return nil
	}

	for relationName := range include {
		relMeta, ok := relations[relationName]
		if !ok || relMeta.ForeignKey == "" {
			continue
		}

		// Find the relation field in the struct
		fieldValue := structValue.FieldByName(toPascalCase(relationName))
		if !fieldValue.IsValid() {
			continue
		}

		// Get the ID of the current record
		idField := structValue.FieldByName("Id")
		if !idField.IsValid() {
			idField = structValue.FieldByName("ID")
		}
		if !idField.IsValid() {
			continue
		}

		idValue := idField.Interface()

		// Load relation based on type
		if relMeta.IsList {
			// One-to-many: query related table where foreign_key = id
			if err := e.loadOneToMany(ctx, relMeta, idValue, fieldValue); err != nil {
				return fmt.Errorf("failed to load %s: %w", relationName, err)
			}
		} else {
			// Many-to-one: query related table where id = foreign_key
			if err := e.loadManyToOne(ctx, relMeta, structValue, fieldValue); err != nil {
				return fmt.Errorf("failed to load %s: %w", relationName, err)
			}
		}
	}

	return nil
}

// loadOneToMany loads a one-to-many relation
func (e *Executor) loadOneToMany(ctx context.Context, relMeta RelationMetadata, parentID interface{}, fieldValue reflect.Value) error {
	// Build WHERE clause: foreign_key = parent_id
	where := &sqlgen.WhereClause{
		Conditions: []sqlgen.Condition{
			{
				Field:    relMeta.ForeignKey,
				Operator: "=",
				Value:    parentID,
			},
		},
		Operator: "AND",
	}

	// Query related records
	elementType := fieldValue.Type().Elem()
	if elementType.Kind() == reflect.Ptr {
		elementType = elementType.Elem()
	}

	// Create a slice to hold results
	sliceType := reflect.SliceOf(elementType)
	results := reflect.New(sliceType).Interface()

	if err := e.FindMany(ctx, relMeta.RelatedTable, nil, where, nil, nil, nil, nil, results); err != nil {
		return err
	}

	// Set the field value
	resultsValue := reflect.ValueOf(results).Elem()
	fieldValue.Set(resultsValue)

	return nil
}

// loadManyToOne loads a many-to-one relation
func (e *Executor) loadManyToOne(ctx context.Context, relMeta RelationMetadata, structValue reflect.Value, fieldValue reflect.Value) error {
	// Get foreign key value from struct
	fkFieldName := toPascalCase(relMeta.ForeignKey)
	fkField := structValue.FieldByName(fkFieldName)
	if !fkField.IsValid() {
		// Try snake_case version
		fkField = structValue.FieldByName(toPascalCase(fromSnakeCase(relMeta.ForeignKey)))
	}
	if !fkField.IsValid() {
		return fmt.Errorf("foreign key field %s not found", relMeta.ForeignKey)
	}

	fkValue := fkField.Interface()
	if fkValue == nil {
		return nil // Foreign key is NULL
	}

	// Build WHERE clause: id = foreign_key_value
	where := &sqlgen.WhereClause{
		Conditions: []sqlgen.Condition{
			{
				Field:    relMeta.LocalKey,
				Operator: "=",
				Value:    fkValue,
			},
		},
		Operator: "AND",
	}

	// Query related record
	elementType := fieldValue.Type()
	if elementType.Kind() == reflect.Ptr {
		elementType = elementType.Elem()
	}

	result := reflect.New(elementType).Interface()

	if err := e.FindFirst(ctx, relMeta.RelatedTable, nil, where, nil, nil, result); err != nil {
		return err
	}

	// Set the field value
	resultValue := reflect.ValueOf(result).Elem()
	if fieldValue.Type().Kind() == reflect.Ptr {
		fieldValue.Set(resultValue.Addr())
	} else {
		fieldValue.Set(resultValue)
	}

	return nil
}

// Helper functions
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Split(s, "_")
	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

func fromSnakeCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if i > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

