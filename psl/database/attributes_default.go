// Package parserdatabase provides @default attribute handling.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

const (
	FN_DBGENERATED = "dbgenerated"
)

// HandleModelFieldDefault handles @default on a model scalar field.
func HandleModelFieldDefault(
	sfid ScalarFieldId,
	modelID ModelId,
	fieldID uint32,
	fieldType ScalarFieldType,
	ctx *Context,
) {
	// Get the default value argument
	valueExpr, argIdx, err := ctx.VisitDefaultArgWithIdx("value")
	if err != nil {
		if diagErr, ok := err.(diagnostics.DatamodelError); ok {
			ctx.PushError(diagErr)
		} else {
			ctx.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.EmptySpan()))
		}
		return
	}

	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return
	}

	mappedName := defaultAttributeMappedName(ctx)
	defaultAttributeID := ctx.CurrentAttributeID()

	accept := func() {
		defaultValue := DefaultAttribute{
			ArgumentIdx:      argIdx,
			MappedName:       mappedName,
			DefaultAttribute: uint32(defaultAttributeID.Index), // Simplified
		}

		if int(sfid) < len(ctx.types.ScalarFields) {
			ctx.types.ScalarFields[sfid].Default = &defaultValue
		}
	}

	// @default(dbgenerated(...)) is always valid.
	if funcCall, ok := valueExpr.(ast.FunctionCall); ok {
		if funcCall.Name.Name == FN_DBGENERATED {
			validateDbGeneratedArgs(funcCall.Arguments, accept, ctx)
			return
		}
	}

	// Validate based on field type
	if fieldType.EnumID != nil {
		// TODO: Validate enum default
		accept()
	} else if fieldType.CompositeTypeID != nil {
		// TODO: Validate composite type default
		accept()
	} else if fieldType.BuiltInScalar != nil {
		// Validate built-in scalar default based on scalar type
		validateModelBuiltinScalarTypeDefault(sfid, valueExpr, mappedName, accept, modelID, fieldID, ctx)
	} else if fieldType.ExtensionID != nil {
		ctx.PushAttributeValidationError("Only @default(dbgenerated(\"...\")) can be used for extension types.")
	} else if fieldType.Unsupported != nil {
		ctx.PushAttributeValidationError("Only @default(dbgenerated(\"...\")) can be used for Unsupported types.")
	}
}

// HandleCompositeFieldDefault handles @default on a composite type field.
func HandleCompositeFieldDefault(
	ctID CompositeTypeId,
	fieldID uint32,
	fieldType ScalarFieldType,
	ctx *Context,
) {
	// Get the default value argument
	valueExpr, argIdx, err := ctx.VisitDefaultArgWithIdx("value")
	if err != nil {
		if diagErr, ok := err.(diagnostics.DatamodelError); ok {
			ctx.PushError(diagErr)
		} else {
			ctx.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.EmptySpan()))
		}
		return
	}

	// Check for @map argument (not allowed on composite type fields)
	if ctx.VisitOptionalArg("map") != nil {
		ctx.PushAttributeValidationError("The `map` argument is not allowed on a composite type field.")
	}

	defaultAttributeID := ctx.CurrentAttributeID()

	accept := func() {
		defaultValue := DefaultAttribute{
			ArgumentIdx:      argIdx,
			MappedName:       nil,                              // Composite type fields don't have mapped names
			DefaultAttribute: uint32(defaultAttributeID.Index), // Simplified
		}

		key := CompositeTypeFieldKeyByID{
			CompositeTypeID: ctID,
			FieldID:         fieldID,
		}
		if ctf, exists := ctx.types.CompositeTypeFields[key]; exists {
			ctf.Default = &defaultValue
			ctx.types.CompositeTypeFields[key] = ctf
		}
	}

	// @default(dbgenerated(...)) is never valid on a composite type's fields.
	if funcCall, ok := valueExpr.(ast.FunctionCall); ok {
		if funcCall.Name.Name == FN_DBGENERATED {
			ctx.PushAttributeValidationError("Fields of composite types cannot have `dbgenerated()` as default.")
			return
		}
	}

	// Validate based on field type
	if fieldType.EnumID != nil {
		// TODO: Validate enum default
		accept()
	} else if fieldType.CompositeTypeID != nil {
		// TODO: Validate composite type default
		accept()
	} else if fieldType.BuiltInScalar != nil {
		// TODO: Validate built-in scalar default
		accept()
	} else if fieldType.ExtensionID != nil {
		ctx.PushAttributeValidationError("Composite field with extension type cannot have default values.")
	} else if fieldType.Unsupported != nil {
		ctx.PushAttributeValidationError("Composite field of type `Unsupported` cannot have default values.")
	}
}

// defaultAttributeMappedName extracts the mapped name from a @map argument on @default.
func defaultAttributeMappedName(ctx *Context) *StringId {
	expr := ctx.VisitOptionalArg("map")
	if expr == nil {
		return nil
	}

	name, ok := CoerceString(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	if name == "" {
		ctx.PushAttributeValidationError("The `map` argument cannot be an empty string.")
		return nil
	}

	nameID := ctx.interner.Intern(name)
	return &nameID
}

// validateDbGeneratedArgs validates the arguments to dbgenerated().
func validateDbGeneratedArgs(args []ast.Expression, accept func(), ctx *Context) {
	// dbgenerated() can have 0 or 1 string argument
	if len(args) == 0 {
		accept()
		return
	}

	if len(args) > 1 {
		ctx.PushAttributeValidationError("The `dbgenerated()` function accepts at most one argument.")
		return
	}

	// Validate that the argument is a string
	if _, ok := CoerceString(args[0], ctx.diagnostics); ok {
		accept()
	} else {
		ctx.PushAttributeValidationError("The `dbgenerated()` function accepts only a string argument.")
	}
}

// validateModelBuiltinScalarTypeDefault validates a default value for a built-in scalar type.
// This is a simplified version - full implementation would validate specific scalar types.
func validateModelBuiltinScalarTypeDefault(
	sfid ScalarFieldId,
	value ast.Expression,
	mappedName *StringId,
	accept func(),
	modelID ModelId,
	fieldID uint32,
	ctx *Context,
) {
	// TODO: Implement full validation based on scalar type
	// For now, just accept the default
	accept()
}
