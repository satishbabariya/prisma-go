// Package pslcore provides composite type validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateCompositeTypesSupport checks if composite types are supported.
func validateCompositeTypesSupport(ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityCompositeTypes) {
		return
	}

	compositeTypes := ctx.Db.WalkCompositeTypes()
	for _, ct := range compositeTypes {
		astCT := ct.AstCompositeType()
		if astCT != nil {
			connectorName := "the current connector"
			if ctx.Connector != nil {
				connectorName = ctx.Connector.ProviderName()
			}
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Composite types are not supported on %s.", connectorName),
				astCT.Span(),
			))
		}
	}
}

// validateCompositeTypeMoreThanOneField checks that composite types have at least one field.
func validateCompositeTypeMoreThanOneField(ct *database.CompositeTypeWalker, ctx *ValidationContext) {
	fields := ct.Fields()
	if len(fields) == 0 {
		astCT := ct.AstCompositeType()
		if astCT != nil {
			ctx.PushError(diagnostics.NewValidationError(
				"A type must have at least one field defined.",
				astCT.Span(),
			))
		}
	}
}

// validateCompositeTypeFieldDefaultValue validates default values on composite type fields.
func validateCompositeTypeFieldDefaultValue(field *database.CompositeTypeFieldWalker, ctx *ValidationContext) {
	// TODO: Get default value when DefaultValue() method is available on CompositeTypeFieldWalker
	// For now, this is a placeholder

	// Validate that composite type fields cannot have mapped names for default values
	// This validation will be added when default value access is available
	_ = field
}

// validateCompositeTypeCycles detects cycles in composite type references.
func validateCompositeTypeCycles(ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityCompositeTypes) {
		return
	}

	// Collect all required fields from composite types
	type fieldToTraverse struct {
		field     *database.CompositeTypeFieldWalker
		visited   []string // Use names instead of IDs since IDs are not accessible
		pathNames []string
	}

	var fieldsToTraverse []fieldToTraverse
	compositeTypes := ctx.Db.WalkCompositeTypes()

	// Build a map to find parent composite type from field
	fieldToCT := make(map[*database.CompositeTypeFieldWalker]*database.CompositeTypeWalker)

	// Initialize with all required fields
	for _, ct := range compositeTypes {
		fields := ct.Fields()
		for _, field := range fields {
			fieldToCT[field] = ct
			astField := field.AstField()
			if astField != nil && !astField.FieldType.IsOptional() && !astField.FieldType.IsArray() {
				fieldsToTraverse = append(fieldsToTraverse, fieldToTraverse{
					field:     field,
					visited:   []string{ct.Name()},
					pathNames: []string{ct.Name()},
				})
			}
		}
	}

	// Traverse fields to detect cycles
	for len(fieldsToTraverse) > 0 {
		current := fieldsToTraverse[len(fieldsToTraverse)-1]
		fieldsToTraverse = fieldsToTraverse[:len(fieldsToTraverse)-1]

		fieldType := current.field.Type()
		if fieldType.CompositeTypeID == nil {
			continue
		}

		ctID := *fieldType.CompositeTypeID
		ct := ctx.Db.WalkCompositeType(ctID)
		if ct == nil {
			continue
		}

		ctName := ct.Name()

		// Get parent composite type from the field
		parentCT, hasParent := fieldToCT[current.field]
		if !hasParent {
			continue
		}
		parentCTName := parentCT.Name()

		// Check for self-reference cycle
		if parentCTName == ctName {
			astField := current.field.AstField()
			if astField != nil {
				message := "The type is the same as the parent and causes an endless cycle. Please change the field to be either optional or a list."
				ctx.PushError(diagnostics.NewCompositeTypeFieldValidationError(
					message,
					parentCT.Name(),
					current.field.Name(),
					diagnostics.NewSpan(0, 0, ct.FileID()),
				))
			}
			continue
		}

		// Check if we've seen this composite type before (cycle detected)
		for _, visitedName := range current.visited {
			if visitedName == ctName {
				// Cycle detected - build path string
				pathStr := ""
				for i, name := range current.pathNames {
					if i > 0 {
						pathStr += " → "
					}
					pathStr += fmt.Sprintf("`%s`", name)
				}
				pathStr += fmt.Sprintf(" → `%s`", ctName)

				message := fmt.Sprintf(
					"The types cause an endless cycle in the path %s. Please change one of the fields to be either optional or a list to break the cycle.",
					pathStr,
				)

				astField := current.field.AstField()
				if astField != nil {
					ctx.PushError(diagnostics.NewCompositeTypeFieldValidationError(
						message,
						parentCTName,
						current.field.Name(),
						diagnostics.NewSpan(0, 0, ct.FileID()),
					))
				}
				continue
			}
		}

		// Add to visited and continue traversal
		newVisited := append(current.visited, ctName)
		newPathNames := append(current.pathNames, ctName)

		// Add all required fields from this composite type
		fields := ct.Fields()
		for _, field := range fields {
			astField := field.AstField()
			if astField != nil && !astField.FieldType.IsOptional() && !astField.FieldType.IsArray() {
				fieldsToTraverse = append(fieldsToTraverse, fieldToTraverse{
					field:     field,
					visited:   newVisited,
					pathNames: newPathNames,
				})
			}
		}
	}
}

// validateCompositeTypeFieldTypes validates composite type field types.
func validateCompositeTypeFieldTypes(field *database.CompositeTypeFieldWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityCompositeTypes) {
		return
	}

	fieldType := field.Type()

	// Check if field type is unsupported
	if fieldType.Unsupported != nil {
		// Get file ID from AST field
		astField := field.AstField()
		fileID := diagnostics.FileIDZero
		if astField != nil {
			// Use a placeholder span - TODO: Add Span() method to Field
			fileID = diagnostics.FileIDZero
		}
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Composite type field '%s' has unsupported type.", field.Name()),
			diagnostics.NewSpan(0, 0, fileID),
		))
	}
}

// validateCompositeTypeFieldArity validates composite type field arity.
func validateCompositeTypeFieldArity(field *database.CompositeTypeFieldWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityCompositeTypes) {
		return
	}

	// Composite type fields can be optional or lists
	// This is a placeholder for future validations
	_ = field
}
