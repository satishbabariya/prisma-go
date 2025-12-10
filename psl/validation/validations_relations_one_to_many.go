// Package pslcore provides one-to-many relation validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateOneToManyBothSidesAreDefined validates that both sides of a one-to-many relation are defined.
func validateOneToManyBothSidesAreDefined(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil && backField == nil {
		return
	}

	var errorField *database.RelationFieldWalker
	if forwardField != nil && backField == nil {
		errorField = forwardField
	} else if forwardField == nil && backField != nil {
		errorField = backField
	} else {
		return
	}

	containerType := "model"
	model := errorField.Model()
	astModel := model.AstModel()
	if astModel != nil && astModel.IsView {
		containerType = "view"
	}

	// TODO: Get related model when RelatedModel() method is available
	// For now, get it from the relation
	var relatedModel *database.ModelWalker
	if forwardField != nil {
		relatedModel = relation.ReferencedModel()
	} else if backField != nil {
		relatedModel = relation.ReferencingModel()
	}
	if relatedModel == nil {
		return
	}

	message := fmt.Sprintf(
		"The relation field `%s` on %s `%s` is missing an opposite relation field on the model `%s`. Either run `prisma format` or add it manually.",
		errorField.Name(),
		containerType,
		model.Name(),
		relatedModel.Name(),
	)

	astField := errorField.AstField()
	if astField != nil {
		ctx.PushError(diagnostics.NewFieldValidationError(
			message,
			containerType,
			model.Name(),
			errorField.Name(),
			astField.Span(),
		))
	}
}

// validateOneToManyFieldsAndReferencesAreDefined validates that fields and references are defined for one-to-many relations.
func validateOneToManyFieldsAndReferencesAreDefined(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	// Helper function to check if fields are empty
	isEmptyFields := func(fields []*database.ScalarFieldWalker) bool {
		return len(fields) == 0
	}

	// Check if forward field has referencing fields (fields argument)
	forwardFields := forwardField.ReferencingFields()
	if isEmptyFields(forwardFields) {
		message := fmt.Sprintf(
			"The relation field `%s` on Model `%s` must specify the `fields` argument in the @relation attribute. You can run `prisma format` to fix this automatically.",
			forwardField.Name(),
			forwardField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				forwardAstField.Span(),
			))
		}
	}

	// Check if forward field has referenced fields (references argument)
	forwardRefs := forwardField.ReferencedFields()
	if isEmptyFields(forwardRefs) {
		message := fmt.Sprintf(
			"The relation field `%s` on Model `%s` must specify the `references` argument in the @relation attribute.",
			forwardField.Name(),
			forwardField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				forwardAstField.Span(),
			))
		}
	}

	// Check if back field has fields or references (should not have them)
	backFields := backField.ReferencingFields()
	backRefs := backField.ReferencedFields()

	if !isEmptyFields(backFields) || !isEmptyFields(backRefs) {
		message := fmt.Sprintf(
			"The relation field `%s` on Model `%s` must not specify the `fields` or `references` argument in the @relation attribute. You must only specify it on the opposite field `%s` on model `%s`.",
			backField.Name(),
			backField.Model().Name(),
			forwardField.Name(),
			forwardField.Model().Name(),
		)

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				backAstField.Span(),
			))
		}
	}
}

// validateOneToManyReferentialActions validates that referential actions are only on the forward side.
func validateOneToManyReferentialActions(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	// Back side should not have referential actions
	backOnDelete := backField.OnDelete()
	backOnUpdate := backField.OnUpdate()

	if backOnDelete != nil || backOnUpdate != nil {
		message := fmt.Sprintf(
			"The relation field `%s` on Model `%s` must not specify the `onDelete` or `onUpdate` argument in the @relation attribute. You must only specify it on the opposite field `%s` on model `%s`, or in case of a many to many relation, in an explicit join table.",
			backField.Name(),
			backField.Model().Name(),
			forwardField.Name(),
			forwardField.Model().Name(),
		)

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				backAstField.Span(),
			))
		}
	}
}
