// Package parserdatabase provides @updatedAt attribute handling.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// HandleUpdatedAt handles @updatedAt on a scalar field.
func HandleUpdatedAt(
	sfid ScalarFieldId,
	fieldType ScalarFieldType,
	astField *ast.Field,
	ctx *Context,
) {
	// Validate that the field type is DateTime
	if fieldType.BuiltInScalar == nil || *fieldType.BuiltInScalar != ScalarTypeDateTime {
		ctx.PushAttributeValidationError("Fields that are marked with @updatedAt must be of type DateTime.")
	}

	// Validate that the field is not a list
	if astField.FieldType.IsArray() {
		ctx.PushAttributeValidationError("Fields that are marked with @updatedAt cannot be lists.")
	}

	// Set the field as updated_at
	if int(sfid) < len(ctx.types.ScalarFields) {
		ctx.types.ScalarFields[sfid].IsUpdatedAt = true
	}
}
