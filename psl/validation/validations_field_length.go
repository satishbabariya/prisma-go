// Package pslcore provides field length validation functions.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateLengthUsedWithCorrectTypes validates that length argument is only used with String or Bytes types.
func validateLengthUsedWithCorrectTypes(field *database.ScalarFieldWalker, attributeName string, span diagnostics.Span, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityIndexColumnLengthPrefixing) {
		return
	}

	// TODO: Check if field has length() when scalar_field_attributes is available
	// For now, this is a placeholder
	_ = field
	_ = attributeName
	_ = span
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
	// TODO: Check if field has length() when available
	// For now, this is a placeholder
	_ = scalarType
}
