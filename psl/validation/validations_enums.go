// Package pslcore provides enum validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateEnumDatabaseNameClashes checks for database name clashes between enums.
func validateEnumDatabaseNameClashes(ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityEnums) {
		return
	}

	enumNames := make(map[string][]*database.EnumWalker)

	enums := ctx.Db.WalkEnums()
	for _, enum := range enums {
		dbName := enum.DatabaseName()
		enumNames[dbName] = append(enumNames[dbName], enum)
	}

	for dbName, enums := range enumNames {
		if len(enums) > 1 {
			enumNamesList := ""
			for i, e := range enums {
				if i > 0 {
					enumNamesList += ", "
				}
				enumNamesList += e.Name()
			}
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Multiple enums have the same database name '%s': %s", dbName, enumNamesList),
				diagnostics.NewSpan(0, 0, enums[0].FileID()),
			))
		}
	}
}

// validateEnumHasValues checks that an enum has at least one value.
func validateEnumHasValues(enum *database.EnumWalker, ctx *ValidationContext) {
	values := enum.Values()
	if len(values) == 0 {
		astEnum := enum.AstEnum()
		if astEnum != nil {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Enum '%s' must have at least one value.", enum.Name()),
				astEnum.Span(),
			))
		} else {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Enum '%s' must have at least one value.", enum.Name()),
				diagnostics.NewSpan(0, 0, enum.FileID()),
			))
		}
	}
}

// validateEnumConnectorSupport checks that the connector supports enums.
func validateEnumConnectorSupport(enum *database.EnumWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityEnums) {
		astEnum := enum.AstEnum()
		if astEnum != nil {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("You defined the enum `%s`. But the current connector does not support enums.", enum.Name()),
				astEnum.Span(),
			))
		} else {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("You defined the enum `%s`. But the current connector does not support enums.", enum.Name()),
				diagnostics.NewSpan(0, 0, enum.FileID()),
			))
		}
	}
}

// validateEnumSchemaAttribute checks that @@schema is properly defined for enums.
func validateEnumSchemaAttribute(enum *database.EnumWalker, ctx *ValidationContext) {
	if ctx.Datasource == nil {
		return
	}

	schema := enum.Schema()
	if schema != nil {
		// Get schema name from StringId
		schemaName := ctx.Db.GetString(schema.Name)

		// Check if schema is in datasource schemas list
		if len(ctx.Datasource.Schemas) > 0 {
			found := false
			for _, dsSchema := range ctx.Datasource.Schemas {
				if dsSchema == schemaName {
					found = true
					break
				}
			}
			if !found {
				astEnum := enum.AstEnum()
				if astEnum != nil {
					ctx.PushError(diagnostics.NewValidationError(
						fmt.Sprintf("Enum '%s' schema '%s' is not defined in datasource schemas.", enum.Name(), schemaName),
						astEnum.Span(),
					))
				} else {
					ctx.PushError(diagnostics.NewValidationError(
						fmt.Sprintf("Enum '%s' schema '%s' is not defined in datasource schemas.", enum.Name(), schemaName),
						diagnostics.NewSpan(0, 0, enum.FileID()),
					))
				}
			}
		} else if !ctx.HasCapability(ConnectorCapabilityMultiSchema) {
			astEnum := enum.AstEnum()
			if astEnum != nil {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Enum '%s' has @@schema attribute but connector does not support multi-schema.", enum.Name()),
					astEnum.Span(),
				))
			} else {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Enum '%s' has @@schema attribute but connector does not support multi-schema.", enum.Name()),
					diagnostics.NewSpan(0, 0, enum.FileID()),
				))
			}
		}
	}
}
