// Package pslcore provides default value validation functions.
package validation

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateDefaultValueType validates that default values match their field types.
func validateDefaultValueType(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	defaultValue := field.DefaultValue()
	if defaultValue == nil {
		return
	}

	value := defaultValue.Value()
	if value == nil {
		return
	}

	scalarType := field.ScalarType()
	fieldType := field.ScalarFieldType()

	// Check if this is an enum type
	if fieldType.EnumID != nil {
		validateEnumDefaultValue(field, value, ctx)
		return
	}

	if scalarType == nil {
		return
	}

	// Type-specific validations
	switch *scalarType {
	case database.ScalarTypeInt, database.ScalarTypeBigInt:
		validateIntDefaultValue(value, ctx)
	case database.ScalarTypeFloat, database.ScalarTypeDecimal:
		validateFloatDefaultValue(value, ctx)
	case database.ScalarTypeBoolean:
		validateBooleanDefaultValue(value, ctx)
	case database.ScalarTypeString:
		validateStringDefaultValue(value, ctx)
	case database.ScalarTypeDateTime:
		validateDateTimeDefaultValue(value, ctx)
	case database.ScalarTypeJson:
		validateJsonDefaultValue(value, ctx)
	case database.ScalarTypeBytes:
		validateBytesDefaultValue(value, ctx)
	}
}

// validateEnumDefaultValue validates that enum default values are valid enum values.
func validateEnumDefaultValue(field *database.ScalarFieldWalker, value v2ast.Expression, ctx *ValidationContext) {
	fieldType := field.ScalarFieldType()
	if fieldType.EnumID == nil {
		return
	}

	enum := ctx.Db.WalkEnum(*fieldType.EnumID)
	if enum == nil {
		return
	}

	// Get the enum value name from the default value
	var enumValueName string
	if strLit, ok := value.AsStringValue(); ok {
		enumValueName = strLit.Value
	} else if constVal, ok := value.AsConstantValue(); ok {
		enumValueName = constVal.Value
	} else {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, field.Model().FileID())
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Enum default value must be a valid enum value name.",
			"@default",
			span,
		))
		return
	}

	// Check if the value exists in the enum
	values := enum.Values()
	found := false
	for _, enumValue := range values {
		if enumValue.Name() == enumValueName {
			found = true
			break
		}
	}

	if !found {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(enumValueName), field.Model().FileID())
		ctx.PushError(diagnostics.NewAttributeValidationError(
			fmt.Sprintf("Enum value '%s' is not a valid value for enum '%s'.", enumValueName, enum.Name()),
			"@default",
			span,
		))
	}
}

// validateIntDefaultValue validates integer default values.
func validateIntDefaultValue(value v2ast.Expression, ctx *ValidationContext) {
	if _, ok := value.AsNumericValue(); ok {
		// Valid integer literal - check if it can be parsed as int64
		// The parser should have already validated this, but we can add extra validation if needed
		return
	} else if funcCall, ok := value.AsFunction(); ok {
		if funcCall.Name == "autoincrement" || funcCall.Name == "cuid" || funcCall.Name == "uuid" {
			// Valid function calls
			return
		}
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(funcCall.Name), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Default value must be an integer or a valid function call.",
			"@default",
			span,
		))
	} else if strLit, ok := value.AsStringValue(); ok {
		// Try to parse as integer
		// This handles cases where a string is provided but should be an integer
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(strLit.Value), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewAttributeValidationError(
			fmt.Sprintf("Parse error: \"%s\" is not a valid integer.", strLit.Value),
			"@default",
			span,
		))
	} else {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Default value must be an integer or a function call.",
			"@default",
			span,
		))
	}
}

// validateFloatDefaultValue validates float default values.
func validateFloatDefaultValue(value v2ast.Expression, ctx *ValidationContext) {
	if _, ok := value.AsNumericValue(); ok {
		// Valid float literal
		// Integers can be used as floats
		return
	} else {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewValidationError(
			"Default value must be a number.",
			span,
		))
	}
}

// validateBooleanDefaultValue validates boolean default values.
func validateBooleanDefaultValue(value v2ast.Expression, ctx *ValidationContext) {
	// Boolean values are often Constants in the parser if true/false are treated as keywords/constants
	// But V2 AST might not have specific BooleanValue type from parser, likely ConstantValue "true"/"false"
	// Check AsConstantValue
	if cv, ok := value.AsConstantValue(); ok {
		if cv.Value == "true" || cv.Value == "false" {
			return
		}
	}

	pos := value.Span()
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
	ctx.PushError(diagnostics.NewValidationError(
		"Default value must be a boolean.",
		span,
	))
}

