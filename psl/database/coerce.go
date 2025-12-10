// Package parserdatabase provides expression coercion functionality.
package database

import (
	"fmt"
	"strconv"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// CoerceString coerces an expression to a string.
// Returns the string value and true if successful, empty string and false otherwise.
func CoerceString(expr ast.Expression, diags *diagnostics.Diagnostics) (string, bool) {
	if strLit, ok := expr.(ast.StringLiteral); ok {
		return strLit.Value, true
	}

	// Try to get span for error
	var span diagnostics.Span
	if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
		span = spanExpr.Span()
	} else {
		span = diagnostics.EmptySpan()
	}

	diags.PushError(diagnostics.NewValueParserError(
		"string",
		fmt.Sprintf("%T", expr),
		span,
	))
	return "", false
}

// CoerceConstant coerces an expression to a constant string (identifier).
// Returns the constant value and true if successful, empty string and false otherwise.
func CoerceConstant(expr ast.Expression, diags *diagnostics.Diagnostics) (string, bool) {
	// In Prisma, constants are typically identifiers
	// For now, we'll check if it's a string literal that looks like an identifier
	if strLit, ok := expr.(ast.StringLiteral); ok {
		return strLit.Value, true
	}

	// TODO: Handle actual identifier expressions when they're added to schemaast

	var span diagnostics.Span
	if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
		span = spanExpr.Span()
	} else {
		span = diagnostics.EmptySpan()
	}

	diags.PushError(diagnostics.NewValueParserError(
		"constant",
		fmt.Sprintf("%T", expr),
		span,
	))
	return "", false
}

// CoerceInteger coerces an expression to an integer.
// Returns the integer value and true if successful, 0 and false otherwise.
func CoerceInteger(expr ast.Expression, diags *diagnostics.Diagnostics) (int64, bool) {
	if intLit, ok := expr.(ast.IntLiteral); ok {
		return int64(intLit.Value), true
	}

	// Try parsing from string literal
	if strLit, ok := expr.(ast.StringLiteral); ok {
		if val, err := strconv.ParseInt(strLit.Value, 10, 64); err == nil {
			return val, true
		}
	}

	var span diagnostics.Span
	if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
		span = spanExpr.Span()
	} else {
		span = diagnostics.EmptySpan()
	}

	diags.PushError(diagnostics.NewValueParserError(
		"numeric",
		fmt.Sprintf("%T", expr),
		span,
	))
	return 0, false
}

// CoerceBoolean coerces an expression to a boolean.
// Returns the boolean value and true if successful, false and false otherwise.
func CoerceBoolean(expr ast.Expression, diags *diagnostics.Diagnostics) (bool, bool) {
	if boolLit, ok := expr.(ast.BooleanLiteral); ok {
		return boolLit.Value, true
	}

	// Try parsing from string literal
	if strLit, ok := expr.(ast.StringLiteral); ok {
		if val, err := strconv.ParseBool(strLit.Value); err == nil {
			return val, true
		}
	}

	var span diagnostics.Span
	if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
		span = spanExpr.Span()
	} else {
		span = diagnostics.EmptySpan()
	}

	diags.PushError(diagnostics.NewValueParserError(
		"boolean",
		fmt.Sprintf("%T", expr),
		span,
	))
	return false, false
}

// CoerceFloat coerces an expression to a float.
// Returns the float value and true if successful, 0 and false otherwise.
func CoerceFloat(expr ast.Expression, diags *diagnostics.Diagnostics) (float64, bool) {
	if floatLit, ok := expr.(ast.FloatLiteral); ok {
		return floatLit.Value, true
	}

	if intLit, ok := expr.(ast.IntLiteral); ok {
		return float64(intLit.Value), true
	}

	// Try parsing from string literal
	if strLit, ok := expr.(ast.StringLiteral); ok {
		if val, err := strconv.ParseFloat(strLit.Value, 64); err == nil {
			return val, true
		}
	}

	var span diagnostics.Span
	if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
		span = spanExpr.Span()
	} else {
		span = diagnostics.EmptySpan()
	}

	diags.PushError(diagnostics.NewValueParserError(
		"float",
		fmt.Sprintf("%T", expr),
		span,
	))
	return 0, false
}

// CoerceFunction coerces an expression to a function call.
// Returns the function name, arguments (as expressions), span and true if successful.
func CoerceFunction(expr ast.Expression, diags *diagnostics.Diagnostics) (string, []ast.Expression, diagnostics.Span, bool) {
	if funcCall, ok := expr.(ast.FunctionCall); ok {
		var span diagnostics.Span
		if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
			span = spanExpr.Span()
		} else {
			span = diagnostics.EmptySpan()
		}
		return funcCall.Name.Name, funcCall.Arguments, span, true
	}

	var span diagnostics.Span
	if spanExpr, ok := expr.(interface{ Span() diagnostics.Span }); ok {
		span = spanExpr.Span()
	} else {
		span = diagnostics.EmptySpan()
	}

	diags.PushError(diagnostics.NewValueParserError(
		"function",
		fmt.Sprintf("%T", expr),
		span,
	))
	return "", nil, diagnostics.EmptySpan(), false
}
