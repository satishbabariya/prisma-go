// Package pslcore provides name validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// Names tracks names used in the schema for validation.
type Names struct {
	// Track constraint names to detect duplicates
	PrimaryKeyNames  map[string]bool
	UniqueNames      map[string]bool
	IndexNames       map[string]bool
	DefaultNames     map[string]bool
	RelationNames    map[string]bool
	FieldClientNames map[string]bool
	ModelClientNames map[string]bool
	EnumClientNames  map[string]bool
}

// NewNames creates a new Names tracker.
func NewNames() *Names {
	return &Names{
		PrimaryKeyNames:  make(map[string]bool),
		UniqueNames:      make(map[string]bool),
		IndexNames:       make(map[string]bool),
		DefaultNames:     make(map[string]bool),
		RelationNames:    make(map[string]bool),
		FieldClientNames: make(map[string]bool),
		ModelClientNames: make(map[string]bool),
		EnumClientNames:  make(map[string]bool),
	}
}

// validateUniquePrimaryKeyName validates that primary key names are unique.
func validateUniquePrimaryKeyName(model *database.ModelWalker, names *Names, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	pk := model.PrimaryKey()
	if pk == nil {
		return
	}

	if name := pk.Name(); name != nil {
		if names.PrimaryKeyNames[*name] {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Primary key name '%s' is already used.", *name),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
		}
		names.PrimaryKeyNames[*name] = true
	}

	if mappedName := pk.MappedName(); mappedName != nil {
		key := fmt.Sprintf("pk:%s", *mappedName)
		if names.PrimaryKeyNames[key] {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Primary key mapped name '%s' is already used.", *mappedName),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
		}
		names.PrimaryKeyNames[key] = true
	}
}

// validateUniqueIndexName validates that index names are unique.
func validateUniqueIndexName(index *database.IndexWalker, model *database.ModelWalker, names *Names, ctx *ValidationContext) {
	if name := index.Name(); name != nil {
		if names.IndexNames[*name] {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Index name '%s' is already used.", *name),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
		}
		names.IndexNames[*name] = true
	}
}

// validateUniqueDefaultConstraintName validates that default constraint names are unique.
func validateUniqueDefaultConstraintName(field *database.ScalarFieldWalker, names *Names, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	defaultValue := field.DefaultValue()
	if defaultValue == nil {
		return
	}

	if mappedName := defaultValue.MappedName(); mappedName != nil {
		key := fmt.Sprintf("default:%s", *mappedName)
		if names.DefaultNames[key] {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Default constraint name '%s' is already used.", *mappedName),
				diagnostics.NewSpan(0, 0, field.Model().FileID()),
			))
		}
		names.DefaultNames[key] = true
	}
}

// validateFieldClientName validates that field client names don't clash.
func validateFieldClientName(field *database.ScalarFieldWalker, model *database.ModelWalker, names *Names, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	clientName := field.DatabaseName()
	key := fmt.Sprintf("%s.%s", model.Name(), clientName)

	if names.FieldClientNames[key] {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Field '%s' in model '%s' has duplicate client name '%s'.", field.Name(), model.Name(), clientName),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
	names.FieldClientNames[key] = true
}

// validateRelationName validates that relation names are unique.
func validateRelationName(relation *database.RelationWalker, names *Names, ctx *ValidationContext) {
	if relation.IsIgnored() {
		return
	}

	relName := relation.RelationName()
	if names.RelationNames[relName] {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Relation name '%s' is already used.", relName),
			diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
		))
	}
	names.RelationNames[relName] = true
}
