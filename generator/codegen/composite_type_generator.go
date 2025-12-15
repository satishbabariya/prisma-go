// Package codegen provides composite type code generation.
package codegen

import (
	"fmt"
	"strings"

	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// CompositeTypeInfo represents information about a composite type for code generation
type CompositeTypeInfo struct {
	Name   string
	Fields []FieldInfo
}

// GenerateCompositeTypesFromAST generates composite type information from the AST
func GenerateCompositeTypesFromAST(schemaAST *ast.SchemaAst) []CompositeTypeInfo {
	var compositeTypes []CompositeTypeInfo

	// Use helper method if available, or manual traversal
	for _, compositeType := range schemaAST.CompositeTypes() {
		compositeTypeInfo := CompositeTypeInfo{
			Name:   compositeType.Name.Name,
			Fields: []FieldInfo{},
		}

		for _, field := range compositeType.Fields {
			fieldInfo := generateFieldInfo(field, compositeType.Name.Name)
			compositeTypeInfo.Fields = append(compositeTypeInfo.Fields, fieldInfo)
		}

		compositeTypes = append(compositeTypes, compositeTypeInfo)
	}

	return compositeTypes
}

// GenerateCompositeTypeStruct generates Go struct code for a composite type
func GenerateCompositeTypeStruct(ct CompositeTypeInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// %s represents a composite type\n", ct.Name))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", ct.Name))

	for _, field := range ct.Fields {
		sb.WriteString(fmt.Sprintf("\t%s %s %s\n", field.GoName, field.GoType, field.Tags))
	}

	sb.WriteString("}\n\n")

	return sb.String()
}
