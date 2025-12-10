// Package pslcore provides many-to-many relation validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateImplicitManyToManySingularId validates that implicit many-to-many relations require singular ID fields.
func validateImplicitManyToManySingularId(relation *database.ImplicitManyToManyRelationWalker, ctx *ValidationContext) {
	fieldA := relation.FieldA()
	fieldB := relation.FieldB()

	if fieldA == nil || fieldB == nil {
		return
	}

	// Helper function to check if a model has a single ID field
	hasSingleIdField := func(model *database.ModelWalker) bool {
		if model == nil {
			return false
		}
		pk := model.PrimaryKey()
		if pk == nil {
			return false
		}
		fields := pk.Fields()
		return len(fields) == 1
	}

	// Check field A's related model
	relatedModelA := fieldA.ReferencedModel()
	if relatedModelA != nil && !hasSingleIdField(relatedModelA) {
		containerType := "model"
		astModelA := relatedModelA.AstModel()
		if astModelA != nil && astModelA.IsView {
			containerType = "view"
		}

		message := fmt.Sprintf(
			"The relation field `%s` on %s `%s` references `%s` which does not have an `@id` field. Models without `@id` cannot be part of a many to many relation. Use an explicit intermediate Model to represent this relationship.",
			fieldA.Name(),
			containerType,
			fieldA.Model().Name(),
			relatedModelA.Name(),
		)

		astFieldA := fieldA.AstField()
		if astFieldA != nil {
			ctx.PushError(diagnostics.NewFieldValidationError(
				message,
				containerType,
				fieldA.Model().Name(),
				fieldA.Name(),
				astFieldA.Span(),
			))
		}

		// Check if references point to singular ID field
		referencedFields := fieldA.ReferencedFields()
		if len(referencedFields) > 0 {
			// Check if referenced fields are not the primary key
			pk := relatedModelA.PrimaryKey()
			if pk != nil {
				pkFields := pk.Fields()
				if len(pkFields) != len(referencedFields) || len(pkFields) != 1 {
					// Not referencing singular ID
					fieldNames := make([]string, len(referencedFields))
					for i, f := range referencedFields {
						fieldNames[i] = f.Name()
					}

					message := fmt.Sprintf(
						"Implicit many-to-many relations must always reference the id field of the related model. Change the argument `references` to use the id field of the related model `%s`. But it is referencing the following fields that are not the id: %s",
						relatedModelA.Name(),
						fmt.Sprintf("%v", fieldNames),
					)

					if astFieldA != nil {
						ctx.PushError(diagnostics.NewValidationError(
							message,
							astFieldA.Span(),
						))
					}
				}
			}
		}
	}

	// Check field B's related model
	relatedModelB := fieldB.ReferencedModel()
	if relatedModelB != nil && !hasSingleIdField(relatedModelB) {
		containerType := "model"
		astModelB := relatedModelB.AstModel()
		if astModelB != nil && astModelB.IsView {
			containerType = "view"
		}

		message := fmt.Sprintf(
			"The relation field `%s` on %s `%s` references `%s` which does not have an `@id` field. Models without `@id` cannot be part of a many to many relation. Use an explicit intermediate Model to represent this relationship.",
			fieldB.Name(),
			containerType,
			fieldB.Model().Name(),
			relatedModelB.Name(),
		)

		astFieldB := fieldB.AstField()
		if astFieldB != nil {
			ctx.PushError(diagnostics.NewFieldValidationError(
				message,
				containerType,
				fieldB.Model().Name(),
				fieldB.Name(),
				astFieldB.Span(),
			))
		}

		// Check if references point to singular ID field
		referencedFields := fieldB.ReferencedFields()
		if len(referencedFields) > 0 {
			// Check if referenced fields are not the primary key
			pk := relatedModelB.PrimaryKey()
			if pk != nil {
				pkFields := pk.Fields()
				if len(pkFields) != len(referencedFields) || len(pkFields) != 1 {
					// Not referencing singular ID
					fieldNames := make([]string, len(referencedFields))
					for i, f := range referencedFields {
						fieldNames[i] = f.Name()
					}

					message := fmt.Sprintf(
						"Implicit many-to-many relations must always reference the id field of the related model. Change the argument `references` to use the id field of the related model `%s`. But it is referencing the following fields that are not the id: %s",
						relatedModelB.Name(),
						fmt.Sprintf("%v", fieldNames),
					)

					if astFieldB != nil {
						ctx.PushError(diagnostics.NewValidationError(
							message,
							astFieldB.Span(),
						))
					}
				}
			}
		}
	}
}

