// Package pslcore provides database name validation functions.
package validation

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateDatabaseName validates a database name against connector requirements.
func validateDatabaseName(name string, maxLength int, ctx *ValidationContext) error {
	if len(name) == 0 {
		return fmt.Errorf("database name cannot be empty")
	}

	if len(name) > maxLength {
		return fmt.Errorf("database name '%s' exceeds maximum length of %d characters", name, maxLength)
	}

	// Check for reserved keywords (basic check)
	reservedKeywords := []string{
		"select", "from", "where", "insert", "update", "delete", "create", "drop",
		"alter", "table", "index", "constraint", "primary", "key", "foreign",
		"references", "default", "null", "not", "unique", "check", "view",
	}
	nameLower := strings.ToLower(name)
	for _, keyword := range reservedKeywords {
		if nameLower == keyword {
			return fmt.Errorf("database name '%s' is a reserved keyword", name)
		}
	}

	// Check for invalid characters (basic check - connector-specific rules may vary)
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return fmt.Errorf("database name '%s' contains invalid character '%c'", name, r)
		}
	}

	return nil
}

// validateModelDatabaseName validates a model's database name.
func validateModelDatabaseName(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	dbName := model.DatabaseName()
	maxLength := ctx.MaxIdentifierLength()

	if err := validateDatabaseName(dbName, maxLength, ctx); err != nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Model '%s' has invalid database name: %s", model.Name(), err.Error()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}

// validateFieldDatabaseName validates a field's database name.
func validateFieldDatabaseName(field *database.ScalarFieldWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	dbName := field.DatabaseName()
	maxLength := ctx.MaxIdentifierLength()

	if err := validateDatabaseName(dbName, maxLength, ctx); err != nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Field '%s' in model '%s' has invalid database name: %s", field.Name(), model.Name(), err.Error()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}

// validateEnumDatabaseName validates an enum's database name.
func validateEnumDatabaseName(enum *database.EnumWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityEnums) {
		return
	}

	dbName := enum.DatabaseName()
	maxLength := ctx.MaxIdentifierLength()

	if err := validateDatabaseName(dbName, maxLength, ctx); err != nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Enum '%s' has invalid database name: %s", enum.Name(), err.Error()),
			diagnostics.NewSpan(0, 0, enum.FileID()),
		))
	}
}

// validateIndexDatabaseName validates an index's database name.
func validateIndexDatabaseName(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	mappedName := index.MappedName()
	if mappedName == nil {
		return
	}

	maxLength := ctx.MaxIdentifierLength()

	if err := validateDatabaseName(*mappedName, maxLength, ctx); err != nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Index in model '%s' has invalid database name: %s", model.Name(), err.Error()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}

// validatePrimaryKeyDatabaseName validates a primary key's database name.
func validatePrimaryKeyDatabaseName(pk *database.PrimaryKeyWalker, model *database.ModelWalker, ctx *ValidationContext) {
	mappedName := pk.MappedName()
	if mappedName == nil {
		return
	}

	maxLength := ctx.MaxIdentifierLength()

	if err := validateDatabaseName(*mappedName, maxLength, ctx); err != nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Primary key in model '%s' has invalid database name: %s", model.Name(), err.Error()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}

// validateDefaultDatabaseName validates a default value's database name.
func validateDefaultDatabaseName(field *database.ScalarFieldWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	defaultValue := field.DefaultValue()
	if defaultValue == nil {
		return
	}

	mappedName := defaultValue.MappedName()
	if mappedName == nil {
		return
	}

	maxLength := ctx.MaxIdentifierLength()

	if err := validateDatabaseName(*mappedName, maxLength, ctx); err != nil {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Default value for field '%s' in model '%s' has invalid database name: %s", field.Name(), model.Name(), err.Error()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}
