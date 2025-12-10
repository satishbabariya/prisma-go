// Package pslcore provides enum value validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateEnumValueNames validates that enum value names are unique.
func validateEnumValueNames(enum *database.EnumWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityEnums) {
		return
	}

	values := enum.Values()
	valueNames := make(map[string]bool)

	for _, value := range values {
		name := value.Name()
		if valueNames[name] {
			astEnum := enum.AstEnum()
			if astEnum != nil {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Enum '%s' has duplicate value name '%s'.", enum.Name(), name),
					astEnum.Span(),
				))
			} else {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Enum '%s' has duplicate value name '%s'.", enum.Name(), name),
					diagnostics.NewSpan(0, 0, enum.FileID()),
				))
			}
		}
		valueNames[name] = true
	}
}

// validateEnumValueDatabaseNames validates that enum value database names are unique.
func validateEnumValueDatabaseNames(enum *database.EnumWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityEnums) {
		return
	}

	values := enum.Values()
	dbNames := make(map[string]bool)

	for _, value := range values {
		dbName := value.DatabaseName()
		if dbNames[dbName] {
			astEnum := enum.AstEnum()
			if astEnum != nil {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Enum '%s' has duplicate value database name '%s'.", enum.Name(), dbName),
					astEnum.Span(),
				))
			} else {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Enum '%s' has duplicate value database name '%s'.", enum.Name(), dbName),
					diagnostics.NewSpan(0, 0, enum.FileID()),
				))
			}
		}
		dbNames[dbName] = true
	}
}

// validateEnumValueReservedNames validates that enum values don't use reserved names.
func validateEnumValueReservedNames(enum *database.EnumWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityEnums) {
		return
	}

	reservedNames := []string{"null", "undefined", "true", "false"}
	values := enum.Values()

	for _, value := range values {
		name := value.Name()
		for _, reserved := range reservedNames {
			if name == reserved {
				astEnum := enum.AstEnum()
				if astEnum != nil {
					// TODO: Add warning support when available
					// For now, we'll skip warnings since they're not yet implemented
					// ctx.PushWarning(diagnostics.NewDatamodelWarning(
					// 	fmt.Sprintf("Enum value '%s' in enum '%s' uses reserved name '%s'.", name, enum.Name(), reserved),
					// 	astEnum.Span(),
					// ))
					_ = astEnum
				}
			}
		}
	}
}
