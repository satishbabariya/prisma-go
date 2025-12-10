// Package executor provides validation utilities.
package executor

import (
	"fmt"
	"reflect"
)

// validateDestination validates that the destination is appropriate for scanning
func validateDestination(dest interface{}) error {
	if dest == nil {
		return fmt.Errorf("destination cannot be nil")
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer, got %v", destValue.Kind())
	}

	if destValue.IsNil() {
		return fmt.Errorf("destination pointer cannot be nil")
	}

	elemType := destValue.Elem().Type()
	if destValue.Elem().Kind() == reflect.Slice {
		elemType = elemType.Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
	} else {
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("destination element type must be a struct, got %v", elemType.Kind())
	}

	return nil
}

// validateRelations validates relation metadata
func validateRelations(relations map[string]RelationMetadata) error {
	for name, relMeta := range relations {
		if relMeta.RelatedTable == "" {
			return fmt.Errorf("relation %s has empty RelatedTable", name)
		}
		if relMeta.ForeignKey == "" && relMeta.IsList {
			// One-to-many relations need foreign key
			return fmt.Errorf("relation %s is one-to-many but has no ForeignKey", name)
		}
		if relMeta.LocalKey == "" {
			return fmt.Errorf("relation %s has empty LocalKey", name)
		}
	}
	return nil
}