// validateImplicitManyToManySupports validates that implicit many-to-many relations are supported.
func validateImplicitManyToManySupports(relation *database.ImplicitManyToManyRelationWalker, ctx *ValidationContext) {
	fieldA := relation.FieldA()
	fieldB := relation.FieldB()

	if fieldA == nil || fieldB == nil {
		return
	}

	// Check if one side is a view
	modelA := fieldA.Model()
	modelB := fieldB.Model()
	astModelA := modelA.AstModel()
	astModelB := modelB.AstModel()

	isViewA := astModelA != nil && astModelA.IsView
	isViewB := astModelB != nil && astModelB.IsView

	if isViewA || isViewB {
		message := "Implicit many-to-many relations are not supported for views."
		astFieldA := fieldA.AstField()
		astFieldB := fieldB.AstField()

		if astFieldA != nil {
			ctx.PushError(diagnostics.NewValidationError(
				message,
				astFieldA.Span(),
			))
		}

		if astFieldB != nil {
			ctx.PushError(diagnostics.NewValidationError(
				message,
				astFieldB.Span(),
			))
		}
		return
	}

	// Check connector capability
	if !ctx.HasCapability(ConnectorCapabilityImplicitManyToManyRelation) {
		connectorName := "the current connector"
		if ctx.Connector != nil {
			connectorName = ctx.Connector.ProviderName()
		}

		message := fmt.Sprintf(
			"Implicit many-to-many relations are not supported on %s. Please use the syntax defined in https://pris.ly/d/document-database-many-to-many",
			connectorName,
		)

		astFieldA := fieldA.AstField()
		astFieldB := fieldB.AstField()

		if astFieldA != nil {
			ctx.PushError(diagnostics.NewValidationError(
				message,
				astFieldA.Span(),
			))
		}

		if astFieldB != nil {
			ctx.PushError(diagnostics.NewValidationError(
				message,
				astFieldB.Span(),
			))
		}
	}
}

// validateImplicitManyToManyCannotDefineReferences validates that implicit many-to-many relations cannot have references argument.
func validateImplicitManyToManyCannotDefineReferences(relation *database.ImplicitManyToManyRelationWalker, ctx *ValidationContext) {
	fieldA := relation.FieldA()
	fieldB := relation.FieldB()

	message := "Implicit many-to-many relation should not have references argument defined. Either remove it, or change the relation to one-to-many."

	// Check field A
	if fieldA != nil {
		referencedFields := fieldA.ReferencedFields()
		if len(referencedFields) > 0 {
			astFieldA := fieldA.AstField()
			if astFieldA != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					message,
					"@relation",
					astFieldA.Span(),
				))
			}
		}
	}

	// Check field B
	if fieldB != nil {
		referencedFields := fieldB.ReferencedFields()
		if len(referencedFields) > 0 {
			astFieldB := fieldB.AstField()
			if astFieldB != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					message,
					"@relation",
					astFieldB.Span(),
				))
			}
		}
	}
}

// validateEmbeddedManyToManySupports validates that embedded many-to-many relations are supported.
func validateEmbeddedManyToManySupports(relation *database.TwoWayEmbeddedManyToManyRelationWalker, ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityTwoWayEmbeddedManyToManyRelation) {
		return
	}

	fieldA := relation.FieldA()
	fieldB := relation.FieldB()

	if fieldA == nil || fieldB == nil {
		return
	}

	connectorName := "the current connector"
	if ctx.Connector != nil {
		connectorName = ctx.Connector.ProviderName()
	}

	message := fmt.Sprintf(
		"Embedded many-to-many relations are not supported on %s. Please use the syntax defined in https://pris.ly/d/relational-database-many-to-many",
		connectorName,
	)

	astFieldA := fieldA.AstField()
	astFieldB := fieldB.AstField()

	if astFieldA != nil {
		ctx.PushError(diagnostics.NewValidationError(
			message,
			astFieldA.Span(),
		))
	}

	if astFieldB != nil {
		ctx.PushError(diagnostics.NewValidationError(
			message,
			astFieldB.Span(),
		))
	}
}

