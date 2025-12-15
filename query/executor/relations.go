// Package executor provides relation metadata extraction from schema AST.
package executor

import (
	"fmt"
	"strings"

	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// ExtractRelationMetadata extracts relation metadata from PSL schema AST
func ExtractRelationMetadata(schemaAST *ast.SchemaAst, modelName string) (map[string]RelationMetadata, error) {
	relations := make(map[string]RelationMetadata)

	// Find the model
	var model *ast.Model
	for _, m := range schemaAST.Models() {
		if m.Name.Name == modelName {
			model = m
			break
		}
	}

	if model == nil {
		return relations, nil
	}

	// Get table name (default to model name in snake_case)
	_ = toSnakeCase(modelName)

	// Find all relation fields
	for _, field := range model.Fields {
		// field is *ast.Field

		if !isRelationField(field) {
			continue
		}

		relationName := field.Name.Name
		relatedModelName := getRelatedModelName(field)

		if relatedModelName == "" {
			continue
		}

		// Find the related model
		var relatedModel *ast.Model
		for _, m := range schemaAST.Models() {
			if m.Name.Name == relatedModelName {
				relatedModel = m
				break
			}
		}

		if relatedModel == nil {
			continue
		}

		relatedTableName := toSnakeCase(relatedModelName)

		// Parse @relation attribute
		fields, _, _ := parseRelationAttribute(field)

		// Determine relation type
		isList := field.Arity.IsList()
		isManyToMany := false
		junctionTable := ""
		junctionFKToSelf := ""
		junctionFKToOther := ""

		// Check if this is a many-to-many relation
		if isList {
			// Check if opposite field is also a list
			for _, oppField := range relatedModel.Fields {
				if isRelationField(oppField) && getRelatedModelName(oppField) == modelName {
					if oppField.Arity.IsList() {
						isManyToMany = true
						// Generate junction table name
						modelNames := []string{modelName, relatedModelName}
						if strings.Compare(modelNames[0], modelNames[1]) > 0 {
							modelNames[0], modelNames[1] = modelNames[1], modelNames[0]
						}
						junctionTable = fmt.Sprintf("_%s_%s", toSnakeCase(modelNames[0]), toSnakeCase(modelNames[1]))
						junctionFKToSelf = fmt.Sprintf("%s_id", toSnakeCase(modelName))
						junctionFKToOther = fmt.Sprintf("%s_id", toSnakeCase(relatedModelName))
						break
					}
				}
			}
		}

		// Determine foreign key and local key
		var foreignKey, localKey string

		if isManyToMany {
			// Many-to-many: handled via junction table
			foreignKey = ""
			localKey = "id"
		} else if isList {
			// One-to-many: FK is on the related table
			if len(fields) > 0 {
				foreignKey = fields[0]
			} else {
				// Infer FK name: modelName + "Id"
				foreignKey = fmt.Sprintf("%s_id", toSnakeCase(modelName))
			}
			localKey = "id"
		} else {
			// Many-to-one or one-to-one: FK is on this table
			if len(fields) > 0 {
				foreignKey = fields[0]
			} else {
				// Infer FK name: relatedModelName + "Id"
				foreignKey = fmt.Sprintf("%s_id", toSnakeCase(relatedModelName))
			}
			localKey = "id"
		}

		relations[relationName] = RelationMetadata{
			RelatedTable:      relatedTableName,
			ForeignKey:        foreignKey,
			LocalKey:          localKey,
			IsList:            isList,
			IsManyToMany:      isManyToMany,
			JunctionTable:     junctionTable,
			JunctionFKToSelf:  junctionFKToSelf,
			JunctionFKToOther: junctionFKToOther,
		}
	}

	return relations, nil
}

// isRelationField checks if a field is a relation field
func isRelationField(field *ast.Field) bool {
	// Check if field type is a model (not a scalar)
	typeName := ""
	if field.Type != nil {
		typeName = field.Type.Name
	}
	scalarTypes := map[string]bool{
		"int": true, "bigint": true, "string": true, "boolean": true, "bool": true,
		"datetime": true, "float": true, "decimal": true, "json": true, "bytes": true,
		"date": true, "time": true, "timestamp": true,
	}
	return !scalarTypes[strings.ToLower(typeName)]
}

// getRelatedModelName extracts the related model name from a relation field
func getRelatedModelName(field *ast.Field) string {
	if field.Type == nil {
		return ""
	}
	typeName := field.Type.Name
	// Remove array brackets if present (though V2 AST usually separates type name from arity)
	typeName = strings.TrimPrefix(typeName, "[")
	typeName = strings.TrimSuffix(typeName, "]")
	return typeName
}

// parseRelationAttribute parses @relation attribute
func parseRelationAttribute(field *ast.Field) (fields []string, references []string, relationName string) {
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
				if arrayExpr, ok := arg.Value.(*ast.ArrayExpression); ok {
					for _, elem := range arrayExpr.Elements {
						if ident, ok := elem.(*ast.ConstantValue); ok {
							fields = append(fields, ident.Value)
						} else if str, ok := elem.(*ast.StringValue); ok {
							fields = append(fields, str.GetValue())
						} else if path, ok := elem.(*ast.PathValue); ok {
							fields = append(fields, path.String())
						}
					}
				}
			} else if argName == "references" {
				// Parse array of field names
				if arrayExpr, ok := arg.Value.(*ast.ArrayExpression); ok {
					for _, elem := range arrayExpr.Elements {
						if ident, ok := elem.(*ast.ConstantValue); ok {
							references = append(references, ident.Value)
						} else if str, ok := elem.(*ast.StringValue); ok {
							references = append(references, str.GetValue())
						} else if path, ok := elem.(*ast.PathValue); ok {
							references = append(references, path.String())
						}
					}
				}
			} else if argName == "name" {
				if str, ok := arg.Value.(*ast.StringValue); ok {
					relationName = str.GetValue()
				}
			}
		}

		// Check for unnamed positional arguments
		if relationName == "" && len(attr.Arguments.Arguments) > 0 {
			firstArg := attr.Arguments.Arguments[0]
			if firstArg.Name == nil {
				if str, ok := firstArg.Value.(*ast.StringValue); ok {
					relationName = str.GetValue()
				}
			}
		}
	}

	return fields, references, relationName
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		first := strings.ToUpper(string(part[0]))
		rest := strings.ToLower(part[1:])
		result.WriteString(first + rest)
	}
	return result.String()
}
