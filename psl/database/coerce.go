// Package parserdatabase provides expression coercion functionality.
package database

import (
	"fmt"
	"strconv"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// CoerceString coerces an expression to a string.
// Returns the string value and true if successful, empty string and false otherwise.
func CoerceString(expr v2ast.Expression, diags *diagnostics.Diagnostics) (string, bool) {
	if strVal, ok := expr.(*v2ast.StringValue); ok {
		return strVal.GetValue(), true
	}

	// Try to get span for error
	var span diagnostics.Span
	pos := expr.Span()
	span = diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)

	diags.PushError(diagnostics.NewValueParserError(
		"string",
		fmt.Sprintf("%T", expr),
		span,
	))
	return "", false
}

// CoerceConstant coerces an expression to a constant string (identifier).
// Returns the constant value and true if successful, empty string and false otherwise.
func CoerceConstant(expr v2ast.Expression, diags *diagnostics.Diagnostics) (string, bool) {
	// In Prisma, constants are typically identifiers or constant values
	if constVal, ok := expr.(*v2ast.ConstantValue); ok {
		return constVal.Value, true
	}

	// Try string value as fallback
	if strVal, ok := expr.(*v2ast.StringValue); ok {
		return strVal.GetValue(), true
	}

	pos := expr.Span()
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)

	diags.PushError(diagnostics.NewValueParserError(
		"constant",
		fmt.Sprintf("%T", expr),
		span,
	))
	return "", false
}

// CoerceInteger coerces an expression to an integer.
// Returns the integer value and true if successful, 0 and false otherwise.
func CoerceInteger(expr v2ast.Expression, diags *diagnostics.Diagnostics) (int64, bool) {
	// Try numeric value
	if numVal, ok := expr.(*v2ast.NumericValue); ok {
		if val, err := strconv.ParseInt(numVal.Value, 10, 64); err == nil {
			return val, true
		}
	}

	// Try parsing from string value
	if strVal, ok := expr.(*v2ast.StringValue); ok {
		if val, err := strconv.ParseInt(strVal.GetValue(), 10, 64); err == nil {
			return val, true
		}
	}

	pos := expr.Span()
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)

	diags.PushError(diagnostics.NewValueParserError(
		"numeric",
		fmt.Sprintf("%T", expr),
		span,
	))
	return 0, false
}

// CoerceBoolean coerces an expression to a boolean.
// Returns the boolean value and true if successful, false and false otherwise.
func CoerceBoolean(expr v2ast.Expression, diags *diagnostics.Diagnostics) (bool, bool) {
	// Try constant value (true/false)
	if constVal, ok := expr.(*v2ast.ConstantValue); ok {
		if val, ok := constVal.AsBooleanValue(); ok {
			return val, true
		}
	}

	// Try parsing from string value
	if strVal, ok := expr.(*v2ast.StringValue); ok {
		if val, err := strconv.ParseBool(strVal.GetValue()); err == nil {
			return val, true
		}
	}

	pos := expr.Span()
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)

	diags.PushError(diagnostics.NewValueParserError(
		"boolean",
		fmt.Sprintf("%T", expr),
		span,
	))
	return false, false
}

// CoerceFloat coerces an expression to a float.
// Returns the float value and true if successful, 0 and false otherwise.
func CoerceFloat(expr v2ast.Expression, diags *diagnostics.Diagnostics) (float64, bool) {
	// Try numeric value
	if numVal, ok := expr.(*v2ast.NumericValue); ok {
		if val, err := strconv.ParseFloat(numVal.Value, 64); err == nil {
			return val, true
		}
	}

	// Try parsing from string value
	if strVal, ok := expr.(*v2ast.StringValue); ok {
		if val, err := strconv.ParseFloat(strVal.GetValue(), 64); err == nil {
			return val, true
		}
	}

	pos := expr.Span()
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)

	diags.PushError(diagnostics.NewValueParserError(
		"float",
		fmt.Sprintf("%T", expr),
		span,
	))
	return 0, false
}

// CoerceFunction coerces an expression to a function call.
// Returns the function name, arguments (as expressions), span and true if successful.
func CoerceFunction(expr v2ast.Expression, diags *diagnostics.Diagnostics) (string, []v2ast.Expression, diagnostics.Span, bool) {
	if funcCall, ok := expr.(*v2ast.FunctionCall); ok {
		var args []v2ast.Expression
		if funcCall.Arguments != nil && len(funcCall.Arguments.Arguments) > 0 {
			args = make([]v2ast.Expression, len(funcCall.Arguments.Arguments))
			for i, arg := range funcCall.Arguments.Arguments {
				if arg != nil {
					args[i] = arg.Value
				}
			}
		}
		pos := funcCall.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(funcCall.Name), diagnostics.FileIDZero)
		return funcCall.Name, args, span, true
	}

	pos := expr.Span()
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)

	diags.PushError(diagnostics.NewValueParserError(
		"function",
		fmt.Sprintf("%T", expr),
		span,
	))
	return "", nil, diagnostics.EmptySpan(), false
}
