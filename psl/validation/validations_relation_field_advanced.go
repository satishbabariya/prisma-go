// Package pslcore provides advanced relation field validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// isLeftwiseIncluded checks if a subgroup is left-wise included in a supergroup.
// A subgroup is left-wise included in a supergroup if the subgroup is contained in the supergroup,
// and all the left-most entries of the supergroup match the order of definitions of the subgroup.
// More formally: { x_1, x_2, ..., x_n } is left-wise included in { y_1, y_2, ..., y_m } if and only if
// n <= m and x_i = y_i for all i in [1, n].
func isLeftwiseIncluded(subgroup []database.ScalarFieldId, supergroup []database.ScalarFieldId) bool {
	if len(subgroup) > len(supergroup) {
		return false
	}

	for i := 0; i < len(subgroup); i++ {
		if subgroup[i] != supergroup[i] {
			return false
		}
	}

	return true
}

// convertReferentialAction converts database.ReferentialAction (string) to pslcore.ReferentialAction (int).
func convertReferentialAction(action database.ReferentialAction) ReferentialAction {
	switch string(action) {
	case "Cascade":
		return ReferentialActionCascade
	case "Restrict":
		return ReferentialActionRestrict
	case "NoAction":
		return ReferentialActionNoAction
	case "SetNull":
		return ReferentialActionSetNull
	case "SetDefault":
		return ReferentialActionSetDefault
	default:
		return ReferentialActionNoAction
	}
}

// validateRelationFieldAmbiguity validates relation field ambiguity.
func validateRelationFieldAmbiguity(field *database.RelationFieldWalker, names *Names, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	model := field.Model()
	refModel := field.ReferencedModel()

	if model == nil || refModel == nil {
		return
	}

	// TODO: Check for relation name ambiguity when relation name tracking is available
	// This requires checking if multiple fields in the same model reference the same related model
	// with the same relation name
	_ = names
}

// validateRelationFieldIgnoredRelatedModel validates that relation fields to ignored models are also ignored.
func validateRelationFieldIgnoredRelatedModel(field *database.RelationFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	model := field.Model()
	refModel := field.ReferencedModel()

	if model == nil || refModel == nil {
		return
	}

	// If the related model is ignored, the relation field must also be ignored
	if refModel.IsIgnored() && !model.IsIgnored() {
		astField := field.AstField()
		if astField != nil {
			span := diagnostics.NewSpan(0, 0, model.FileID())
			message := fmt.Sprintf(
				"The relation field `%s` on Model `%s` must specify the `@ignore` attribute, because the model %s it is pointing to is marked ignored.",
				field.Name(),
				model.Name(),
				refModel.Name(),
			)

			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@ignore",
				span,
			))
		}
	}
}

