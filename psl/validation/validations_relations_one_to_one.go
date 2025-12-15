// Package pslcore provides one-to-one relation validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateOneToOneBothSidesAreDefined validates that both sides of a one-to-one relation are defined.
func validateOneToOneBothSidesAreDefined(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	backField := relation.BackRelationField()
	if backField != nil {
		return
	}

	forwardField := relation.ForwardRelationField()
	if forwardField == nil {
		return
	}

	containerType := "model"
	model := forwardField.Model()
	astModel := model.AstModel()
	if astModel != nil && astModel.IsView() {
		containerType = "view"
	}

	// Get related model from the forward relation field
	referencedModel := forwardField.ReferencedModel()
	if referencedModel == nil {
		return
	}

	message := fmt.Sprintf(
		"The relation field `%s` on %s `%s` is missing an opposite relation field on the model `%s`. Either run `prisma format` or add it manually.",
		forwardField.Name(),
		containerType,
		model.Name(),
		referencedModel.Name(),
	)

	astField := forwardField.AstField()
	if astField != nil {
		ctx.PushError(diagnostics.NewFieldValidationError(
			message,
			containerType,
			model.Name(),
			forwardField.Name(),
			diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), model.FileID()),
		))
	}
}

// validateOneToOneFieldsAndReferencesAreDefined validates that fields and references are defined for one-to-one relations.
func validateOneToOneFieldsAndReferencesAreDefined(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	// Get referencing fields
	forwardFields := forwardField.ReferencingFields()
	backFields := backField.ReferencingFields()

	if len(forwardFields) == 0 && len(backFields) == 0 {
		message := fmt.Sprintf(
			"The relation fields `%s` on Model `%s` and `%s` on Model `%s` do not provide the `fields` argument in the @relation attribute. You have to provide it on one of the two fields.",
			forwardField.Name(),
			forwardField.Model().Name(),
			backField.Name(),
			backField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(backAstField.Pos.Offset, backAstField.Pos.Offset+len(backAstField.Name.Name), backField.Model().FileID()),
			))
		}
	}

	// Get referenced fields
	forwardRefs := forwardField.ReferencedFields()
	backRefs := backField.ReferencedFields()

	if len(forwardRefs) == 0 && len(backRefs) == 0 {
		message := fmt.Sprintf(
			"The relation fields `%s` on Model `%s` and `%s` on Model `%s` do not provide the `references` argument in the @relation attribute. You have to provide it on one of the two fields.",
			forwardField.Name(),
			forwardField.Model().Name(),
			backField.Name(),
			backField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(backAstField.Pos.Offset, backAstField.Pos.Offset+len(backAstField.Name.Name), backField.Model().FileID()),
			))
		}
	}
}

// validateOneToOneFieldsAndReferencesOnOneSideOnly validates that fields and references are only on one side.
func validateOneToOneFieldsAndReferencesOnOneSideOnly(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	// Get referenced fields
	forwardRefs := forwardField.ReferencedFields()
	backRefs := backField.ReferencedFields()

	// Check if both sides have references
	if len(forwardRefs) > 0 && len(backRefs) > 0 {
		message := fmt.Sprintf(
			"The relation fields `%s` on Model `%s` and `%s` on Model `%s` both provide the `references` argument in the @relation attribute. You have to provide it only on one of the two fields.",
			forwardField.Name(),
			forwardField.Model().Name(),
			backField.Name(),
			backField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(backAstField.Pos.Offset, backAstField.Pos.Offset+len(backAstField.Name.Name), backField.Model().FileID()),
			))
		}
	}

	// Get referencing fields
	forwardFields := forwardField.ReferencingFields()
	backFields := backField.ReferencingFields()

	// Check if both sides have fields
	if len(forwardFields) > 0 && len(backFields) > 0 {
		message := fmt.Sprintf(
			"The relation fields `%s` on Model `%s` and `%s` on Model `%s` both provide the `fields` argument in the @relation attribute. You have to provide it only on one of the two fields.",
			forwardField.Name(),
			forwardField.Model().Name(),
			backField.Name(),
			backField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(backAstField.Pos.Offset, backAstField.Pos.Offset+len(backAstField.Name.Name), backField.Model().FileID()),
			))
		}
	}
}

// validateOneToOneReferentialActions validates that referential actions are only on the forward side.
func validateOneToOneReferentialActions(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	// TODO: Check explicit_on_delete and explicit_on_update when exposed
	// For now, this is a placeholder
	_ = forwardField
	_ = backField
}

// validateOneToOneFieldsReferencesMixups validates that fields and references are not mixed up.
func validateOneToOneFieldsReferencesMixups(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	// Get referencing and referenced fields
	forwardFields := forwardField.ReferencingFields()
	backRefs := backField.ReferencedFields()

	if len(forwardFields) > 0 && len(backRefs) > 0 {
		message := fmt.Sprintf(
			"The relation field `%s` on Model `%s` provides the `fields` argument in the @relation attribute. And the related field `%s` on Model `%s` provides the `references` argument. You must provide both arguments on the same side.",
			forwardField.Name(),
			forwardField.Model().Name(),
			backField.Name(),
			backField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(backAstField.Pos.Offset, backAstField.Pos.Offset+len(backAstField.Name.Name), backField.Model().FileID()),
			))
		}
	}

	// Get referencing and referenced fields
	forwardRefs := forwardField.ReferencedFields()
	backFields := backField.ReferencingFields()

	if len(forwardRefs) > 0 && len(backFields) > 0 {
		message := fmt.Sprintf(
			"The relation field `%s` on Model `%s` provides the `references` argument in the @relation attribute. And the related field `%s` on Model `%s` provides the `fields` argument. You must provide both arguments on the same side.",
			forwardField.Name(),
			forwardField.Model().Name(),
			backField.Name(),
			backField.Model().Name(),
		)

		forwardAstField := forwardField.AstField()
		if forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}

		backAstField := backField.AstField()
		if backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(backAstField.Pos.Offset, backAstField.Pos.Offset+len(backAstField.Name.Name), backField.Model().FileID()),
			))
		}
	}
}

