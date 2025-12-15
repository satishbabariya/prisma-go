// Package parserdatabase provides attribute resolution functionality.
package database

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// ResolveAttributes resolves attributes for all models, enums, composite types, and fields.
// This is the third pass of validation after type resolution.
func ResolveAttributes(ctx *Context) {
	// First, resolve relation field attributes
	for _, entry := range ctx.types.IterRelationFields() {
		visitRelationFieldAttributes(entry.ID, ctx)
	}

	// Then resolve top-level attributes (models, enums, composite types)
	for _, entry := range ctx.IterTops() {
		switch t := entry.Top.(type) {
		case *v2ast.Model:
			modelID := ModelId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			resolveModelAttributes(modelID, ctx)
		case *v2ast.Enum:
			enumID := EnumId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			resolveEnumAttributes(enumID, t, ctx)
		case *v2ast.CompositeType:
			ctID := CompositeTypeId{
				FileID: entry.TopID.FileID,
				ID:     entry.TopID.ID,
			}
			resolveCompositeTypeAttributes(ctID, t, ctx)
		}
	}
}

// visitRelationFieldAttributes processes attributes on a relation field.
func visitRelationFieldAttributes(rfid RelationFieldId, ctx *Context) {
	rf := &ctx.types.RelationFields[rfid]

	// Get the model from AST
	astModel := getModelFromID(rf.ModelID, ctx)

	if astModel == nil {
		return
	}

	// Create attribute container for field
	// TODO: Properly encode field container in AttributeContainer
	container := AttributeContainer{
		FileID: rf.ModelID.FileID,
		ID:     rf.ModelID.ID, // This should encode model + field
	}
	ctx.VisitAttributes(container)

	// @ignore
	if ctx.VisitOptionalSingleAttr("ignore") {
		HandleRelationFieldIgnore(rfid, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @relation
	if ctx.VisitOptionalSingleAttr("relation") {
		HandleRelation(rf.ModelID, rfid, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @id (error - relation fields can't be @id)
	if ctx.VisitOptionalSingleAttr("id") {
		if int(rf.FieldID) < len(astModel.Fields) {
			astField := astModel.Fields[rf.FieldID]
			if astField != nil {
				ctx.PushAttributeValidationError(
					"The field `" + astField.GetName() + "` is a relation field and cannot be marked with `@id`. Only scalar fields can be declared as id.",
				)
			}
		}
		ctx.DiscardArguments()
	}

	// @default (error - relation fields can't have defaults)
	if ctx.VisitOptionalSingleAttr("default") {
		ctx.PushAttributeValidationError("Cannot set a default value on a relation field.")
		ctx.DiscardArguments()
	}

	// @map (error - relation fields can't have @map)
	if ctx.VisitOptionalSingleAttr("map") {
		ctx.PushAttributeValidationError("The attribute `@map` cannot be used on relation fields.")
		ctx.DiscardArguments()
	}

	ctx.ValidateVisitedAttributes()
}

// resolveModelAttributes processes attributes on a model.
func resolveModelAttributes(modelID ModelId, ctx *Context) {
	// Initialize model attributes
	modelAttrs := ModelAttributes{
		PrimaryKey: nil,
		IsIgnored:  false,
		AstIndexes: make([]IndexAttributeEntry, 0),
		MappedName: nil,
		Schema:     nil,
		ShardKey:   nil,
	}

	// Get the model from AST
	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return
	}

	// First resolve all scalar field attributes in isolation
	for _, entry := range ctx.types.RangeModelScalarFields(modelID) {
		visitScalarFieldAttributes(entry.ID, &modelAttrs, ctx)
	}

	// Then resolve model-level attributes
	// Create attribute container for model
	container := AttributeContainer{
		FileID: modelID.FileID,
		ID:     modelID.ID,
	}
	ctx.VisitAttributes(container)

	// @@ignore
	if ctx.VisitOptionalSingleAttr("ignore") {
		HandleModelIgnore(modelID, &modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@id
	if ctx.VisitOptionalSingleAttr("id") {
		HandleModelId(modelID, &modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@map
	if ctx.VisitOptionalSingleAttr("map") {
		HandleModelMap(&modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@schema
	if ctx.VisitOptionalSingleAttr("schema") {
		HandleModelSchema(&modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@index
	for ctx.VisitRepeatedAttr("index") {
		HandleModelIndex(modelID, &modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@unique
	for ctx.VisitRepeatedAttr("unique") {
		HandleModelUnique(modelID, &modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@fulltext
	for ctx.VisitRepeatedAttr("fulltext") {
		HandleModelFulltext(modelID, &modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@shardKey
	if ctx.VisitOptionalSingleAttr("shardKey") {
		HandleModelShardKey(modelID, &modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	ctx.ValidateVisitedAttributes()

	// Store the model attributes
	ctx.types.ModelAttributes[modelID] = modelAttrs
}

// visitScalarFieldAttributes processes attributes on a scalar field.
func visitScalarFieldAttributes(sfid ScalarFieldId, modelAttrs *ModelAttributes, ctx *Context) {
	sf := &ctx.types.ScalarFields[sfid]

	// Get the model and field from AST
	astModel := getModelFromID(sf.ModelID, ctx)
	if astModel == nil {
		return
	}

	if int(sf.FieldID) >= len(astModel.Fields) {
		return
	}
	astField := astModel.Fields[sf.FieldID]
	if astField == nil {
		return
	}

	// Create attribute container for field
	// TODO: Properly encode field container in AttributeContainer
	container := AttributeContainer{
		FileID: sf.ModelID.FileID,
		ID:     sf.ModelID.ID, // This should encode model + field
	}
	ctx.VisitAttributes(container)

	// @map
	if ctx.VisitOptionalSingleAttr("map") {
		HandleScalarFieldMap(sfid, astModel, astField, sf.ModelID, sf.FieldID, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @ignore
	if ctx.VisitOptionalSingleAttr("ignore") {
		HandleScalarFieldIgnore(sfid, sf.Type, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @id
	if ctx.VisitOptionalSingleAttr("id") {
		HandleFieldId(astModel, sfid, sf.FieldID, modelAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @default
	if ctx.VisitOptionalSingleAttr("default") {
		HandleModelFieldDefault(sfid, sf.ModelID, sf.FieldID, sf.Type, ctx)
		ctx.ValidateVisitedArguments()
	}

	// @updatedAt
	if ctx.VisitOptionalSingleAttr("updatedAt") {
		HandleUpdatedAt(sfid, sf.Type, astField, ctx)
		ctx.ValidateVisitedArguments()
	}

	// Native types (e.g., @db.Text)
	if sf.Type.BuiltInScalar != nil || sf.Type.ExtensionID != nil {
		if dsName, typeName, attrID, found := ctx.VisitDatasourceScoped(); found {
			// Get the attribute from AST
			attr := ctx.getAttribute(attrID)
			if attr != nil {
				HandleModelFieldNativeType(sfid, dsName, typeName, attr, ctx)
			}
		}
	}

	ctx.ValidateVisitedAttributes()
}

// resolveEnumAttributes processes attributes on an enum.
func resolveEnumAttributes(enumID EnumId, astEnum *v2ast.Enum, ctx *Context) {
	enumAttrs := EnumAttributes{
		MappedName:   nil,
		Schema:       nil,
		MappedValues: make(map[uint32]StringId),
	}

	// Process enum value attributes (@map on values)
	for i := range astEnum.Values {
		valueID := uint32(i)
		// Visit attributes for this enum value
		container := AttributeContainer{
			FileID: enumID.FileID,
			ID:     enumID.ID, // TODO: Properly encode enum + value in container
		}
		ctx.VisitAttributes(container)

		// Process @map attribute on enum value
		if ctx.VisitOptionalSingleAttr("map") {
			mappedName := VisitMapAttribute(ctx)
			if mappedName != nil {
				enumAttrs.MappedValues[valueID] = *mappedName
			}
			ctx.ValidateVisitedArguments()
		}

		ctx.ValidateVisitedAttributes()
	}

	// Process enum-level attributes
	container := AttributeContainer{
		FileID: enumID.FileID,
		ID:     enumID.ID,
	}
	ctx.VisitAttributes(container)

	// @@map
	if ctx.VisitOptionalSingleAttr("map") {
		enumAttrs.MappedName = VisitMapAttribute(ctx)
		ctx.ValidateVisitedArguments()
	}

	// @@schema
	if ctx.VisitOptionalSingleAttr("schema") {
		HandleEnumSchema(&enumAttrs, ctx)
		ctx.ValidateVisitedArguments()
	}

	ctx.ValidateVisitedAttributes()

	ctx.types.EnumAttributes[enumID] = enumAttrs
}

// resolveCompositeTypeAttributes processes attributes on a composite type.
func resolveCompositeTypeAttributes(ctID CompositeTypeId, astCT *v2ast.CompositeType, ctx *Context) {
	// Process each field in the composite type
	for i, field := range astCT.Fields {
		if field == nil {
			continue
		}
		fieldID := uint32(i)

		// Get the composite type field from types
		key := CompositeTypeFieldKeyByID{
			CompositeTypeID: ctID,
			FieldID:         fieldID,
		}
		ctf, exists := ctx.types.CompositeTypeFields[key]
		if !exists {
			continue
		}

		// Visit attributes for composite type field
		// TODO: Properly encode field container in AttributeContainer
		container := AttributeContainer{
			FileID: ctID.FileID,
			ID:     ctID.ID, // This should encode composite type + field
		}
		ctx.VisitAttributes(container)

		// Native types (e.g., @db.Text)
		if ctf.Type.BuiltInScalar != nil {
			if dsName, typeName, attrID, found := ctx.VisitDatasourceScoped(); found {
				attr := ctx.getAttribute(attrID)
				if attr != nil {
					HandleCompositeTypeFieldNativeType(ctID, fieldID, dsName, typeName, attr, ctx)
				}
			}
		}

		// @default
		if ctx.VisitOptionalSingleAttr("default") {
			HandleCompositeFieldDefault(ctID, fieldID, ctf.Type, ctx)
			ctx.ValidateVisitedArguments()
		}

		// @map
		if ctx.VisitOptionalSingleAttr("map") {
			HandleCompositeTypeFieldMap(astCT, astCT.Fields[i], ctID, fieldID, ctx)
			ctx.ValidateVisitedArguments()
		}

		ctx.ValidateVisitedAttributes()
	}
}
