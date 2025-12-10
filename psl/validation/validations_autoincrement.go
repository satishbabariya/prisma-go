// Package pslcore provides autoincrement validation functions.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateAutoIncrement validates autoincrement usage on fields.
func validateAutoIncrement(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	// Collect all autoincrement fields
	autoincrementFields := []*database.ScalarFieldWalker{}
	for _, field := range model.ScalarFields() {
		if field.IsIgnored() {
			continue
		}

		defaultValue := field.DefaultValue()
		if defaultValue != nil && defaultValue.IsAutoIncrement() {
			autoincrementFields = append(autoincrementFields, field)
		}
	}

	// Early return if no autoincrement fields
	if len(autoincrementFields) == 0 {
		return
	}

	// First check if the provider supports autoincrement at all
	if !ctx.HasCapability(ConnectorCapabilityAutoIncrement) {
		for _, field := range autoincrementFields {
			defaultValue := field.DefaultValue()
			if defaultValue == nil {
				continue
			}

			value := defaultValue.Value()
			if value == nil {
				continue
			}

			span := value.Span()
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The `autoincrement()` default value is used with a datasource that does not support it.",
				"@default",
				span,
			))
		}
		return
	}

	// Check if multiple autoincrement fields are allowed
	if !ctx.HasCapability(ConnectorCapabilityAutoIncrementMultipleAllowed) && len(autoincrementFields) > 1 {
		astModel := model.AstModel()
		if astModel != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The `autoincrement()` default value is used multiple times on this model even though the underlying datasource only supports one instance per table.",
				"@default",
				astModel.Span(),
			))
		}
	}

	// Validate each autoincrement field
	for _, field := range autoincrementFields {
		astField := field.AstField()
		if astField == nil {
			continue
		}

		// Check if autoincrement is allowed on non-id fields
		if !ctx.HasCapability(ConnectorCapabilityAutoIncrementAllowedOnNonId) {
			if !field.IsSinglePK() {
				defaultValue := field.DefaultValue()
				if defaultValue != nil {
					value := defaultValue.Value()
					if value != nil {
						ctx.PushError(diagnostics.NewAttributeValidationError(
							"The `autoincrement()` default value is used on a non-id field even though the datasource does not support this.",
							"@default",
							value.Span(),
						))
					}
				}
			}
		}

		// Check if autoincrement is allowed on non-indexed fields
		if !ctx.HasCapability(ConnectorCapabilityAutoIncrementNonIndexedAllowed) {
			// Check if field is indexed for autoincrement purposes
			if !model.FieldIsIndexedForAutoincrement(database.ScalarFieldId(field.FieldID())) {
				defaultValue := field.DefaultValue()
				if defaultValue != nil {
					value := defaultValue.Value()
					if value != nil {
						ctx.PushError(diagnostics.NewAttributeValidationError(
							"The `autoincrement()` default value is used on a non-indexed field even though the datasource does not support this.",
							"@default",
							value.Span(),
						))
					}
				}
			}
		}
	}
}
