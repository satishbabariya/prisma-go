package codegen

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// ModelInfo represents information about a model for code generation
type ModelInfo struct {
	Name      string
	TableName string
	Fields    []FieldInfo
	Relations []RelationInfo // Relations from this model
}

// RelationInfo represents a relation between models
type RelationInfo struct {
	FieldName       string // Name of the relation field (e.g., "posts", "author")
	RelatedModel    string // Name of the related model (e.g., "Post", "User")
	IsList          bool   // true if one-to-many or many-to-many
	ForeignKey      string // Foreign key field name (e.g., "authorId")
	ForeignKeyTable string // Table name of the model with the foreign key
	LocalKey        string // Local key field name (usually "id")
}

// FieldInfo represents information about a field
type FieldInfo struct {
	Name         string
	GoName       string
	GoType       string
	Tags         string
	IsID         bool
	IsUnique     bool
	IsRelation   bool
	RelationTo   string // Model name if this is a relation field
	IsList       bool   // true if relation is a list (one-to-many or many-to-many)
	IsForeignKey bool   // true if this is a foreign key field (e.g., authorId)
	ForeignKeyTo string // Model name this foreign key references
}

// GenerateModelsFromAST generates model information from the AST
func GenerateModelsFromAST(schemaAST *ast.SchemaAst) []ModelInfo {
	var models []ModelInfo
	modelMap := make(map[string]*ModelInfo)

	// First pass: create all models (including views)
	for _, top := range schemaAST.Tops {
		if model := top.AsModel(); model != nil {
			// Extract table name from @@map attribute if present
			tableName := extractTableNameFromModel(model)
			
			modelInfo := ModelInfo{
				Name:      model.Name.Name,
				TableName: tableName,
				Fields:    []FieldInfo{},
				Relations: []RelationInfo{},
			}

			for _, field := range model.Fields {
				fieldInfo := generateFieldInfo(&field, model.Name.Name)
				modelInfo.Fields = append(modelInfo.Fields, fieldInfo)
			}

			models = append(models, modelInfo)
			modelMap[model.Name.Name] = &models[len(models)-1]
		}
	}

	// Also generate composite types as structs (foundation for future expansion)
	// Composite types are already parsed and validated, but code generation
	// would need to be added to generate Go structs for them
	for _, top := range schemaAST.Tops {
		if compositeType := top.AsCompositeType(); compositeType != nil {
			// Foundation: Composite types can be used as field types
			// Full implementation would generate Go structs for composite types
			_ = compositeType // Mark as used for now
		}
	}

	// Second pass: detect relations and foreign keys from AST
	astModelMap := make(map[string]*ast.Model)
	for _, top := range schemaAST.Tops {
		if astModel := top.AsModel(); astModel != nil {
			astModelMap[astModel.Name.Name] = astModel
		}
	}

	for i := range models {
		model := &models[i]
		astModel := astModelMap[model.Name]
		if astModel == nil {
			continue
		}

		for j := range model.Fields {
			field := &model.Fields[j]

			// If this is a relation field, find the foreign key
			if field.IsRelation {
				relatedModel := modelMap[field.RelationTo]
				relatedASTModel := astModelMap[field.RelationTo]

				if relatedModel != nil {
					// Find the corresponding AST field
					var astRelationField *ast.Field
					for k := range astModel.Fields {
						if astModel.Fields[k].Name.Name == field.Name {
							astRelationField = &astModel.Fields[k]
							break
						}
					}

					// Create relation info
					relation := RelationInfo{
						FieldName:    field.Name,
						RelatedModel: field.RelationTo,
						IsList:       field.IsList,
					}

					// Try to parse @relation attribute first
					if astRelationField != nil {
						fkField, refField, err := findForeignKeyFromRelation(astModel, astRelationField)
						if err == nil && fkField != "" {
							// Found foreign key from @relation attribute
							if field.IsList {
								// One-to-many: foreign key is on the related model
								// Need to find the foreign key field in the related model
								if relatedASTModel != nil {
									for k := range relatedASTModel.Fields {
										if relatedASTModel.Fields[k].Name.Name == fkField {
											relation.ForeignKey = fkField
											relation.ForeignKeyTable = relatedModel.TableName
											relation.LocalKey = refField
											break
										}
									}
								}
							} else {
								// Many-to-one: foreign key is on this model
								relation.ForeignKey = fkField
								relation.ForeignKeyTable = model.TableName
								relation.LocalKey = refField

								// Mark the foreign key field
								for k := range model.Fields {
									if model.Fields[k].Name == fkField {
										model.Fields[k].IsForeignKey = true
										model.Fields[k].ForeignKeyTo = field.RelationTo
										break
									}
								}
							}
						}
					}

					// Fallback: pattern matching if @relation parsing failed
					if relation.ForeignKey == "" {
						var foreignKeyField *FieldInfo
						localKeyField := "id" // default

						// Check current model for foreign keys pointing to related model (many-to-one)
						if !field.IsList {
							for k := range model.Fields {
								checkField := &model.Fields[k]
								if (strings.HasSuffix(strings.ToLower(checkField.Name), "id")) &&
									!checkField.IsID && !checkField.IsRelation {
									// Check if this matches the pattern for foreign key
									expectedFK := toSnakeCase(field.RelationTo) + "_id"
									if toSnakeCase(checkField.Name) == expectedFK {
										foreignKeyField = checkField
										foreignKeyField.IsForeignKey = true
										foreignKeyField.ForeignKeyTo = field.RelationTo
										break
									}
								}
							}
						}

						if foreignKeyField != nil {
							if field.IsList {
								// One-to-many: foreign key is on the related model
								relation.ForeignKey = foreignKeyField.Name
								relation.ForeignKeyTable = relatedModel.TableName
								relation.LocalKey = localKeyField
							} else {
								// Many-to-one: foreign key is on this model
								relation.ForeignKey = foreignKeyField.Name
								relation.ForeignKeyTable = model.TableName
								relation.LocalKey = localKeyField
							}
						} else if field.IsList && relatedASTModel != nil {
							// One-to-many: foreign key should be on related model
							// Look for foreign key in related model
							for k := range relatedASTModel.Fields {
								checkASTField := &relatedASTModel.Fields[k]
								if (strings.HasSuffix(strings.ToLower(checkASTField.Name.Name), "id")) &&
									!hasAttribute(checkASTField, "id") {
									expectedFK := toSnakeCase(model.Name) + "_id"
									if toSnakeCase(checkASTField.Name.Name) == expectedFK {
										relation.ForeignKey = checkASTField.Name.Name
										relation.ForeignKeyTable = relatedModel.TableName
										relation.LocalKey = localKeyField
										break
									}
								}
							}
						}
					}

					if relation.ForeignKey != "" {
						model.Relations = append(model.Relations, relation)
					}
				}
			}
		}
	}

	return models
}

