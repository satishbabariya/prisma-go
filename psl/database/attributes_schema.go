// Package parserdatabase provides @@schema attribute handling.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// HandleModelSchema handles @@schema on a model.
func HandleModelSchema(modelAttrs *ModelAttributes, ctx *Context) {
	schemaInfo := visitSchemaAttribute(ctx)
	modelAttrs.Schema = schemaInfo
}

// HandleEnumSchema handles @@schema on an enum.
func HandleEnumSchema(enumAttrs *EnumAttributes, ctx *Context) {
	schemaInfo := visitSchemaAttribute(ctx)
	if schemaInfo != nil {
		enumAttrs.Schema = schemaInfo
	}
}

// visitSchemaAttribute visits a @@schema attribute and returns the schema name and span.
func visitSchemaAttribute(ctx *Context) *SchemaInfo {
	arg, _, err := ctx.VisitDefaultArg("map")
	if err != nil {
		if diagErr, ok := err.(diagnostics.DatamodelError); ok {
			ctx.PushError(diagErr)
		} else {
			ctx.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.EmptySpan()))
		}
		return nil
	}

	name, ok := CoerceString(arg, ctx.diagnostics)
	if !ok {
		return nil
	}

	nameID := ctx.interner.Intern(name)
	var span diagnostics.Span
	if spanExpr, ok := arg.(interface{ Span() diagnostics.Span }); ok {
		span = spanExpr.Span()
	} else {
		span = diagnostics.EmptySpan()
	}

	return &SchemaInfo{
		Name: nameID,
		Span: span,
	}
}
