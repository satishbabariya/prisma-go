// Package pslcore provides advanced relation validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateRelationReferencingScalarFieldTypes validates that referencing and referenced fields have compatible types.
func validateRelationReferencingScalarFieldTypes(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	if forwardField == nil {
		return
	}

	referencingFields := forwardField.ReferencingFields()
	referencedFields := forwardField.ReferencedFields()

	if len(referencingFields) == 0 || len(referencedFields) == 0 {
		return
	}

	if len(referencingFields) != len(referencedFields) {
		return // Length mismatch is handled by another validation
	}

	astField := forwardField.AstField()
	if astField == nil {
		return
	}

	// Check that each pair of fields has compatible types
	for i := 0; i < len(referencingFields); i++ {
		referencingType := referencingFields[i].ScalarFieldType()
		referencedType := referencedFields[i].ScalarFieldType()

		if !fieldTypesMatchForRelations(referencingType, referencedType) {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf(
					"The type of the field `%s` (`%s`) does not match the type of the referenced field `%s` (`%s`) in the related model.",
					referencingFields[i].Name(),
					getScalarTypeDisplayNameForRelations(&referencingType),
					referencedFields[i].Name(),
					getScalarTypeDisplayNameForRelations(&referencedType),
				),
				astField.Span(),
			))
		}
	}
}

// fieldTypesMatchForRelations checks if two scalar field types are compatible for relations.
// This is a helper function for relation validations.
func fieldTypesMatchForRelations(referencing, referenced database.ScalarFieldType) bool {
	// Check if both are composite types
	if referencing.CompositeTypeID != nil && referenced.CompositeTypeID != nil {
		return *referencing.CompositeTypeID == *referenced.CompositeTypeID
	}

	// Check if both are enum types
	if referencing.EnumID != nil && referenced.EnumID != nil {
		return *referencing.EnumID == *referenced.EnumID
	}

	// Check if both are built-in scalar types
	if referencing.BuiltInScalar != nil && referenced.BuiltInScalar != nil {
		return *referencing.BuiltInScalar == *referenced.BuiltInScalar
	}

	// Check if both are unsupported types
	if referencing.Unsupported != nil && referenced.Unsupported != nil {
		// Compare unsupported type names
		return referencing.Unsupported.Name == referenced.Unsupported.Name
	}

	return false
}

// getScalarTypeDisplayNameForRelations returns a display name for a ScalarFieldType.
func getScalarTypeDisplayNameForRelations(scalarType *database.ScalarFieldType) string {
	if scalarType == nil {
		return "Unknown"
	}
	if scalarType.BuiltInScalar != nil {
		return string(*scalarType.BuiltInScalar)
	}
	if scalarType.EnumID != nil {
		return "Enum"
	}
	if scalarType.CompositeTypeID != nil {
		return "CompositeType"
	}
	if scalarType.Unsupported != nil {
		// Unsupported.Name is a StringId, need to get the actual string
		// For now, return a placeholder
		return "Unsupported"
	}
	return "Unknown"
}

// validateRelationHasUniqueConstraintName validates that relation constraint names are unique.
func validateRelationHasUniqueConstraintName(relation *database.InlineRelationWalker, names *Names, ctx *ValidationContext) {
	// TODO: Get constraint name when ConstraintName() method is available on InlineRelationWalker
	// The validation should:
	// 1. Get the constraint name from the relation
	// 2. Check if it conflicts with other constraint names in the namespace
	// 3. Report errors for conflicts

	_ = relation
	_ = names
}

// validateRelationCycles detects cycles in relations.
func validateRelationCycles(relation *database.CompleteInlineRelationWalker, ctx *ValidationContext) {
	// TODO: Implement cycle detection when CompleteInlineRelationWalker is available
	// The validation should:
	// 1. Traverse relations starting from the given relation
	// 2. Track visited relations
	// 3. Detect cycles in referential actions
	// 4. Report errors for cycles

	_ = relation
}

// validateRelationMultipleCascadingPaths detects multiple cascading paths.
func validateRelationMultipleCascadingPaths(relation *database.CompleteInlineRelationWalker, ctx *ValidationContext) {
	// TODO: Implement multiple cascading paths detection when CompleteInlineRelationWalker is available
	// The validation should:
	// 1. Check if relation triggers modifications (onDelete/onUpdate)
	// 2. Find all paths from the referencing model to other models
	// 3. Detect if there are multiple paths to the same model
	// 4. Report warnings/errors for multiple cascading paths

	_ = relation
}

// validateRelationFieldArityAdvanced validates field arity in relations with more detail.
func validateRelationFieldArityAdvanced(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	if forwardField == nil {
		return
	}

	astField := forwardField.AstField()
	if astField == nil {
		return
	}

	// If the relation field is required, all referencing fields must also be required
	if !astField.FieldType.IsOptional() {
		referencingFields := forwardField.ReferencingFields()
		if len(referencingFields) == 0 {
			return
		}

		// Check if any referencing field is optional
		optionalFields := []string{}
		for _, field := range referencingFields {
			fieldAst := field.AstField()
			if fieldAst != nil && fieldAst.FieldType.IsOptional() {
				optionalFields = append(optionalFields, field.Name())
			}
		}

		if len(optionalFields) > 0 {
			fieldNames := make([]string, len(referencingFields))
			for i, f := range referencingFields {
				fieldNames[i] = f.Name()
			}

			fieldNamesStr := ""
			for i, name := range fieldNames {
				if i > 0 {
					fieldNamesStr += ", "
				}
				fieldNamesStr += name
			}

			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf(
					"The relation field `%s` uses the scalar fields %s. At least one of those fields is optional. Hence the relation field must be optional as well.",
					forwardField.Name(),
					fieldNamesStr,
				),
				astField.Span(),
			))
		}
	}
}

// validateRelationSameLengthAdvanced validates that referencing and referenced fields have the same length.
func validateRelationSameLengthAdvanced(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	if forwardField == nil {
		return
	}

	referencingFields := forwardField.ReferencingFields()
	referencedFields := forwardField.ReferencedFields()

	if len(referencingFields) == 0 && len(referencedFields) == 0 {
		return
	}

	if len(referencingFields) != len(referencedFields) {
		astField := forwardField.AstField()
		if astField == nil {
			return
		}

		// Try to get the relation attribute span
		span := astField.Span()
		// TODO: Get span from relation attribute when available

		ctx.PushError(diagnostics.NewValidationError(
			"You must specify the same number of fields in `fields` and `references`.",
			span,
		))
	}
}