func generateFieldInfo(field *ast.Field, modelName string) FieldInfo {
	fieldName := field.Name.Name
	typeName := ""
	if field.FieldType.Type != nil {
		typeName = field.FieldType.Type.Name()
	}

	// Check if this is a relation field (type is a model name, not a scalar)
	isRelation := false
	relationTo := ""
	isList := false

	// Check if field type is a model (relation) by checking if it's not a scalar type
	scalarTypes := map[string]bool{
		"Int": true, "BigInt": true, "String": true, "Boolean": true,
		"DateTime": true, "Float": true, "Decimal": true, "Json": true, "Bytes": true,
	}

	if typeName != "" && !scalarTypes[typeName] {
		// This might be a relation field (model name)
		isRelation = true
		relationTo = typeName
	}

	goType := mapPrismaTypeToGo(&field.FieldType)

	// Check for optional/list fields using Arity
	switch field.Arity {
	case ast.Optional:
		if !isRelation {
			goType = "*" + goType
		} else {
			// For relations, optional means pointer to the model
			goType = "*" + goType
		}
	case ast.List:
		goType = "[]" + goType
		if isRelation {
			isList = true
		}
	}

	tags := generateFieldTags(field, isRelation)
	isID := hasAttribute(field, "id")
	isUnique := hasAttribute(field, "unique")

	return FieldInfo{
		Name:       fieldName,
		GoName:     toPascalCase(fieldName),
		GoType:     goType,
		Tags:       tags,
		IsID:       isID,
		IsUnique:   isUnique,
		IsRelation: isRelation,
		RelationTo: relationTo,
		IsList:     isList,
	}
}

func mapPrismaTypeToGo(fieldType *ast.FieldType) string {
	if fieldType.Type == nil {
		return "interface{}"
	}

	typeName := fieldType.Type.Name()

	switch typeName {
	case "Int":
		return "int"
	case "BigInt":
		return "int64"
	case "String":
		return "string"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "time.Time"
	case "Float":
		return "float64"
	case "Decimal":
		return "string" // Use string for now, can be improved with decimal library
	case "Json":
		return "interface{}"
	case "Bytes":
		return "[]byte"
	default:
		// For custom types (enums, other models), use the type name as-is
		return typeName
	}
}

func generateFieldTags(field *ast.Field, isRelation bool) string {
	tags := []string{}

	// JSON tag
	jsonTag := fmt.Sprintf(`json:"%s"`, toSnakeCase(field.Name.Name))
	tags = append(tags, jsonTag)

	// DB tag - only for non-relation fields (relations are not database columns)
	if !isRelation {
		dbTag := fmt.Sprintf(`db:"%s"`, toSnakeCase(field.Name.Name))
		tags = append(tags, dbTag)
	}

	if len(tags) > 0 {
		return "`" + strings.Join(tags, " ") + "`"
	}

	return ""
}

func hasAttribute(field *ast.Field, attrName string) bool {
	for _, attr := range field.Attributes {
		if attr.Name.Name == attrName {
			return true
		}
	}
	return false
}

func toPascalCase(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Split(s, "_")
	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// extractTableNameFromModel extracts the table name from a model's @@map attribute
// or falls back to snake_case of the model name
func extractTableNameFromModel(model *ast.Model) string {
	// Check for @@map attribute
	for _, attr := range model.Attributes {
		if attr.Name.Name == "map" {
			// Check for named argument "name"
			for _, arg := range attr.Arguments.Arguments {
				if arg.Name != nil && arg.Name.Name == "name" {
					if strLit, ok := arg.Value.(ast.StringLiteral); ok {
						return strLit.Value
					}
				}
			}
			// Check for unnamed first argument (positional)
			if len(attr.Arguments.Arguments) > 0 {
				firstArg := attr.Arguments.Arguments[0]
				if firstArg.Name == nil {
					if strLit, ok := firstArg.Value.(ast.StringLiteral); ok {
						return strLit.Value
					}
				}
			}
		}
	}
	// Fall back to snake_case of model name
	return toSnakeCase(model.Name.Name)
}
