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
}

// FieldInfo represents information about a field
type FieldInfo struct {
	Name   string
	GoName string
	GoType string
	Tags   string
	IsID   bool
	IsUnique bool
}

// GenerateModelsFromAST generates model information from the AST
func GenerateModelsFromAST(schemaAST *ast.SchemaAst) []ModelInfo {
	var models []ModelInfo

	for _, top := range schemaAST.Tops {
		if model := top.AsModel(); model != nil {
			modelInfo := ModelInfo{
				Name:      model.Name.Name,
				TableName: toSnakeCase(model.Name.Name),
				Fields:    []FieldInfo{},
			}

			for _, field := range model.Fields {
				fieldInfo := generateFieldInfo(&field)
				modelInfo.Fields = append(modelInfo.Fields, fieldInfo)
			}

			models = append(models, modelInfo)
		}
	}

	return models
}

func generateFieldInfo(field *ast.Field) FieldInfo {
	fieldName := field.Name.Name
	goType := mapPrismaTypeToGo(&field.FieldType)
	
	// Check for optional/list fields using Arity
	switch field.Arity {
	case ast.Optional:
		goType = "*" + goType
	case ast.List:
		goType = "[]" + goType
	}

	tags := generateFieldTags(field)
	isID := hasAttribute(field, "id")
	isUnique := hasAttribute(field, "unique")

	return FieldInfo{
		Name:     fieldName,
		GoName:   toPascalCase(fieldName),
		GoType:   goType,
		Tags:     tags,
		IsID:     isID,
		IsUnique: isUnique,
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

func generateFieldTags(field *ast.Field) string {
	tags := []string{}
	
	// JSON tag
	jsonTag := fmt.Sprintf(`json:"%s"`, toSnakeCase(field.Name.Name))
	tags = append(tags, jsonTag)

	// DB tag
	dbTag := fmt.Sprintf(`db:"%s"`, toSnakeCase(field.Name.Name))
	tags = append(tags, dbTag)

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
	// Simple implementation - capitalize first letter
	return strings.ToUpper(s[:1]) + s[1:]
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

