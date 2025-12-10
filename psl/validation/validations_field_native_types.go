// Package pslcore provides native type validation functions.
package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateNativeTypeArguments validates native type arguments for a field.
func validateNativeTypeArguments(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	// TODO: Get raw native type when RawNativeType() method is available
	// For now, check if field has native type
	nativeType := field.NativeType()
	if nativeType == nil {
		return
	}

	// Get datasource name
	datasourceName := "Default"
	if ctx.Datasource != nil {
		datasourceName = ctx.Datasource.Provider
	}

	// Validate that the attribute is scoped with the right datasource name
	if ctx.Datasource != nil {
		// TODO: Check if native type scope matches datasource name
		// For now, this is a placeholder
		_ = datasourceName
	}

	// TODO: Validate native type constructor exists
	// TODO: Validate argument count matches constructor requirements
	// TODO: Validate native type is compatible with scalar type
	// This requires connector support which is not yet fully implemented
	_ = nativeType
}

// validateUnsupportedFieldType validates unsupported field types.
func validateUnsupportedFieldType(field *database.ScalarFieldWalker, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	if ctx.Datasource == nil {
		return
	}

	fieldType := field.ScalarFieldType()
	if fieldType.Unsupported == nil {
		return
	}

	astField := field.AstField()
	if astField == nil {
		return
	}

	// Get the unsupported type string from the field type name
	unsupportedTypeName := astField.FieldType.TypeName()
	if unsupportedTypeName == "" {
		return
	}

	// Regex pattern to match Unsupported("TypeName(params)") format
	// Matches: prefix (required), optional params in parentheses, optional suffix
	typeRegex := regexp.MustCompile(`(?m)^\s*(?P<prefix>[^(]+)\s*(?:\((?P<params>.*?)\))?\s*(?P<suffix>.+)?\s*$`)

	matches := typeRegex.FindStringSubmatch(unsupportedTypeName)
	if matches == nil {
		return
	}

	prefixIdx := typeRegex.SubexpIndex("prefix")
	paramsIdx := typeRegex.SubexpIndex("params")

	if prefixIdx < 0 || len(matches) <= prefixIdx {
		return
	}

	prefix := strings.TrimSpace(matches[prefixIdx])
	var args []string
	if paramsIdx >= 0 && len(matches) > paramsIdx && matches[paramsIdx] != "" {
		// Split params by comma and trim
		paramStr := strings.TrimSpace(matches[paramsIdx])
		if paramStr != "" {
			parts := strings.Split(paramStr, ",")
			for _, part := range parts {
				args = append(args, strings.TrimSpace(part))
			}
		}
	}

	// Try to parse as native type using connector
	if ctx.Connector != nil {
		span := astField.Span()
		diags := diagnostics.NewDiagnostics()
		nativeType := ctx.Connector.ParseNativeType(prefix, args, span, &diags)

		if nativeType != nil && len(diags.Errors()) == 0 {
			// Get scalar type for this native type
			scalarType := ctx.Connector.ScalarTypeForNativeType(nativeType, ctx.ExtensionTypes)
			if scalarType != nil {
				// Get datasource name
				datasourceName := ctx.Datasource.Provider
				if datasourceName == "" {
					datasourceName = "datasource"
				}

				// Get native type string representation
				nativeTypeStr, _ := ctx.Connector.NativeTypeToParts(nativeType)
				if nativeTypeStr == "" {
					nativeTypeStr = prefix
				}

				// Get scalar type display name
				scalarTypeName := getScalarTypeDisplayName(scalarType)
				if scalarTypeName == "" {
					scalarTypeName = "Unknown"
				}

				message := fmt.Sprintf(
					"The type `Unsupported(\"%s\")` you specified in the type definition for the field `%s` is supported as a native type by Prisma. Please use the native type notation `%s @%s.%s` for full support.",
					unsupportedTypeName,
					field.Name(),
					scalarTypeName,
					datasourceName,
					nativeTypeStr,
				)

				ctx.PushError(diagnostics.NewValidationError(
					message,
					astField.Span(),
				))
			}
		}
	}
}

// getScalarTypeDisplayName returns a display name for a ScalarFieldType.
func getScalarTypeDisplayName(scalarType *ScalarFieldType) string {
	if scalarType == nil {
		return ""
	}
	// ScalarFieldType is an alias for database.ScalarFieldType
	pdType := (*database.ScalarFieldType)(scalarType)
	if pdType.BuiltInScalar != nil {
		return string(*pdType.BuiltInScalar)
	}
	if pdType.EnumID != nil {
		return "Enum"
	}
	if pdType.CompositeTypeID != nil {
		return "CompositeType"
	}
	return "Unknown"
}