// validateRelationFieldReferentialActions validates referential actions on relation fields.
func validateRelationFieldReferentialActionsAdvanced(field *database.RelationFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	if ctx.Connector == nil {
		return
	}

	astField := field.AstField()
	if astField == nil {
		return
	}

	// Get relation mode (default to Prisma if not set)
	relationMode := ctx.RelationMode
	if relationMode == "" {
		relationMode = RelationModePrisma
	}

	// Helper function to format allowed actions
	fmtAllowedActions := func(actions []ReferentialAction) string {
		if len(actions) == 0 {
			return ""
		}
		result := ""
		for i, action := range actions {
			if i > 0 {
				result += ", "
			}
			result += fmt.Sprintf("`%s`", action.String())
		}
		return result
	}

	// Helper function to create error message
	createErrorMessage := func(action ReferentialAction) string {
		var allowedActions []ReferentialAction
		if relationMode == RelationModeForeignKeys {
			allowedActions = ctx.Connector.ReferentialActions(relationMode)
		} else {
			allowedActions = ctx.Connector.EmulatedReferentialActions()
		}

		allowedStr := fmtAllowedActions(allowedActions)
		message := fmt.Sprintf(
			"Invalid referential action: `%s`. Allowed values: (%s)",
			action.String(),
			allowedStr,
		)

		// Add additional info for NoAction in Prisma mode
		if relationMode == RelationModePrisma && action == ReferentialActionNoAction {
			emulatedActions := ctx.Connector.EmulatedReferentialActions()
			hasRestrict := false
			for _, a := range emulatedActions {
				if a == ReferentialActionRestrict {
					hasRestrict = true
					break
				}
			}
			if hasRestrict {
				message += fmt.Sprintf(
					". `%s` is not implemented for %s when using `relationMode = \"prisma\"`, you could try using `%s` instead. Learn more at https://pris.ly/d/relation-mode",
					action.String(),
					ctx.Connector.ProviderName(),
					ReferentialActionRestrict.String(),
				)
			}
		}

		return message
	}

	// Validate onDelete
	onDelete := field.OnDelete()
	if onDelete != nil {
		action := convertReferentialAction(onDelete.Action)
		if !ctx.Connector.SupportsReferentialAction(relationMode, action) {
			span := diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), field.Model().FileID())
			// TODO: Get span for "onDelete" argument when span_for_argument is available
			ctx.PushError(diagnostics.NewValidationError(
				createErrorMessage(action),
				span,
			))
		}
	}

	// Validate onUpdate
	onUpdate := field.OnUpdate()
	if onUpdate != nil {
		action := convertReferentialAction(onUpdate.Action)
		if !ctx.Connector.SupportsReferentialAction(relationMode, action) {
			span := diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), field.Model().FileID())
			// TODO: Get span for "onUpdate" argument when span_for_argument is available
			ctx.PushError(diagnostics.NewValidationError(
				createErrorMessage(action),
				span,
			))
		}
	}
}

// validateRelationFieldMap validates the map argument on relation fields.
func validateRelationFieldMap(field *database.RelationFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	// TODO: Get mapped name when MappedName() method is available on RelationFieldWalker
	// If mapped name exists:
	// - Check if connector supports named foreign keys
	// - Validate database name format
	_ = field
}

// validateRelationFieldMissingIndexes validates that relation fields have proper indexes.
func validateRelationFieldMissingIndexes(field *database.RelationFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	if ctx.Connector == nil {
		return
	}

	// Check if connector suggests missing indexes
	if !ctx.Connector.ShouldSuggestMissingReferencingFieldsIndexes() {
		return
	}

	// Check if relation mode is Prisma
	if ctx.RelationMode != RelationModePrisma {
		return
	}

	// Get referencing fields
	referencingFields := field.ReferencingFields()
	if len(referencingFields) == 0 {
		return
	}

	model := field.Model()
	if model == nil {
		return
	}

	// Get referencing field IDs
	referencingFieldIDs := make([]database.ScalarFieldId, len(referencingFields))
	for i, f := range referencingFields {
		referencingFieldIDs[i] = database.ScalarFieldId(f.FieldID())
	}

	// Check if referencing fields are covered by any index
	for _, index := range model.Indexes() {
		indexFields := index.Fields()
		indexFieldIDs := make([]database.ScalarFieldId, len(indexFields))
		for i, f := range indexFields {
			indexFieldIDs[i] = database.ScalarFieldId(f.FieldID())
		}

		if isLeftwiseIncluded(referencingFieldIDs, indexFieldIDs) {
			return
		}
	}

	// Check if referencing fields are covered by primary key
	if pk := model.PrimaryKey(); pk != nil {
		pkFields := pk.Fields()
		pkFieldIDs := make([]database.ScalarFieldId, len(pkFields))
		for i, f := range pkFields {
			pkFieldIDs[i] = f.FieldID()
		}

		if isLeftwiseIncluded(referencingFieldIDs, pkFieldIDs) {
			return
		}
	}

	// Referencing fields are not covered by any index or primary key
	astField := field.AstField()
	if astField != nil {
		fieldNames := make([]string, len(referencingFields))
		for i, f := range referencingFields {
			fieldNames[i] = f.Name()
		}
		ctx.PushWarning(diagnostics.NewDatamodelWarning(
			fmt.Sprintf("The relation field '%s' on model '%s' references fields [%s] that are not covered by any index or primary key. This may cause performance issues.", field.Name(), model.Name(), fmt.Sprintf("%v", fieldNames)),
			diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), field.Model().FileID()),
		))
	}
}
