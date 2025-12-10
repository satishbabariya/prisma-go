// Package pslcore provides relation type validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateReferencingScalarFieldTypes validates that referencing and referenced scalar field types match.
func validateReferencingScalarFieldTypes(relation *database.RelationWalker, ctx *ValidationContext) {
	if relation.IsIgnored() {
		return
	}

	refined := relation.Refine()
	if refined == nil {
		return
	}

	inline := refined.AsInline()
	if inline == nil {
		return
	}

	forwardField := inline.ForwardRelationField()
	if forwardField == nil {
		return
	}

	// Get referencing and referenced fields
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

		if !fieldTypesMatch(referencingType, referencedType) {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf(
					"The type of the field `%s` (`%s`) does not match the type of the referenced field `%s` (`%s`) in the related model.",
					referencingFields[i].Name(),
					getScalarTypeDisplayNameForRelation(&referencingType),
					referencedFields[i].Name(),
					getScalarTypeDisplayNameForRelation(&referencedType),
				),
				astField.Span(),
			))
		}
	}
}

// fieldTypesMatch checks if two scalar field types match.
func fieldTypesMatch(referencing, referenced database.ScalarFieldType) bool {
	// Check if both are composite types
	if referencing.CompositeTypeID != nil && referenced.CompositeTypeID != nil {
		return *referencing.CompositeTypeID == *referenced.CompositeTypeID
	}

	// Check if both are enum types
	if referencing.EnumID != nil && referenced.EnumID != nil {
		return *referencing.EnumID == *referenced.EnumID
	}

	// Check if both are built-in scalars
	if referencing.BuiltInScalar != nil && referenced.BuiltInScalar != nil {
		return *referencing.BuiltInScalar == *referenced.BuiltInScalar
	}

	// Check if both are unsupported
	if referencing.Unsupported != nil && referenced.Unsupported != nil {
		return referencing.Unsupported.Name == referenced.Unsupported.Name
	}

	return false
}

// getScalarTypeDisplayNameForRelation returns a display name for a ScalarFieldType.
func getScalarTypeDisplayNameForRelation(scalarType *database.ScalarFieldType) string {
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

// validateRequiredRelationCannotUseSetNull validates that required relations cannot use SetNull.
func validateRequiredRelationCannotUseSetNullDetailed(relation *database.RelationWalker, ctx *ValidationContext) {
	if relation.IsIgnored() {
		return
	}

	refined := relation.Refine()
	if refined == nil {
		return
	}

	inline := refined.AsInline()
	if inline == nil {
		return
	}

	// Check forward relation field
	forwardField := inline.ForwardRelationField()
	if forwardField != nil {
		validateRequiredFieldSetNull(forwardField, ctx)
	}

	// Check back relation field
	backField := inline.BackRelationField()
	if backField != nil {
		validateRequiredFieldSetNull(backField, ctx)
	}
}

// validateRequiredFieldSetNull validates that a required field cannot use SetNull.
func validateRequiredFieldSetNull(field *database.RelationFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	astField := field.AstField()
	if astField == nil {
		return
	}

	// Check if any referencing field is required
	referencingFields := field.ReferencingFields()
	if len(referencingFields) == 0 {
		return
	}

	hasRequiredField := false
	for _, refField := range referencingFields {
		refAstField := refField.AstField()
		if refAstField != nil && !refAstField.FieldType.IsOptional() {
			hasRequiredField = true
			break
		}
	}

	if !hasRequiredField {
		return
	}

	// Check if onDelete or onUpdate is SetNull
	onDelete := field.OnDelete()
	onUpdate := field.OnUpdate()

	hasSetNullOnDelete := onDelete != nil && onDelete.Action == database.ReferentialActionSetNull
	hasSetNullOnUpdate := onUpdate != nil && onUpdate.Action == database.ReferentialActionSetNull

	if !hasSetNullOnDelete && !hasSetNullOnUpdate {
		return
	}

	// Get relation mode (default to Prisma if not set)
	relationMode := ctx.RelationMode
	if relationMode == "" {
		relationMode = RelationModePrisma
	}

	// Check if connector allows SetNull on non-nullable fields
	allowsSetNull := false
	if ctx.Connector != nil {
		allowsSetNull = ctx.Connector.AllowsSetNullReferentialActionOnNonNullableFields(relationMode)
	}

	span := astField.Span()

	if allowsSetNull {
		// Connector allows SetNull on non-nullable fields, but we should still warn
		// For now, we'll add an error since we don't have a warning system yet
		// TODO: Add warning support when available
		if hasSetNullOnDelete {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The `onDelete` referential action of a relation should not be set to `SetNull` when a referenced field is required. We recommend either to choose another referential action, or to make the referenced fields optional. Read more at https://pris.ly/d/postgres-set-null",
				"@relation",
				span,
			))
		}
		if hasSetNullOnUpdate {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The `onUpdate` referential action of a relation should not be set to `SetNull` when a referenced field is required. We recommend either to choose another referential action, or to make the referenced fields optional. Read more at https://pris.ly/d/postgres-set-null",
				"@relation",
				span,
			))
		}
	} else {
		// Connector does not allow SetNull on non-nullable fields, add error
		if hasSetNullOnDelete {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The `onDelete` referential action of a relation must not be set to `SetNull` when a referenced field is required. Either choose another referential action, or make the referenced fields optional.",
				"@relation",
				span,
			))
		}
		if hasSetNullOnUpdate {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The `onUpdate` referential action of a relation must not be set to `SetNull` when a referenced field is required. Either choose another referential action, or make the referenced fields optional.",
				"@relation",
				span,
			))
		}
	}
}

// validateRelationAmbiguity validates relation ambiguity.
func validateRelationAmbiguity(relation *database.RelationWalker, names *Names, ctx *ValidationContext) {
	if relation.IsIgnored() {
		return
	}

	// TODO: Implement relation ambiguity detection
	// This requires checking if multiple relation fields have the same relation name
	_ = names
}

// validateSelfRelationFields validates self-relation fields.
func validateSelfRelationFields(relation *database.RelationWalker, ctx *ValidationContext) {
	if relation.IsIgnored() {
		return
	}

	models := relation.Models()
	if len(models) != 2 {
		return
	}

	// Check if this is a self-relation
	if models[0] == models[1] {
		// Self-relations need at least two fields with the same relation name
		// TODO: Validate self-relation requirements
		_ = models
	}
}
