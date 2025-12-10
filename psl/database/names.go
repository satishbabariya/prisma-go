// Package parserdatabase provides name resolution functionality.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// Names holds resolved names for use in the validation process.
type Names struct {
	// Models, enums, composite types and type aliases
	Tops map[StringId]TopId
	// Generators have their own namespace.
	Generators map[StringId]TopId
	// Datasources have their own namespace.
	Datasources map[StringId]TopId
	// Model fields: (ModelId, field name StringId) -> FieldId
	ModelFields map[ModelFieldKey]uint32
	// Composite type fields: (CompositeTypeId, field name StringId) -> FieldId
	CompositeTypeFields map[CompositeTypeFieldKey]uint32
}

// ModelFieldKey is a key for model field lookup.
type ModelFieldKey struct {
	ModelID ModelId
	NameID  StringId
}

// CompositeTypeFieldKey is a key for composite type field lookup.
type CompositeTypeFieldKey struct {
	CompositeTypeID CompositeTypeId
	NameID          StringId
}

// NewNames creates a new empty Names structure.
func NewNames() Names {
	return Names{
		Tops:                make(map[StringId]TopId),
		Generators:          make(map[StringId]TopId),
		Datasources:         make(map[StringId]TopId),
		ModelFields:         make(map[ModelFieldKey]uint32),
		CompositeTypeFields: make(map[CompositeTypeFieldKey]uint32),
	}
}

// ResolveNames populates Names and validates that there are no name collisions
// in the following namespaces:
// - Model, enum and type alias names
// - Generators
// - Datasources
// - Model fields for each model
// - Enum variants for each enum
func ResolveNames(db *ParserDatabase, diags *diagnostics.Diagnostics) Names {
	names := NewNames()
	tmpNames := make(map[string]bool) // throwaway container for duplicate checking

	for _, file := range db.asts.files {
		for i, top := range file.AST.Tops {
			topID := TopId{
				FileID: file.FileID,
				ID:     uint32(i),
			}

			// Validate identifier
			var identifier *ast.Identifier
			var topType string

			if model := top.AsModel(); model != nil {
				identifier = &model.Name
				topType = "Model"
				validateIdentifier(identifier, "Model", diags)
				validateModelName(model, "model", diags)

				// Validate model fields
				for j, field := range model.Fields {
					validateIdentifier(&field.Name, "Field", diags)
					fieldNameID := db.interner.Intern(field.Name.Name)
					key := ModelFieldKey{
						ModelID: ModelId(topID),
						NameID:  fieldNameID,
					}
					if _, exists := names.ModelFields[key]; exists {
						diags.PushError(diagnostics.NewDuplicateFieldError(
							model.Name.Name,
							field.Name.Name,
							"model",
							field.Name.Span(),
						))
					} else {
						names.ModelFields[key] = uint32(j)
					}
				}
			} else if enum := top.AsEnum(); enum != nil {
				identifier = &enum.Name
				topType = "Enum"
				validateIdentifier(identifier, "Enum", diags)
				validateEnumName(enum, diags)

				// Validate enum values
				tmpNames = make(map[string]bool)
				for _, value := range enum.Values {
					validateIdentifier(&value.Name, "Enum Value", diags)
					if tmpNames[value.Name.Name] {
						diags.PushError(diagnostics.NewDuplicateEnumValueError(
							enum.Name.Name,
							value.Name.Name,
							value.Name.Span(),
						))
					}
					tmpNames[value.Name.Name] = true
				}
			} else if source := top.AsSource(); source != nil {
				identifier = &source.Name
				topType = "Datasource"
				checkForDuplicateProperties(top, source.Properties, tmpNames, diags)
			} else if generator := top.AsGenerator(); generator != nil {
				identifier = &generator.Name
				topType = "Generator"
				checkForDuplicateProperties(top, generator.Properties, tmpNames, diags)
			}

			if identifier != nil {
				nameID := db.interner.Intern(identifier.Name)
				var namespace map[StringId]TopId

				switch topType {
				case "Datasource":
					namespace = names.Datasources
				case "Generator":
					namespace = names.Generators
				default:
					namespace = names.Tops
				}

				if existingTop, exists := namespace[nameID]; exists {
					// Get existing top type
					existingFile := db.asts.Get(existingTop.FileID)
					if existingFile != nil && int(existingTop.ID) < len(existingFile.AST.Tops) {
						existingTopAST := existingFile.AST.Tops[existingTop.ID]
						existingTopType := getTopType(existingTopAST)
						diags.PushError(diagnostics.NewDuplicateTopError(
							identifier.Name,
							topType,
							existingTopType,
							identifier.Span(),
						))
					}
				} else {
					namespace[nameID] = topID
				}
			}
		}
	}

	return names
}

