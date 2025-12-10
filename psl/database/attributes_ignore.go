// Package parserdatabase provides @ignore and @@ignore attribute handling.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// HandleScalarFieldIgnore handles @ignore on a scalar field.
func HandleScalarFieldIgnore(sfid ScalarFieldId, fieldType ScalarFieldType, ctx *Context) {
	// Fields of type Unsupported cannot take @ignore
	if fieldType.Unsupported != nil {
		ctx.PushAttributeValidationError("Fields of type `Unsupported` cannot take an `@ignore` attribute. They are already treated as ignored by the client due to their type.")
		return
	}

	// Set the field as ignored
	if int(sfid) < len(ctx.types.ScalarFields) {
		ctx.types.ScalarFields[sfid].IsIgnored = true
	}
}

// HandleRelationFieldIgnore handles @ignore on a relation field.
func HandleRelationFieldIgnore(rfid RelationFieldId, ctx *Context) {
	// Set the field as ignored
	if int(rfid) < len(ctx.types.RelationFields) {
		ctx.types.RelationFields[rfid].IsIgnored = true
	}
}

// HandleModelIgnore handles @@ignore on a model.
func HandleModelIgnore(modelID ModelId, modelAttrs *ModelAttributes, ctx *Context) {
	// Check for fields that already have @ignore
	// These should error because if the model is ignored, fields don't need @ignore
	scalarFields := ctx.types.RangeModelScalarFields(modelID)
	for _, entry := range scalarFields {
		if entry.Field.IsIgnored {
			astModel := getModelFromID(modelID, ctx)
			if astModel != nil && int(entry.Field.FieldID) < len(astModel.Fields) {
				astField := &astModel.Fields[entry.Field.FieldID]
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"Fields on an already ignored Model do not need an `@ignore` annotation.",
					"@ignore",
					astField.Name.Span(),
				))
			}
		}
	}

	// Also check relation fields
	relationFields := ctx.types.RangeModelRelationFields(modelID)
	for _, entry := range relationFields {
		if entry.Field.IsIgnored {
			astModel := getModelFromID(modelID, ctx)
			if astModel != nil && int(entry.Field.FieldID) < len(astModel.Fields) {
				astField := &astModel.Fields[entry.Field.FieldID]
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"Fields on an already ignored Model do not need an `@ignore` annotation.",
					"@ignore",
					astField.Name.Span(),
				))
			}
		}
	}

	// Set the model as ignored
	modelAttrs.IsIgnored = true
}
