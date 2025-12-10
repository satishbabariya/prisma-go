// Package pslcore provides scalar type validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateScalarTypeSupport validates that scalar types are supported by the connector.
func validateScalarTypeSupport(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	scalarType := field.ScalarType()
	if scalarType == nil {
		return
	}

	fieldType := field.ScalarFieldType()

	// Check if type is unsupported
	if fieldType.Unsupported != nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Field '%s' has unsupported type.", field.Name()),
			diagnostics.NewSpan(0, 0, field.Model().FileID()),
		))
		return
	}

	// Check connector-specific type support
	switch *scalarType {
	case database.ScalarTypeJson:
		if !ctx.HasCapability(ConnectorCapabilityJson) {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Field '%s' has Json type but connector does not support Json.", field.Name()),
				diagnostics.NewSpan(0, 0, field.Model().FileID()),
			))
		}
	case database.ScalarTypeBigInt:
		// BigInt support is generally available, but some connectors may not support it
		// TODO: Add ConnectorCapabilityBigInt if needed
	case database.ScalarTypeDecimal:
		// Decimal support varies by connector
		// TODO: Add ConnectorCapabilityDecimal if needed
	case database.ScalarTypeBytes:
		// Bytes support varies by connector
		// TODO: Add ConnectorCapabilityBytes if needed
	}
}

// validateNativeTypeSupport validates that native types are supported.
func validateNativeTypeSupport(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	nativeType := field.NativeType()
	if nativeType == nil {
		return
	}

	scalarType := field.ScalarType()
	if scalarType == nil {
		return
	}

	// Basic validation - native type should match scalar type
	// TODO: Validate native type compatibility with scalar type based on connector
	_ = scalarType
}

// validateFieldArity validates field arity (required, optional, list).
func validateFieldArity(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	// Check if field is optional and has a default value
	if field.IsOptional() {
		defaultValue := field.DefaultValue()
		if defaultValue == nil {
			// Optional fields without defaults are valid
			return
		}
	}

	// Check if field is a list
	if field.IsList() {
		// Lists cannot be primary keys
		if field.IsSinglePK() || field.IsPartOfCompoundPK() {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Field '%s' is a list and cannot be part of the primary key.", field.Name()),
				diagnostics.NewSpan(0, 0, field.Model().FileID()),
			))
		}
	}
}

// validateCompositeTypeFieldSupport validates composite type field support.
func validateCompositeTypeFieldSupport(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	fieldType := field.ScalarFieldType()
	if fieldType.CompositeTypeID != nil {
		if !ctx.HasCapability(ConnectorCapabilityCompositeTypes) {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Field '%s' has composite type but connector does not support composite types.", field.Name()),
				diagnostics.NewSpan(0, 0, field.Model().FileID()),
			))
		}
	}
}

// validateEnumFieldSupport validates enum field support.
func validateEnumFieldSupport(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	fieldType := field.ScalarFieldType()
	if fieldType.EnumID != nil {
		if !ctx.HasCapability(ConnectorCapabilityEnums) {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Field '%s' has enum type but connector does not support enums.", field.Name()),
				diagnostics.NewSpan(0, 0, field.Model().FileID()),
			))
		}
	}
}
