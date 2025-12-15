// Package parserdatabase provides name resolution functionality.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
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
			var identifierName string
			var identifierSpan diagnostics.Span
			var topType string

			switch t := top.(type) {
			case *v2ast.Model:
				identifierName = t.GetName()
				pos := t.TopPos()
				identifierSpan = diagnostics.NewSpan(pos.Offset, pos.Offset+len(identifierName), diagnostics.FileIDZero)
				topType = "Model"
				validateIdentifier(identifierName, identifierSpan, "Model", diags)
				validateModelName(t, "model", diags)

				// Validate model fields
				for j, field := range t.Fields {
					if field == nil {
						continue
					}
					fieldName := field.GetName()
					fieldPos := field.Pos
					fieldSpan := diagnostics.NewSpan(fieldPos.Offset, fieldPos.Offset+len(fieldName), diagnostics.FileIDZero)
					validateIdentifier(fieldName, fieldSpan, "Field", diags)
					fieldNameID := db.interner.Intern(fieldName)
					key := ModelFieldKey{
						ModelID: ModelId(topID),
						NameID:  fieldNameID,
					}
					if _, exists := names.ModelFields[key]; exists {
						diags.PushError(diagnostics.NewDuplicateFieldError(
							t.GetName(),
							fieldName,
							"model",
							fieldSpan,
						))
					} else {
						names.ModelFields[key] = uint32(j)
					}
				}
			case *v2ast.Enum:
				identifierName = t.GetName()
				pos := t.TopPos()
				identifierSpan = diagnostics.NewSpan(pos.Offset, pos.Offset+len(identifierName), diagnostics.FileIDZero)
				topType = "Enum"
				validateIdentifier(identifierName, identifierSpan, "Enum", diags)
				validateEnumName(t, diags)

				// Validate enum values
				tmpNames = make(map[string]bool)
				for _, value := range t.Values {
					if value == nil {
						continue
					}
					valueName := value.GetName()
					valuePos := value.Pos
					valueSpan := diagnostics.NewSpan(valuePos.Offset, valuePos.Offset+len(valueName), diagnostics.FileIDZero)
					validateIdentifier(valueName, valueSpan, "Enum Value", diags)
					if tmpNames[valueName] {
						diags.PushError(diagnostics.NewDuplicateEnumValueError(
							t.GetName(),
							valueName,
							valueSpan,
						))
					}
					tmpNames[valueName] = true
				}
			case *v2ast.SourceConfig:
				identifierName = t.GetName()
				pos := t.TopPos()
				identifierSpan = diagnostics.NewSpan(pos.Offset, pos.Offset+len(identifierName), diagnostics.FileIDZero)
				topType = "Datasource"
				checkForDuplicateProperties(top, t.Properties, tmpNames, diags)
			case *v2ast.GeneratorConfig:
				identifierName = t.GetName()
				pos := t.TopPos()
				identifierSpan = diagnostics.NewSpan(pos.Offset, pos.Offset+len(identifierName), diagnostics.FileIDZero)
				topType = "Generator"
				checkForDuplicateProperties(top, t.Properties, tmpNames, diags)
			}

			if identifierName != "" {
				nameID := db.interner.Intern(identifierName)
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
							identifierName,
							topType,
							existingTopType,
							identifierSpan,
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
func validateIdentifier(name string, span diagnostics.Span, schemaItem string, diags *diagnostics.Diagnostics) {
	if name == "" {
		diags.PushError(diagnostics.NewValidationError(
			"The name of a "+schemaItem+" must not be empty.",
			span,
		))
	} else if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		diags.PushError(diagnostics.NewValidationError(
			"The name of a "+schemaItem+" must not start with a number.",
			span,
		))
	} else if contains(name, '-') {
		diags.PushError(diagnostics.NewValidationError(
			"The character `-` is not allowed in "+schemaItem+" names.",
			span,
		))
	}
}

// validateModelName validates a model name.
func validateModelName(model *v2ast.Model, containerType string, diags *diagnostics.Diagnostics) {
	modelName := model.GetName()
	if IsReservedTypeName(modelName) {
		pos := model.TopPos()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(modelName), diagnostics.FileIDZero)
		diags.PushError(diagnostics.NewModelValidationError(
			"The "+containerType+" name `"+modelName+"` is invalid. It is a reserved name. Please change it. Read more at https://pris.ly/d/naming-models",
			"model",
			modelName,
			span,
		))
	}
}

// validateEnumName validates an enum name.
func validateEnumName(enum *v2ast.Enum, diags *diagnostics.Diagnostics) {
	enumName := enum.GetName()
	if IsReservedTypeName(enumName) {
		pos := enum.TopPos()
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(enumName), diagnostics.FileIDZero)
		diags.PushError(diagnostics.NewEnumValidationError(
			"The enum name `"+enumName+"` is invalid. It is a reserved name. Please change it. Read more at https://www.prisma.io/docs/reference/tools-and-interfaces/prisma-schema/data-model#naming-enums",
			enumName,
			span,
		))
	}
}

// checkForDuplicateProperties checks for duplicate properties in config blocks.
func checkForDuplicateProperties(top v2ast.Top, props []*v2ast.ConfigBlockProperty, tmpNames map[string]bool, diags *diagnostics.Diagnostics) {
	clearMap(tmpNames)
	for _, prop := range props {
		if prop == nil {
			continue
		}
		propName := prop.GetName()
		if tmpNames[propName] {
			pos := prop.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(propName), diagnostics.FileIDZero)
			diags.PushError(diagnostics.NewDuplicateConfigKeyError(
				getTopType(top)+" \""+getName(top)+"\"",
				propName,
				span,
			))
		}
		tmpNames[propName] = true
	}
}

// getTopType returns the type string for a top-level AST node.
func getTopType(top v2ast.Top) string {
	switch top.(type) {
	case *v2ast.Model:
		return "model"
	case *v2ast.Enum:
		return "enum"
	case *v2ast.SourceConfig:
		return "datasource"
	case *v2ast.GeneratorConfig:
		return "generator"
	}
	return "unknown"
}

// getName returns the name for a top-level AST node.
func getName(top v2ast.Top) string {
	switch t := top.(type) {
	case *v2ast.Model:
		return t.GetName()
	case *v2ast.Enum:
		return t.GetName()
	case *v2ast.SourceConfig:
		return t.GetName()
	case *v2ast.GeneratorConfig:
		return t.GetName()
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
