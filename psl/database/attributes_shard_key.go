// Package parserdatabase provides @shardKey and @@shardKey attribute handling.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// HandleModelShardKey handles @@shardKey on a model.
func HandleModelShardKey(
	modelID ModelId,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	attr := ctx.CurrentAttribute()
	fieldsExpr, _, err := ctx.VisitDefaultArg("fields")
	if err != nil {
		if diagErr, ok := err.(diagnostics.DatamodelError); ok {
			ctx.PushError(diagErr)
		} else {
			ctx.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.EmptySpan()))
		}
		return
	}

	resolvedFields, resolveErr := resolveFieldArrayWithoutArgs(fieldsExpr, attr.Span, modelID, ctx)
	if resolveErr == errFieldResolutionAlreadyDealtWith {
		return
	}
	if resolveErr != nil {
		// Errors already pushed
		return
	}

	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return
	}

	// Validate that all fields are required
	var fieldsThatAreNotRequired []string
	for _, sfid := range resolvedFields {
		if int(sfid) < len(ctx.types.ScalarFields) {
			sf := &ctx.types.ScalarFields[sfid]
			if astModel != nil && int(sf.FieldID) < len(astModel.Fields) {
				astField := &astModel.Fields[sf.FieldID]
				// Check if field is required (not optional and not array)
				if astField.FieldType.IsOptional() || astField.FieldType.IsArray() {
					fieldsThatAreNotRequired = append(fieldsThatAreNotRequired, astField.Name.Name)
				}
			}
		}
	}

	if len(fieldsThatAreNotRequired) > 0 && !modelAttrs.IsIgnored {
		ctx.PushError(diagnostics.NewModelValidationError(
			"The shard key definition refers to the optional fields: "+formatFieldNames(fieldsThatAreNotRequired)+". Shard key definitions must reference only required fields.",
			"model",
			astModel.Name.Name,
			attr.Span,
		))
	}

	if modelAttrs.ShardKey != nil {
		ctx.PushError(diagnostics.NewModelValidationError(
			"Each model must have at most one shard key. You can't have `@shardKey` and `@@shardKey` at the same time.",
			"model",
			astModel.Name.Name,
			astModel.Span(),
		))
		return
	}

	attrID := ctx.CurrentAttributeID()
	modelAttrs.ShardKey = &ShardKeyAttribute{
		SourceAttribute: uint32(attrID.Index), // Simplified
		Fields:          resolvedFields,
		SourceField:     nil, // @@shardKey doesn't have a source field
	}
}

// HandleFieldShardKey handles @shardKey on a scalar field.
func HandleFieldShardKey(
	astModel *ast.Model,
	sfid ScalarFieldId,
	fieldID uint32,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	if modelAttrs.ShardKey != nil {
		ctx.PushError(diagnostics.NewModelValidationError(
			"At most one field must be marked as the shard key with the `@shardKey` attribute.",
			"model",
			astModel.Name.Name,
			astModel.Span(),
		))
		return
	}

	attrID := ctx.CurrentAttributeID()
	modelAttrs.ShardKey = &ShardKeyAttribute{
		SourceAttribute: uint32(attrID.Index), // Simplified
		Fields:          []ScalarFieldId{sfid},
		SourceField:     &fieldID,
	}
}

// ValidateShardKeyFieldArities validates that @shardKey fields are required.
// This must be called after all attributes are resolved.
func ValidateShardKeyFieldArities(
	modelID ModelId,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	if modelAttrs.IsIgnored {
		return
	}

	if modelAttrs.ShardKey == nil {
		return
	}

	sk := modelAttrs.ShardKey
	if sk.SourceField == nil {
		return // @@shardKey doesn't have a source field to validate
	}

	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return
	}

	if int(*sk.SourceField) >= len(astModel.Fields) {
		return
	}

	astField := &astModel.Fields[*sk.SourceField]
	if astField.FieldType.IsOptional() || astField.FieldType.IsArray() {
		// TODO: Get proper attribute span from AttributeId
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Fields that are marked as shard keys must be required.",
			"@shardKey",
			astField.Name.Span(),
		))
	}
}
