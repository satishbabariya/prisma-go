// Package executor provides relation metadata extraction from schema AST.
package executor

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// ExtractRelationMetadata extracts relation metadata from PSL schema AST
func ExtractRelationMetadata(schemaAST *ast.SchemaAst, modelName string) (map[string]RelationMetadata, error) {
	relations := make(map[string]RelationMetadata)

	// Find the model
	var model *ast.Model
	for _, top := range schemaAST.Tops {
		if m := top.AsModel(); m != nil && m.Name.Name == modelName {
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
	for i := range model.Fields {
		fieldPtr := &model.Fields[i]
		if !isRelationField(fieldPtr) {
			continue
		}

		relationName := fieldPtr.Name.Name
		relatedModelName := getRelatedModelName(fieldPtr)

		if relatedModelName == "" {
			continue
		}

		// Find the related model
		var relatedModel *ast.Model
		for _, top := range schemaAST.Tops {
			if m := top.AsModel(); m != nil && m.Name.Name == relatedModelName {
				relatedModel = m
				break
			}
		}

		if relatedModel == nil {
			continue
		}

		relatedTableName := toSnakeCase(relatedModelName)

		// Parse @relation attribute
		fields, _, _ := parseRelationAttribute(fieldPtr)

		// Determine relation type
		isList := fieldPtr.Arity.IsList()
		isManyToMany := false
		junctionTable := ""
		junctionFKToSelf := ""
		junctionFKToOther := ""

		// Check if this is a many-to-many relation
		if isList {
			// Check if opposite field is also a list
			for j := range relatedModel.Fields {
				oppFieldPtr := &relatedModel.Fields[j]
				if isRelationField(oppFieldPtr) && getRelatedModelName(oppFieldPtr) == modelName {
					if oppFieldPtr.Arity.IsList() {
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
			IsManyToMany:       isManyToMany,
			JunctionTable:     junctionTable,
			JunctionFKToSelf:   junctionFKToSelf,
			JunctionFKToOther:  junctionFKToOther,
		}
	}

	return relations, nil
}

// isRelationField checks if a field is a relation field
func isRelationField(field *ast.Field) bool {
	// Check if field type is a model (not a scalar)
	typeName := field.FieldType.TypeName()
	scalarTypes := map[string]bool{
		"int": true, "bigint": true, "string": true, "boolean": true, "bool": true,
		"datetime": true, "float": true, "decimal": true, "json": true, "bytes": true,
		"date": true, "time": true, "timestamp": true,
	}
	return !scalarTypes[strings.ToLower(typeName)]
}

// getRelatedModelName extracts the related model name from a relation field
func getRelatedModelName(field *ast.Field) string {
	typeName := field.FieldType.TypeName()
	// Remove array brackets if present
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