// validateEmbeddedManyToManyDefinesReferences validates that embedded many-to-many relations define references on both sides.
func validateEmbeddedManyToManyDefinesReferences(relation *database.TwoWayEmbeddedManyToManyRelationWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityTwoWayEmbeddedManyToManyRelation) {
		return
	}

	fieldA := relation.FieldA()
	fieldB := relation.FieldB()

	if fieldA == nil || fieldB == nil {
		return
	}

	message := "The `references` argument must be defined and must point to exactly one scalar field. https://pris.ly/d/many-to-many-relations"

	// Check field A
	referencedFieldsA := fieldA.ReferencedFields()
	if len(referencedFieldsA) != 1 {
		astFieldA := fieldA.AstField()
		if astFieldA != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				astFieldA.Span(),
			))
		}
	}

	// Check field B
	referencedFieldsB := fieldB.ReferencedFields()
	if len(referencedFieldsB) != 1 {
		astFieldB := fieldB.AstField()
		if astFieldB != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				astFieldB.Span(),
			))
		}
	}
}

// validateEmbeddedManyToManyDefinesFields validates that embedded many-to-many relations define fields on both sides.
func validateEmbeddedManyToManyDefinesFields(relation *database.TwoWayEmbeddedManyToManyRelationWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityTwoWayEmbeddedManyToManyRelation) {
		return
	}

	fieldA := relation.FieldA()
	fieldB := relation.FieldB()

	if fieldA == nil || fieldB == nil {
		return
	}

	message := "The `fields` argument must be defined and must point to exactly one scalar field. https://pris.ly/d/many-to-many-relations"

	// Check field A
	referencingFieldsA := fieldA.ReferencingFields()
	if len(referencingFieldsA) != 1 {
		astFieldA := fieldA.AstField()
		if astFieldA != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				astFieldA.Span(),
			))
		}
	}

	// Check field B
	referencingFieldsB := fieldB.ReferencingFields()
	if len(referencingFieldsB) != 1 {
		astFieldB := fieldB.AstField()
		if astFieldB != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				message,
				"@relation",
				astFieldB.Span(),
			))
		}
	}
}

// validateEmbeddedManyToManyReferencesId validates that embedded many-to-many relations reference ID fields.
func validateEmbeddedManyToManyReferencesId(relation *database.TwoWayEmbeddedManyToManyRelationWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityTwoWayEmbeddedManyToManyRelation) {
		return
	}

	fieldA := relation.FieldA()
	fieldB := relation.FieldB()

	if fieldA == nil || fieldB == nil {
		return
	}

	message := "The `references` argument must point to a singular `id` field"

	// Helper to check if a field is a singular primary key
	isSingularPK := func(field *database.ScalarFieldWalker) bool {
		if field == nil {
			return false
		}
		model := field.Model()
		if model == nil {
			return false
		}
		pk := model.PrimaryKey()
		if pk == nil {
			return false
		}
		pkFields := pk.Fields()
		if len(pkFields) != 1 {
			return false
		}
		return pkFields[0].FieldID() == database.ScalarFieldId(field.FieldID())
	}

	// Check field A
	referencedFieldsA := fieldA.ReferencedFields()
	if len(referencedFieldsA) == 1 {
		if !isSingularPK(referencedFieldsA[0]) {
			astFieldA := fieldA.AstField()
			if astFieldA != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					message,
					"@relation",
					astFieldA.Span(),
				))
			}
		}
	}

	// Check field B
	referencedFieldsB := fieldB.ReferencedFields()
	if len(referencedFieldsB) == 1 {
		if !isSingularPK(referencedFieldsB[0]) {
			astFieldB := fieldB.AstField()
			if astFieldB != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					message,
					"@relation",
					astFieldB.Span(),
				))
			}
		}
	}
}
