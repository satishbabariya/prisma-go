// Package pslcore provides default value validation functions.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

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
func validateEnumDefaultValue(field *database.ScalarFieldWalker, value ast.Expression, ctx *ValidationContext) {
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
	switch v := value.(type) {
	case ast.StringLiteral:
		enumValueName = v.Value
	case ast.ConstantValue:
		enumValueName = v.Value
	default:
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Enum default value must be a valid enum value name.",
			"@default",
			value.Span(),
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
		ctx.PushError(diagnostics.NewAttributeValidationError(
			fmt.Sprintf("Enum value '%s' is not a valid value for enum '%s'.", enumValueName, enum.Name()),
			"@default",
			value.Span(),
		))
	}
}

// validateIntDefaultValue validates integer default values.
func validateIntDefaultValue(value ast.Expression, ctx *ValidationContext) {
	switch v := value.(type) {
	case ast.IntLiteral:
		// Valid integer literal - check if it can be parsed as int64
		// The parser should have already validated this, but we can add extra validation if needed
		return
	case ast.FunctionCall:
		if v.Name.Name == "autoincrement" || v.Name.Name == "cuid" || v.Name.Name == "uuid" {
			// Valid function calls
			return
		}
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Default value must be an integer or a valid function call.",
			"@default",
			value.Span(),
		))
	case ast.StringLiteral:
		// Try to parse as integer
		// This handles cases where a string is provided but should be an integer
		ctx.PushError(diagnostics.NewAttributeValidationError(
			fmt.Sprintf("Parse error: \"%s\" is not a valid integer.", v.Value),
			"@default",
			value.Span(),
		))
	default:
		ctx.PushError(diagnostics.NewAttributeValidationError(
			"Default value must be an integer or a function call.",
			"@default",
			value.Span(),
		))
	}
}

// validateFloatDefaultValue validates float default values.
func validateFloatDefaultValue(value ast.Expression, ctx *ValidationContext) {
	switch value.(type) {
	case ast.FloatLiteral:
		// Valid float literal
		return
	case ast.IntLiteral:
		// Integers can be used as floats
		return
	default:
		ctx.PushError(diagnostics.NewValidationError(
			"Default value must be a number.",
			value.Span(),
		))
	}
}

// validateBooleanDefaultValue validates boolean default values.
func validateBooleanDefaultValue(value ast.Expression, ctx *ValidationContext) {
	switch value.(type) {
	case ast.BooleanLiteral:
		// Valid boolean literal
		return
	default:
		ctx.PushError(diagnostics.NewValidationError(
			"Default value must be a boolean.",
			value.Span(),
		))
	}
}

// validateStringDefaultValue validates string default values.
func validateStringDefaultValue(value ast.Expression, ctx *ValidationContext) {
	switch value.(type) {
	case ast.StringLiteral:
		// Valid string literal
		return
	case ast.FunctionCall:
		// Function calls like cuid(), uuid(), now() are valid
		return
	default:
		ctx.PushError(diagnostics.NewValidationError(
			"Default value must be a string or a function call.",
			value.Span(),
		))
	}
}

// validateDateTimeDefaultValue validates DateTime default values.
func validateDateTimeDefaultValue(value ast.Expression, ctx *ValidationContext) {
	switch v := value.(type) {
	case ast.StringLiteral:
		// Validate RFC3339 format
		if err := validateRFC3339DateTime(v.Value); err != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("Parse error: \"%s\" is not a valid rfc3339 datetime string. (%s)", v.Value, err.Error()),
				"@default",
				v.Span(),
			))
		}
		return
	case ast.FunctionCall:
		if v.Name.Name == "now" {
			return
		}
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("DateTime default value must be a valid RFC3339 string or now()."),
			value.Span(),
		))
	default:
		ctx.PushError(diagnostics.NewValidationError(
			"DateTime default value must be a string or now().",
			value.Span(),
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
func validateJsonDefaultValue(value ast.Expression, ctx *ValidationContext) {
	switch v := value.(type) {
	case ast.StringLiteral:
		// Validate JSON format
		if err := validateJSONString(v.Value); err != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("Parse error: \"%s\" is not a valid JSON string. (%s)", v.Value, err.Error()),
				"@default",
				v.Span(),
			))
		}
		return
	default:
		ctx.PushError(diagnostics.NewValidationError(
			"JSON default value must be a valid JSON string.",
			value.Span(),
		))
	}
}

// validateJSONString validates that a string is valid JSON.
func validateJSONString(value string) error {
	var jsonValue interface{}
	return json.Unmarshal([]byte(value), &jsonValue)
}

// validateBytesDefaultValue validates Bytes default values.
func validateBytesDefaultValue(value ast.Expression, ctx *ValidationContext) {
	switch v := value.(type) {
	case ast.StringLiteral:
		// Validate base64 format
		if err := validateBase64String(v.Value); err != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("Parse error: \"%s\" is not a valid base64 string. (%s)", v.Value, err.Error()),
				"@default",
				v.Span(),
			))
		}
		return
	default:
		ctx.PushError(diagnostics.NewValidationError(
			"Bytes default value must be a valid base64 string.",
			value.Span(),
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
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"You defined a database name for the default value of a field on the model. This is not supported by the provider.",
					"@default",
					value.Span(),
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
	if funcCall, ok := value.(ast.FunctionCall); ok {
		if funcCall.Name.Name == "auto" {
			if !ctx.HasCapability(ConnectorCapabilityDefaultValueAuto) {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"The current connector does not support the `auto()` function.",
					"@default",
					funcCall.Span(),
				))
			}
		}
	}
}
