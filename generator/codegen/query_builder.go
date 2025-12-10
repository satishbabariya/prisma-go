package codegen

import (
	"fmt"
	"strings"
)

// FieldQueryBuilder generates query builder methods for a field
type FieldQueryBuilder struct {
	FieldName string
	GoName    string
	GoType    string
	IsID      bool
	IsUnique  bool
}

// GenerateQueryBuilderMethods generates query builder methods for a model
func GenerateQueryBuilderMethods(model ModelInfo) string {
	var sb strings.Builder

	modelName := model.Name

	// Generate Where methods for each field
	for _, field := range model.Fields {
		fieldName := field.Name
		goFieldName := field.GoName
		dbColumnName := toSnakeCase(fieldName)

		// Equals method
		sb.WriteString(fmt.Sprintf("// %sEquals filters records where %s equals the given value\n", goFieldName, fieldName))
		sb.WriteString(fmt.Sprintf("func (c *%sClient) %sEquals(value %s) *%sWhereBuilder {\n", modelName, goFieldName, field.GoType, modelName))
		sb.WriteString(fmt.Sprintf("\treturn c.Where().Equals(\"%s\", value)\n", dbColumnName))
		sb.WriteString("}\n\n")

		// NotEquals method
		sb.WriteString(fmt.Sprintf("// %sNotEquals filters records where %s does not equal the given value\n", goFieldName, fieldName))
		sb.WriteString(fmt.Sprintf("func (c *%sClient) %sNotEquals(value %s) *%sWhereBuilder {\n", modelName, goFieldName, field.GoType, modelName))
		sb.WriteString(fmt.Sprintf("\treturn c.Where().NotEquals(\"%s\", value)\n", dbColumnName))
		sb.WriteString("}\n\n")

		// For numeric types, add comparison methods
		if isNumericType(field.GoType) {
			// GreaterThan
			sb.WriteString(fmt.Sprintf("// %sGreaterThan filters records where %s is greater than the given value\n", goFieldName, fieldName))
			sb.WriteString(fmt.Sprintf("func (c *%sClient) %sGreaterThan(value %s) *%sWhereBuilder {\n", modelName, goFieldName, field.GoType, modelName))
			sb.WriteString(fmt.Sprintf("\treturn c.Where().GreaterThan(\"%s\", value)\n", dbColumnName))
			sb.WriteString("}\n\n")

			// LessThan
			sb.WriteString(fmt.Sprintf("// %sLessThan filters records where %s is less than the given value\n", goFieldName, fieldName))
			sb.WriteString(fmt.Sprintf("func (c *%sClient) %sLessThan(value %s) *%sWhereBuilder {\n", modelName, goFieldName, field.GoType, modelName))
			sb.WriteString(fmt.Sprintf("\treturn c.Where().LessThan(\"%s\", value)\n", dbColumnName))
			sb.WriteString("}\n\n")
		}

		// For string types, add LIKE methods
		if field.GoType == "string" || strings.HasPrefix(field.GoType, "*string") {
			// Contains (LIKE)
			sb.WriteString(fmt.Sprintf("// %sContains filters records where %s contains the given substring\n", goFieldName, fieldName))
			sb.WriteString(fmt.Sprintf("func (c *%sClient) %sContains(value string) *%sWhereBuilder {\n", modelName, goFieldName, modelName))
			sb.WriteString(fmt.Sprintf("\treturn c.Where().Like(\"%s\", \"%%\"+value+\"%%\")\n", dbColumnName))
			sb.WriteString("}\n\n")
		}

		// For optional fields, add IsNull/IsNotNull
		if strings.HasPrefix(field.GoType, "*") {
			// IsNull
			sb.WriteString(fmt.Sprintf("// %sIsNull filters records where %s is NULL\n", goFieldName, fieldName))
			sb.WriteString(fmt.Sprintf("func (c *%sClient) %sIsNull() *%sWhereBuilder {\n", modelName, goFieldName, modelName))
			sb.WriteString(fmt.Sprintf("\treturn c.Where().IsNull(\"%s\")\n", dbColumnName))
			sb.WriteString("}\n\n")

			// IsNotNull
			sb.WriteString(fmt.Sprintf("// %sIsNotNull filters records where %s is not NULL\n", goFieldName, fieldName))
			sb.WriteString(fmt.Sprintf("func (c *%sClient) %sIsNotNull() *%sWhereBuilder {\n", modelName, goFieldName, modelName))
			sb.WriteString(fmt.Sprintf("\treturn c.Where().IsNotNull(\"%s\")\n", dbColumnName))
			sb.WriteString("}\n\n")
		}
	}

	return sb.String()
}

// isNumericType checks if a type is numeric
func isNumericType(goType string) bool {
	numericTypes := []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64"}
	baseType := strings.TrimPrefix(goType, "*")
	for _, numType := range numericTypes {
		if baseType == numType {
			return true
		}
	}
	return false
}

