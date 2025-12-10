// Package codegen provides relation parsing from AST.
package codegen

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// parseRelationAttribute parses @relation attribute to extract fields and references
func parseRelationAttribute(field *ast.Field) (fields []string, references []string, err error) {
	for _, attr := range field.Attributes {
		if attr.Name.Name != "relation" {
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
				if arrayExpr := arg.Value.AsArray(); arrayExpr != nil {
					for _, elem := range arrayExpr.Elements {
						if ident, ok := elem.(ast.Identifier); ok {
							fields = append(fields, ident.Name)
						}
					}
				}
			} else if argName == "references" {
				// Parse array of field names
				if arrayExpr := arg.Value.AsArray(); arrayExpr != nil {
					for _, elem := range arrayExpr.Elements {
						if ident, ok := elem.(ast.Identifier); ok {
							references = append(references, ident.Name)
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
	if relationField.FieldType.Type != nil {
		relationTo = relationField.FieldType.Type.Name()
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
				if fieldNameLower == pattern && !hasAttribute(&field, "id") {
					return field.Name.Name, "id", nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("could not determine foreign key for relation field %s", relationField.Name.Name)
}
