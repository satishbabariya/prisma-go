// Package pslcore provides field length validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateLengthUsedWithCorrectTypes validates that length argument is only used with String or Bytes types.
func validateLengthUsedWithCorrectTypes(field *database.ScalarFieldWalker, attributeName string, span diagnostics.Span, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityIndexColumnLengthPrefixing) {
		return
	}

	// Check if field has length()
	if field.Length() == nil {
		return
	}

	scalarType := field.ScalarType()
	if scalarType == nil {
		return
	}

	// Length is only allowed with String or Bytes
	if *scalarType != database.ScalarTypeString && *scalarType != database.ScalarTypeBytes {
		ctx.PushError(diagnostics.NewFieldValidationError(
			fmt.Sprintf("The `%s` argument can only be used with String or Bytes types, but the field is of type %s.", attributeName, *scalarType),
			"field",
			field.Model().Name(),
			field.Name(),
			span,
		))
	}
}

// validateFieldLengthPrefix validates that length prefix is only used with String or Bytes.
func validateFieldLengthPrefix(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityIndexColumnLengthPrefixing) {
		return
	}

	fieldType := field.ScalarFieldType()
	if fieldType.Unsupported != nil {
		return
	}

	scalarType := field.ScalarType()
	if scalarType == nil {
		return
	}

	// Length prefix is only allowed with String or Bytes
	if field.Length() == nil {
		return
	}

	if *scalarType != database.ScalarTypeString && *scalarType != database.ScalarTypeBytes {
		ctx.PushError(diagnostics.NewFieldValidationError(
			fmt.Sprintf("Length prefix can only be used with String or Bytes types, but the field is of type %s.", *scalarType),
			"field",
			field.Model().Name(),
			field.Name(),
			field.Span(),
		))
	}
}