// validateStringDefaultValue validates string default values.
func validateStringDefaultValue(value v2ast.Expression, ctx *ValidationContext) {
	if _, ok := value.AsStringValue(); ok {
		// Valid string literal
		return
	} else if _, ok := value.AsFunction(); ok {
		// Function calls like cuid(), uuid(), now() are valid
		return
	} else {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewValidationError(
			"Default value must be a string or a function call.",
			span,
		))
	}
}

// validateDateTimeDefaultValue validates DateTime default values.
func validateDateTimeDefaultValue(value v2ast.Expression, ctx *ValidationContext) {
	if strLit, ok := value.AsStringValue(); ok {
		// Validate RFC3339 format
		if err := validateRFC3339DateTime(strLit.Value); err != nil {
			pos := value.Span()
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(strLit.Value), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("Parse error: \"%s\" is not a valid rfc3339 datetime string. (%s)", strLit.Value, err.Error()),
				"@default",
				span,
			))
		}
		return
	} else if funcCall, ok := value.AsFunction(); ok {
		if funcCall.Name == "now" {
			return
		}
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(funcCall.Name), diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("DateTime default value must be a valid RFC3339 string or now()."),
			span,
		))
	} else {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewValidationError(
			"DateTime default value must be a string or now().",
			span,
		))
	}
}

// validateRFC3339DateTime validates that a string is a valid RFC3339 datetime.
func validateRFC3339DateTime(value string) error {
	// RFC3339 format: 2006-01-02T15:04:05Z07:00
	// Try parsing with time.RFC3339
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		// Try parsing with time.RFC3339Nano for nanosecond precision
		_, err = time.Parse(time.RFC3339Nano, value)
	}
	return err
}

// validateJsonDefaultValue validates JSON default values.
func validateJsonDefaultValue(value v2ast.Expression, ctx *ValidationContext) {
	if strLit, ok := value.AsStringValue(); ok {
		// Validate JSON format
		if err := validateJSONString(strLit.Value); err != nil {
			pos := value.Span()
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(strLit.Value), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("Parse error: \"%s\" is not a valid JSON string. (%s)", strLit.Value, err.Error()),
				"@default",
				span,
			))
		}
		return
	} else {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewValidationError(
			"JSON default value must be a valid JSON string.",
			span,
		))
	}
}

// validateJSONString validates that a string is valid JSON.
func validateJSONString(value string) error {
	var jsonValue interface{}
	return json.Unmarshal([]byte(value), &jsonValue)
}

// validateBytesDefaultValue validates Bytes default values.
func validateBytesDefaultValue(value v2ast.Expression, ctx *ValidationContext) {
	if strLit, ok := value.AsStringValue(); ok {
		// Validate base64 format
		if err := validateBase64String(strLit.Value); err != nil {
			pos := value.Span()
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(strLit.Value), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("Parse error: \"%s\" is not a valid base64 string. (%s)", strLit.Value, err.Error()),
				"@default",
				span,
			))
		}
		return
	} else {
		pos := value.Span()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
		ctx.PushError(diagnostics.NewValidationError(
			"Bytes default value must be a valid base64 string.",
			span,
		))
	}
}

// validateBase64String validates that a string is valid base64.
func validateBase64String(value string) error {
	_, err := base64.StdEncoding.DecodeString(value)
	return err
}

// validateNamedDefaultValues validates that named default values are supported.
func validateNamedDefaultValues(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	defaultValue := field.DefaultValue()
	if defaultValue == nil {
		return
	}

	if mappedName := defaultValue.MappedName(); mappedName != nil {
		if !ctx.HasCapability(ConnectorCapabilityNamedDefaultValues) {
			value := defaultValue.Value()
			if value != nil {
				pos := value.Span()
				span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"You defined a database name for the default value of a field on the model. This is not supported by the provider.",
					"@default",
					span,
				))
			}
		}
	}
}

// validateAutoParam validates that auto() function is only used with MongoDB.
func validateAutoParam(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	defaultValue := field.DefaultValue()
	if defaultValue == nil {
		return
	}

	value := defaultValue.Value()
	if value == nil {
		return
	}

	// Check if it's a function call to auto()
	if funcCall, ok := value.AsFunction(); ok {
		if funcCall.Name == "auto" {
			if !ctx.HasCapability(ConnectorCapabilityDefaultValueAuto) {
				pos := funcCall.Span()
				span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(funcCall.Name), diagnostics.FileIDZero)
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"The current connector does not support the `auto()` function.",
					"@default",
					span,
				))
			}
		}
	}
}
