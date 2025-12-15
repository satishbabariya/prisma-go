// Package pslcore provides relation validation functions.
package validation

import (
	"fmt"
	"sort"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateRelationReferencesUniqueFields validates that relations reference unique fields.
func validateRelationReferencesUniqueFields(relation *database.RelationWalker, ctx *ValidationContext) {
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

	// Get forward relation field
	forwardField := inline.ForwardRelationField()
	if forwardField == nil {
		return
	}

	referencedFields := forwardField.ReferencedFields()
	if len(referencedFields) == 0 {
		return
	}

	referencedModel := inline.ReferencedModel()
	if referencedModel == nil {
		return
	}

	// Helper function to check if a set of fields matches a unique constraint
	fieldsMatchUniqueConstraint := func(fieldIDs []database.ScalarFieldId, fieldNames []string) bool {
		// Sort field names for comparison
		sortedNames := make([]string, len(fieldNames))
		copy(sortedNames, fieldNames)
		sort.Strings(sortedNames)

		// Check primary key
		if pk := referencedModel.PrimaryKey(); pk != nil {
			pkFields := pk.Fields()
			if len(pkFields) == len(fieldIDs) {
				pkFieldIDs := make([]database.ScalarFieldId, len(pkFields))
				pkFieldNames := make([]string, len(pkFields))
				for i, f := range pkFields {
					pkFieldIDs[i] = f.FieldID()
					pkFieldNames[i] = f.ScalarField().Name()
				}
				sort.Strings(pkFieldNames)

				// Check if field IDs match
				idsMatch := true
				for i := range fieldIDs {
					if fieldIDs[i] != pkFieldIDs[i] {
						idsMatch = false
						break
					}
				}

				// Check if field names match (for cases where IDs might differ)
				namesMatch := true
				for i := range sortedNames {
					if sortedNames[i] != pkFieldNames[i] {
						namesMatch = false
						break
					}
				}

				if idsMatch || namesMatch {
					return true
				}
			}
		}

		// Check unique indexes
		for _, index := range referencedModel.Indexes() {
			if !index.IsUnique() {
				continue
			}

			indexFields := index.Fields()
			if len(indexFields) == len(fieldIDs) {
				indexFieldIDs := make([]database.ScalarFieldId, len(indexFields))
				indexFieldNames := make([]string, len(indexFields))
				for i, f := range indexFields {
					indexFieldIDs[i] = f.FieldID()
					sf := f.ScalarField()
					if sf != nil {
						indexFieldNames[i] = sf.Name()
					}
				}
				sort.Strings(indexFieldNames)

				// Check if field IDs match
				idsMatch := true
				for i := range fieldIDs {
					if fieldIDs[i] != indexFieldIDs[i] {
						idsMatch = false
						break
					}
				}

				// Check if field names match
				namesMatch := true
				for i := range sortedNames {
					if sortedNames[i] != indexFieldNames[i] {
						namesMatch = false
						break
					}
				}

				if idsMatch || namesMatch {
					return true
				}
			}
		}

		return false
	}

	// Get referenced field IDs and names
	referencedFieldIDs := make([]database.ScalarFieldId, len(referencedFields))
	referencedFieldNames := make([]string, len(referencedFields))
	for i, f := range referencedFields {
		referencedFieldIDs[i] = database.ScalarFieldId(f.FieldID())
		referencedFieldNames[i] = f.Name()
	}

	// Check if referenced fields form a unique constraint
	if fieldsMatchUniqueConstraint(referencedFieldIDs, referencedFieldNames) {
		// Fields match a unique constraint, now check order
		validateReferencingFieldsInCorrectOrder(inline, forwardField, ctx)
		return
	}

	// Fields don't match a unique constraint, report error
	astField := forwardField.AstField()
	if astField == nil {
		return
	}

	fieldNamesStr := ""
	for i, name := range referencedFieldNames {
		if i > 0 {
			fieldNamesStr += ", "
		}
		fieldNamesStr += name
	}

	message := ""
	if len(referencedFieldNames) == 1 {
		message = fmt.Sprintf(
			"The argument `references` must refer to a unique criterion in the related model. Consider adding an `@unique` attribute to the field `%s` in the model `%s`.",
			fieldNamesStr,
			referencedModel.Name(),
		)
	} else {
		message = fmt.Sprintf(
			"The argument `references` must refer to a unique criterion in the related model. Consider adding an `@@unique([%s])` attribute to the model `%s`.",
			fieldNamesStr,
			referencedModel.Name(),
		)
	}

	ctx.PushError(diagnostics.NewAttributeValidationError(
		message,
		"@relation",
		diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), forwardField.Model().FileID()),
	))
}

