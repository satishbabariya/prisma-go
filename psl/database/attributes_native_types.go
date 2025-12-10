// Package parserdatabase provides native type attribute handling (e.g., @db.Text).
package database

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// HandleModelFieldNativeType handles native type attributes on model scalar fields (e.g., @db.Text).
func HandleModelFieldNativeType(
	sfid ScalarFieldId,
	datasourceName StringId,
	typeName StringId,
	attr *ast.Attribute,
	ctx *Context,
) {
	// Extract arguments as strings
	args := make([]string, len(attr.Arguments.Arguments))
	for i, arg := range attr.Arguments.Arguments {
		// Convert expression to string representation
		args[i] = expressionToString(arg.Value)
	}

	nativeType := &NativeTypeInfo{
		Scope:     datasourceName,
		TypeName:  typeName,
		Arguments: args,
		Span:      attr.Span,
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
	attr *ast.Attribute,
	ctx *Context,
) {
	// Extract arguments as strings
	args := make([]string, len(attr.Arguments.Arguments))
	for i, arg := range attr.Arguments.Arguments {
		args[i] = expressionToString(arg.Value)
	}

	nativeType := &NativeTypeInfo{
		Scope:     datasourceName,
		TypeName:  typeName,
		Arguments: args,
		Span:      attr.Span,
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
func expressionToString(expr ast.Expression) string {
	switch e := expr.(type) {
	case ast.StringLiteral:
		return e.Value
	case ast.IntLiteral:
		// Convert int to string - simplified
		return string(rune(e.Value))
	case ast.FloatLiteral:
		// Convert float to string - simplified
		return string(rune(e.Value))
	case ast.BooleanLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case ast.ArrayLiteral:
		parts := make([]string, len(e.Elements))
		for i, elem := range e.Elements {
			parts[i] = expressionToString(elem)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ast.FunctionCall:
		parts := make([]string, len(e.Arguments))
		for i, arg := range e.Arguments {
			parts[i] = expressionToString(arg)
		}
		return e.Name.Name + "(" + strings.Join(parts, ", ") + ")"
	default:
		// Fallback: try to get span for error reporting
		if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
			_ = spanExpr.Span()
		}
		return ""
	}
}
