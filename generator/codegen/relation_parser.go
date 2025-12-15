// Package codegen provides relation parsing from AST.
package codegen

import (
	"fmt"
	"strings"

	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// parseRelationAttribute parses @relation attribute to extract fields and references
func parseRelationAttribute(field *ast.Field) (fields []string, references []string, err error) {
	for _, attr := range field.Attributes {
		if attr.Name.Name != "relation" {
			continue
		}

		if attr.Arguments == nil {
			continue
		}

		// Parse arguments
		for _, arg := range attr.Arguments.Arguments {
			if arg.Name == nil {
				continue
			}

			argName := arg.Name.Name
			if argName == "fields" {
				// Parse array of field names
				if arrayExpr, ok := arg.Value.AsArray(); ok {
					for _, elem := range arrayExpr.Elements {
						if ident, ok := elem.(*ast.ConstantValue); ok {
							// In V2 AST constant values can be identifiers
							fields = append(fields, ident.Value)
						} else if path, ok := elem.(*ast.PathValue); ok {
							fields = append(fields, path.String())
						} else if str, ok := elem.(*ast.StringValue); ok {
							// Sometimes redundant quotes?
							fields = append(fields, str.GetValue())
						}
					}
				}
			} else if argName == "references" {
				// Parse array of field names
				if arrayExpr, ok := arg.Value.AsArray(); ok {
					for _, elem := range arrayExpr.Elements {
						if ident, ok := elem.(*ast.ConstantValue); ok {
							references = append(references, ident.Value)
						} else if path, ok := elem.(*ast.PathValue); ok {
							references = append(references, path.String())
						} else if str, ok := elem.(*ast.StringValue); ok {
							references = append(references, str.GetValue())
						}
					}
				}
			}
		}
	}

	return fields, references, nil
}

// findForeignKeyFromRelation finds the foreign key field from @relation attribute
func findForeignKeyFromRelation(model *ast.Model, relationField *ast.Field) (string, string, error) {
	fields, references, err := parseRelationAttribute(relationField)
	if err != nil {
		return "", "", err
	}

	// If @relation has fields and references, use them
	if len(fields) > 0 && len(references) > 0 {
		// fields[0] is the foreign key field name in the current model
		// references[0] is the referenced field name (usually "id")
		return fields[0], references[0], nil
	}

	// Fallback: try to infer from field name pattern
	// For many-to-one: look for field ending in "Id" or "ID"
	relationTo := ""
	if relationField.Type != nil {
		relationTo = relationField.Type.Name
	}

	if relationTo != "" {
		// Look for foreign key field
		expectedFKPatterns := []string{
			strings.ToLower(relationTo) + "id",
			strings.ToLower(relationTo) + "_id",
			toSnakeCase(relationTo) + "_id",
		}

		for _, field := range model.Fields {
			fieldNameLower := strings.ToLower(field.Name.Name)
			for _, pattern := range expectedFKPatterns {
				if fieldNameLower == pattern && !hasAttribute(field, "id") {
					return field.Name.Name, "id", nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("could not determine foreign key for relation field %s", relationField.Name.Name)
}
