// Package parserdatabase provides @map attribute handling.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// VisitMapAttribute visits a @map attribute and returns the mapped name.
func VisitMapAttribute(ctx *Context) *StringId {
	expr, _, err := ctx.VisitDefaultArg("name")
	if err != nil {
		// Error already pushed
		return nil
	}

	name, ok := CoerceString(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	if name == "" {
		ctx.PushAttributeValidationError("The `name` argument cannot be an empty string.")
		return nil
	}

	nameID := ctx.interner.Intern(name)
	return &nameID
}

// HandleModelMap handles @@map on a model.
func HandleModelMap(modelAttrs *ModelAttributes, ctx *Context) {
	mappedName := VisitMapAttribute(ctx)
	if mappedName == nil {
		return
	}

	modelAttrs.MappedName = mappedName
}

// HandleScalarFieldMap handles @map on a scalar field.
func HandleScalarFieldMap(
	sfid ScalarFieldId,
	astModel *v2ast.Model,
	astField *v2ast.Field,
	modelID ModelId,
	fieldID uint32,
	ctx *Context,
) {
	mappedName := VisitMapAttribute(ctx)
	if mappedName == nil {
		return
	}

	// Set mapped name on the scalar field
	if int(sfid) < len(ctx.types.ScalarFields) {
		ctx.types.ScalarFields[sfid].MappedName = mappedName
	}

	// Check for duplicate mapped names
	key := ModelFieldKeyByID{
		ModelID: modelID,
		NameID:  *mappedName,
	}
	if existingFieldID, exists := ctx.mappedModelScalarFieldNames[key]; exists {
		fieldName := astField.GetName()
		pos := astField.Pos
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(fieldName), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewDuplicateFieldError(
			astModel.GetName(),
			fieldName,
			"model",
			span,
		))
		_ = existingFieldID
		return
	}

	ctx.mappedModelScalarFieldNames[key] = fieldID

	// Check for conflicts with regular field names
	if dupFieldID := ctx.FindModelField(modelID, ctx.interner.Get(*mappedName)); dupFieldID != nil {
		// Check if the other field has a different mapped name
		scalarFields := ctx.types.RangeModelScalarFields(modelID)
		for _, entry := range scalarFields {
			if entry.Field.FieldID == *dupFieldID && entry.Field.FieldID != fieldID {
				if entry.Field.MappedName != nil && *entry.Field.MappedName != *mappedName {
					return // Different mapped names, no conflict
				}
				if entry.Field.MappedName == nil {
					// Conflict with regular field name
					fieldName := astField.GetName()
					pos := astField.Pos
					span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(fieldName), diagnostics.FileIDZero)
					ctx.PushError(diagnostics.NewDuplicateFieldError(
						astModel.GetName(),
						fieldName,
						"model",
						span,
					))
				}
				break
			}
		}
	}
}

// HandleCompositeTypeFieldMap handles @map on a composite type field.
func HandleCompositeTypeFieldMap(
	ct *v2ast.CompositeType,
	astField *v2ast.Field,
	ctID CompositeTypeId,
	fieldID uint32,
	ctx *Context,
) {
	mappedName := VisitMapAttribute(ctx)
	if mappedName == nil {
		return
	}

	// Set mapped name on the composite type field
	key := CompositeTypeFieldKeyByID{
		CompositeTypeID: ctID,
		FieldID:         fieldID,
	}
	if ctf, exists := ctx.types.CompositeTypeFields[key]; exists {
		ctf.MappedName = mappedName
		ctx.types.CompositeTypeFields[key] = ctf
	}

	// Check for duplicate mapped names
	// Use a different approach - check if any field already has this mapped name
	for existingKey, existingFieldID := range ctx.mappedCompositeTypeNames {
		if existingKey.CompositeTypeID == ctID && existingFieldID == fieldID {
			// This field already has a mapped name
			continue
		}
		// Check if the mapped name matches by looking up the field
		existingFieldKey := CompositeTypeFieldKeyByID{
			CompositeTypeID: ctID,
			FieldID:         existingFieldID,
		}
		if existingField, exists := ctx.types.CompositeTypeFields[existingFieldKey]; exists {
			if existingField.MappedName != nil && *existingField.MappedName == *mappedName {
				fieldName := astField.GetName()
				pos := astField.Pos
				span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(fieldName), diagnostics.FileIDZero)
				ctx.PushError(diagnostics.NewCompositeTypeDuplicateFieldError(
					ct.GetName(),
					ctx.interner.Get(*mappedName),
					span,
				))
				return
			}
		}
	}

	// Store the mapped name
	ctKey := CompositeTypeFieldKeyByID{
		CompositeTypeID: ctID,
		FieldID:         fieldID,
	}
	ctx.mappedCompositeTypeNames[ctKey] = fieldID

	// Check for conflicts with regular field names
	if dupFieldID := ctx.FindCompositeTypeField(ctID, ctx.interner.Get(*mappedName)); dupFieldID != nil {
		otherKey := CompositeTypeFieldKeyByID{
			CompositeTypeID: ctID,
			FieldID:         *dupFieldID,
		}
		if otherField, exists := ctx.types.CompositeTypeFields[otherKey]; exists {
			if otherField.MappedName != nil {
				return // Other field has mapped name, no conflict
			}
			// Conflict with regular field name
			fieldName := astField.GetName()
			pos := astField.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(fieldName), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewCompositeTypeDuplicateFieldError(
				ct.GetName(),
				fieldName,
				span,
			))
		}
	}
}