// validateIdentifier validates that an identifier follows naming rules.
func validateIdentifier(ident *ast.Identifier, schemaItem string, diags *diagnostics.Diagnostics) {
	if ident.Name == "" {
		diags.PushError(diagnostics.NewValidationError(
			"The name of a "+schemaItem+" must not be empty.",
			ident.Span(),
		))
	} else if len(ident.Name) > 0 && ident.Name[0] >= '0' && ident.Name[0] <= '9' {
		diags.PushError(diagnostics.NewValidationError(
			"The name of a "+schemaItem+" must not start with a number.",
			ident.Span(),
		))
	} else if contains(ident.Name, '-') {
		diags.PushError(diagnostics.NewValidationError(
			"The character `-` is not allowed in "+schemaItem+" names.",
			ident.Span(),
		))
	}
}

// validateModelName validates a model name.
func validateModelName(model *ast.Model, containerType string, diags *diagnostics.Diagnostics) {
	if IsReservedTypeName(model.Name.Name) {
		diags.PushError(diagnostics.NewModelValidationError(
			"The "+containerType+" name `"+model.Name.Name+"` is invalid. It is a reserved name. Please change it. Read more at https://pris.ly/d/naming-models",
			"model",
			model.Name.Name,
			model.Span(),
		))
	}
}

// validateEnumName validates an enum name.
func validateEnumName(enum *ast.Enum, diags *diagnostics.Diagnostics) {
	if IsReservedTypeName(enum.Name.Name) {
		diags.PushError(diagnostics.NewEnumValidationError(
			"The enum name `"+enum.Name.Name+"` is invalid. It is a reserved name. Please change it. Read more at https://www.prisma.io/docs/reference/tools-and-interfaces/prisma-schema/data-model#naming-enums",
			enum.Name.Name,
			enum.Span(),
		))
	}
}

// checkForDuplicateProperties checks for duplicate properties in config blocks.
func checkForDuplicateProperties(top ast.Top, props []ast.ConfigBlockProperty, tmpNames map[string]bool, diags *diagnostics.Diagnostics) {
	clearMap(tmpNames)
	for _, prop := range props {
		if tmpNames[prop.Name.Name] {
			diags.PushError(diagnostics.NewDuplicateConfigKeyError(
				getTopType(top)+" \""+getName(top)+"\"",
				prop.Name.Name,
				prop.Name.Span(),
			))
		}
		tmpNames[prop.Name.Name] = true
	}
}

// getTopType returns the type string for a top-level AST node.
func getTopType(top ast.Top) string {
	if top.AsModel() != nil {
		return "model"
	} else if top.AsEnum() != nil {
		return "enum"
	} else if top.AsSource() != nil {
		return "datasource"
	} else if top.AsGenerator() != nil {
		return "generator"
	}
	return "unknown"
}

// getName returns the name for a top-level AST node.
func getName(top ast.Top) string {
	if model := top.AsModel(); model != nil {
		return model.Name.Name
	} else if enum := top.AsEnum(); enum != nil {
		return enum.Name.Name
	} else if source := top.AsSource(); source != nil {
		return source.Name.Name
	} else if generator := top.AsGenerator(); generator != nil {
		return generator.Name.Name
	}
	return ""
}

// contains checks if a string contains a character.
func contains(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// clearMap clears a map by deleting all keys.
func clearMap(m map[string]bool) {
	for k := range m {
		delete(m, k)
	}
}