// validateOneToOneBackRelationArityIsOptional validates that the back relation side is optional.
func validateOneToOneBackRelationArityIsOptional(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	backAstField := backField.AstField()
	if backAstField == nil {
		return
	}

	// Check if back field is required
	if !backAstField.Arity.IsOptional() {
		message := fmt.Sprintf(
			"The relation field `%s` on Model `%s` is required. This is not valid because it's not possible to enforce this constraint on the database level. Please change the field type from `%s` to `%s?` to fix this.",
			backField.Name(),
			backField.Model().Name(),
			forwardField.Model().Name(),
			forwardField.Model().Name(),
		)

		if backAstField := backField.AstField(); backAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(backAstField.Pos.Offset, backAstField.Pos.Offset+len(backAstField.Name.Name), backField.Model().FileID()),
			))
		}
	}
}

// validateOneToOneFieldsAndReferencesOnWrongSide validates that fields and references are on the correct side of the relation.
func validateOneToOneFieldsAndReferencesOnWrongSide(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	// Only validate if there are no errors already
	if len(ctx.Diagnostics.Errors()) > 0 {
		return
	}

	forwardField := relation.ForwardRelationField()
	backField := relation.BackRelationField()

	if forwardField == nil || backField == nil {
		return
	}

	// Check if forward field is required and back field has fields/references
	if forwardField.IsRequired() {
		backRefFields := backField.ReferencingFields()
		backRefdFields := backField.ReferencedFields()

		if len(backRefFields) > 0 || len(backRefdFields) > 0 {
			message := fmt.Sprintf(
				"The relation field `%s.%s` defines the `fields` and/or `references` argument. You must set them on the required side of the relation (`%s.%s`) in order for the constraints to be enforced. Alternatively, you can change this field to be required and the opposite optional, or make both sides of the relation optional.",
				backField.Model().Name(),
				backField.Name(),
				forwardField.Model().Name(),
				forwardField.Name(),
			)

			astField := backField.AstField()
			if astField != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					message,
					"@relation",
					diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), backField.Model().FileID()),
				))
			}
		}
	}
}

// validateOneToOneFieldsMustBeUnique validates that one-to-one relation fields must be unique.
func validateOneToOneFieldsMustBeUnique(relation *database.InlineRelationWalker, ctx *ValidationContext) {
	forwardField := relation.ForwardRelationField()
	if forwardField == nil {
		return
	}

	referencingModel := relation.ReferencingModel()
	if referencingModel == nil {
		return
	}

	// Get referencing fields
	referencingFields := forwardField.ReferencingFields()
	if len(referencingFields) == 0 {
		return
	}

	// Check if referencing fields form a unique constraint
	isUnique := false

	// Check primary key
	pk := referencingModel.PrimaryKey()
	if pk != nil {
		pkFields := pk.Fields()
		if len(pkFields) == len(referencingFields) {
			matches := true
			for i, pkField := range pkFields {
				if pkField.FieldID() != database.ScalarFieldId(referencingFields[i].FieldID()) {
					matches = false
					break
				}
			}
			if matches {
				isUnique = true
			}
		}
	}

	// Check unique indexes
	if !isUnique {
		for _, index := range referencingModel.Indexes() {
			if !index.IsUnique() {
				continue
			}
			indexFields := index.Fields()
			if len(indexFields) == len(referencingFields) {
				matches := true
				for i, indexField := range indexFields {
					if indexField.FieldID() != database.ScalarFieldId(referencingFields[i].FieldID()) {
						matches = false
						break
					}
				}
				if matches {
					isUnique = true
					break
				}
			}
		}
	}

	if isUnique {
		return
	}

	forwardAstField := forwardField.AstField()
	if forwardAstField == nil {
		return
	}

	// Get field names
	fieldNames := make([]string, len(referencingFields))
	for i, field := range referencingFields {
		fieldNames[i] = field.Name()
	}

	if len(fieldNames) == 1 {
		message := fmt.Sprintf(
			"A one-to-one relation must use unique fields on the defining side. Either add an `@unique` attribute to the field `%s`, or change the relation to one-to-many.",
			fieldNames[0],
		)
		if forwardAstField := forwardField.AstField(); forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}
	} else if len(fieldNames) > 1 {
		message := fmt.Sprintf(
			"A one-to-one relation must use unique fields on the defining side. Either add an `@@unique([%s])` attribute to the model, or change the relation to one-to-many.",
			fmt.Sprintf("%s", fieldNames),
		)
		if forwardAstField := forwardField.AstField(); forwardAstField != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				diagnostics.NewSpan(forwardAstField.Pos.Offset, forwardAstField.Pos.Offset+len(forwardAstField.Name.Name), forwardField.Model().FileID()),
			))
		}
	}
}
