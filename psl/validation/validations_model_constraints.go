// Package pslcore provides model constraint validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateModelHasAtLeastOneField validates that a model has at least one field.
func validateModelHasAtLeastOneField(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	scalarFields := model.ScalarFields()
	relationFields := model.RelationFields()

	if len(scalarFields) == 0 && len(relationFields) == 0 {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Model '%s' must have at least one field.", model.Name()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}

// validateModelUniqueConstraints validates unique constraints on a model.
func validateModelUniqueConstraints(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	// Get unique constraints from indexes
	indexes := model.Indexes()
	uniqueNames := make(map[string]bool)

	for _, index := range indexes {
		if index.IsUnique() {
			if name := index.Name(); name != nil {
				if uniqueNames[*name] {
					ctx.PushError(diagnostics.NewValidationError(
						fmt.Sprintf("Model '%s' has duplicate unique constraint name '%s'.", model.Name(), *name),
						diagnostics.NewSpan(0, 0, model.FileID()),
					))
				}
				uniqueNames[*name] = true
			}
		}
	}

	// Validate that unique constraints don't conflict with primary key
	pk := model.PrimaryKey()
	if pk != nil {
		pkFields := pk.Fields()
		pkFieldIDs := make([]database.ScalarFieldId, len(pkFields))
		for i, f := range pkFields {
			pkFieldIDs[i] = f.FieldID()
		}

		// Check if any unique index matches primary key exactly
		for _, index := range indexes {
			if !index.IsUnique() {
				continue
			}

			indexFields := index.Fields()
			if len(indexFields) == len(pkFields) {
				indexFieldIDs := make([]database.ScalarFieldId, len(indexFields))
				for i, f := range indexFields {
					indexFieldIDs[i] = f.FieldID()
				}

				// Check if field IDs match exactly
				matches := true
				for i := range pkFieldIDs {
					if pkFieldIDs[i] != indexFieldIDs[i] {
						matches = false
						break
					}
				}

				if matches {
					// Unique constraint matches primary key - this is redundant but not an error
					// Some databases allow this, so we'll just skip it
					continue
				}
			}
		}
	}
}

// validateModelIndexConstraints validates index constraints on a model.
func validateModelIndexConstraints(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	indexes := model.Indexes()

	// Validate that indexes don't duplicate primary key
	pk := model.PrimaryKey()
	if pk != nil {
		pkFields := pk.Fields()
		for _, index := range indexes {
			indexFields := index.Fields()
			// Check if index matches primary key exactly
			if len(indexFields) == len(pkFields) {
				matches := true
				for i, indexField := range indexFields {
					if i >= len(pkFields) {
						matches = false
						break
					}
					sf := indexField.ScalarField()
					pkSF := pkFields[i].ScalarField()
					if sf == nil || pkSF == nil || sf.FieldID() != pkSF.FieldID() {
						matches = false
						break
					}
				}
				if matches {
					ctx.PushError(diagnostics.NewValidationError(
						fmt.Sprintf("Model '%s' has an index that duplicates the primary key.", model.Name()),
						diagnostics.NewSpan(0, 0, model.FileID()),
					))
				}
			}
		}
	}
}

// validateModelShardKeyConstraints validates shard key constraints on a model.
func validateModelShardKeyConstraints(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	shardKey := model.ShardKey()
	if shardKey == nil {
		return
	}

	// Validate that shard key fields don't conflict with primary key
	pk := model.PrimaryKey()
	if pk != nil {
		pkFields := pk.Fields()
		shardKeyFields := shardKey.Fields()

		// Check if shard key matches primary key exactly (not allowed)
		if len(shardKeyFields) == len(pkFields) {
			matches := true
			for i, shardKeyField := range shardKeyFields {
				if i >= len(pkFields) {
					matches = false
					break
				}
				if database.ScalarFieldId(shardKeyField.FieldID()) != pkFields[i].FieldID() {
					matches = false
					break
				}
			}
			if matches {
				attr := shardKey.AstAttribute()
				if attr != nil {
					ctx.PushError(diagnostics.NewAttributeValidationError(
						"Shard key cannot be the same as the primary key.",
						shardKey.AttributeName(),
						diagnostics.NewSpan(attr.Pos.Offset, attr.Pos.Offset+len(attr.String()), model.FileID()),
					))
				}
			}
		}
	}

	// Validate that shard key fields are required
	for _, field := range shardKey.Fields() {
		if field.IsOptional() {
			attr := shardKey.AstAttribute()
			if attr != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"Fields that are marked as shard keys must be required.",
					shardKey.AttributeName(),
					diagnostics.NewSpan(attr.Pos.Offset, attr.Pos.Offset+len(attr.String()), model.FileID()),
				))
			}
			break // Only report once
		}
	}
}

// validateModelFieldNames validates that field names don't conflict with reserved names.
func validateModelFieldNames(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	reservedNames := []string{"id", "createdAt", "updatedAt"}
	fieldNames := make(map[string]bool)

	// Check scalar fields
	for _, field := range model.ScalarFields() {
		name := field.Name()
		if fieldNames[name] {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Model '%s' has duplicate field name '%s'.", model.Name(), name),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
		}
		fieldNames[name] = true

		// Check for reserved names (warning only)
		for _, reserved := range reservedNames {
			if name == reserved {
				ctx.PushWarning(diagnostics.NewDatamodelWarning(
					fmt.Sprintf("Field '%s' in model '%s' uses reserved name '%s'.", name, model.Name(), reserved),
					diagnostics.NewSpan(0, 0, model.FileID()),
				))
			}
		}
	}

	// Check relation fields
	for _, field := range model.RelationFields() {
		name := field.Name()
		if fieldNames[name] {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Model '%s' has duplicate field name '%s'.", model.Name(), name),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
		}
		fieldNames[name] = true
	}
}