// validateReferencingFieldsInCorrectOrder validates that referencing fields are in the correct order.
func validateReferencingFieldsInCorrectOrder(relation *database.InlineRelationWalker, forwardField *database.RelationFieldWalker, ctx *ValidationContext) {
	referencedFields := forwardField.ReferencedFields()
	if len(referencedFields) <= 1 {
		return
	}

	// Check if connector supports arbitrary order
	// Note: ConnectorCapabilityRelationFieldsInArbitraryOrder doesn't exist yet, skip for now
	// if ctx.HasCapability(ConnectorCapabilityRelationFieldsInArbitraryOrder) {
	// 	return
	// }

	referencedModel := relation.ReferencedModel()
	if referencedModel == nil {
		return
	}

	// Get referenced field names in order
	referencedFieldNames := make([]string, len(referencedFields))
	for i, f := range referencedFields {
		referencedFieldNames[i] = f.Name()
	}

	// Check if any unique criterion matches the order
	orderCorrect := false

	// Check primary key
	if pk := referencedModel.PrimaryKey(); pk != nil {
		pkFields := pk.Fields()
		if len(pkFields) == len(referencedFields) {
			pkFieldNames := make([]string, len(pkFields))
			for i, f := range pkFields {
				pkFieldNames[i] = f.ScalarField().Name()
			}

			// Check if order matches
			matches := true
			for i := range referencedFieldNames {
				if referencedFieldNames[i] != pkFieldNames[i] {
					matches = false
					break
				}
			}
			if matches {
				orderCorrect = true
			}
		}
	}

	// Check unique indexes
	if !orderCorrect {
		for _, index := range referencedModel.Indexes() {
			if !index.IsUnique() {
				continue
			}

			indexFields := index.Fields()
			if len(indexFields) == len(referencedFields) {
				indexFieldNames := make([]string, len(indexFields))
				for i, f := range indexFields {
					sf := f.ScalarField()
					if sf != nil {
						indexFieldNames[i] = sf.Name()
					}
				}

				// Check if order matches
				matches := true
				for i := range referencedFieldNames {
					if referencedFieldNames[i] != indexFieldNames[i] {
						matches = false
						break
					}
				}
				if matches {
					orderCorrect = true
					break
				}
			}
		}
	}

	if orderCorrect {
		return
	}

	// Order is incorrect, report error
	astField := forwardField.AstField()
	if astField == nil {
		return
	}

	fieldNamesStr := ""
	for i, name := range referencedFieldNames {
		if i > 0 {
			fieldNamesStr += ", "
		}
		fieldNamesStr += name
	}

	ctx.PushError(diagnostics.NewValidationError(
		fmt.Sprintf(
			"The argument `references` must refer to a unique criterion in the related model `%s` using the same order of fields. Please check the ordering in the following fields: `%s`.",
			referencedModel.Name(),
			fieldNamesStr,
		),
		diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), forwardField.Model().FileID()),
	))
}

// validateRelationSameLength validates that referencing and referenced fields have the same length.
func validateRelationSameLength(relation *database.RelationWalker, ctx *ValidationContext) {
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

		ctx.PushError(diagnostics.NewValidationError(
			"You must specify the same number of fields in `fields` and `references`.",
			diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), forwardField.Model().FileID()),
		))
	}
}

// validateRelationArity validates field arity in relations.
func validateRelationArity(relation *database.RelationWalker, ctx *ValidationContext) {
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

	astField := forwardField.AstField()
	if astField == nil {
		return
	}

	// If the relation field is required, all referencing fields must also be required
	if !astField.Arity.IsOptional() {
		referencingFields := forwardField.ReferencingFields()
		if len(referencingFields) == 0 {
			return
		}

		// Check if any referencing field is optional
		hasOptionalField := false
		fieldNames := make([]string, len(referencingFields))
		for i, field := range referencingFields {
			fieldNames[i] = field.Name()
			fieldAst := field.AstField()
			if fieldAst != nil && fieldAst.Arity.IsOptional() {
				hasOptionalField = true
			}
		}

		if hasOptionalField {
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
				diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), forwardField.Model().FileID()),
			))
		}
	}

	// Check if this is a one-to-one relation
	if inline.IsOneToOne() {
		if astField.Arity.IsList() {
			ctx.PushError(diagnostics.NewValidationError(
				"One-to-one relation field cannot be a list.",
				diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), forwardField.Model().FileID()),
			))
		}
	}
}

// validateRelationRequiredCannotUseSetNull validates that required relations cannot use SetNull.
func validateRelationRequiredCannotUseSetNull(relation *database.RelationWalker, ctx *ValidationContext) {
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
