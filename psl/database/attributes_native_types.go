// Package parserdatabase provides native type attribute handling (e.g., @db.Text).
package database

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// HandleModelFieldNativeType handles native type attributes on model scalar fields (e.g., @db.Text).
func HandleModelFieldNativeType(
	sfid ScalarFieldId,
	datasourceName StringId,
	typeName StringId,
	attr *v2ast.Attribute,
	ctx *Context,
) {
	// Extract arguments as strings
	var args []string
	if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
		args = make([]string, len(attr.Arguments.Arguments))
		for i, arg := range attr.Arguments.Arguments {
			// Convert expression to string representation
			args[i] = expressionToString(arg.Value)
		}
	}

	pos := attr.Pos
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
	nativeType := &NativeTypeInfo{
		Scope:     datasourceName,
		TypeName:  typeName,
		Arguments: args,
		Span:      span,
	}

	if int(sfid) < len(ctx.types.ScalarFields) {
		ctx.types.ScalarFields[sfid].NativeType = nativeType
	}
}

// HandleCompositeTypeFieldNativeType handles native type attributes on composite type fields.
func HandleCompositeTypeFieldNativeType(
	ctID CompositeTypeId,
	fieldID uint32,
	datasourceName StringId,
	typeName StringId,
	attr *v2ast.Attribute,
	ctx *Context,
) {
	// Extract arguments as strings
	var args []string
	if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
		args = make([]string, len(attr.Arguments.Arguments))
		for i, arg := range attr.Arguments.Arguments {
			args[i] = expressionToString(arg.Value)
		}
	}

	pos := attr.Pos
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
	nativeType := &NativeTypeInfo{
		Scope:     datasourceName,
		TypeName:  typeName,
		Arguments: args,
		Span:      span,
	}

	key := CompositeTypeFieldKeyByID{
		CompositeTypeID: ctID,
		FieldID:         fieldID,
	}
	if ctf, exists := ctx.types.CompositeTypeFields[key]; exists {
		ctf.NativeType = nativeType
		ctx.types.CompositeTypeFields[key] = ctf
	}
}

// expressionToString converts an expression to its string representation.
// This is a simplified version - full implementation would properly format all expression types.
func expressionToString(expr v2ast.Expression) string {
	switch e := expr.(type) {
	case *v2ast.StringValue:
		return e.GetValue()
	case *v2ast.NumericValue:
		return e.Value
	case *v2ast.ConstantValue:
		return e.Value
	case *v2ast.ArrayExpression:
		parts := make([]string, len(e.Elements))
		for i, elem := range e.Elements {
			parts[i] = expressionToString(elem)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *v2ast.FunctionCall:
		var parts []string
		if e.Arguments != nil && len(e.Arguments.Arguments) > 0 {
			parts = make([]string, len(e.Arguments.Arguments))
			for i, arg := range e.Arguments.Arguments {
				parts[i] = expressionToString(arg.Value)
			}
		}
		return e.Name + "(" + strings.Join(parts, ", ") + ")"
	default:
		// Unknown expression type - return empty string
		return ""
	}
}
