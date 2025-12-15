// Package converter converts Prisma schema AST to database schema format.
package converter

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// ConvertASTToDBSchema converts Prisma schema AST to DatabaseSchema
func ConvertASTToDBSchema(schemaAST *ast.SchemaAst, provider string) (*introspect.DatabaseSchema, error) {
	dbSchema := &introspect.DatabaseSchema{
		Tables:    []introspect.Table{},
		Enums:     []introspect.Enum{},
		Views:     []introspect.View{},
		Sequences: []introspect.Sequence{},
	}

	// Convert models to tables
	for _, model := range schemaAST.Models() {
		// AsModel check not needed as we iterate models directly
		table, err := convertModelToTable(model, schemaAST, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model %s: %w", model.Name.Name, err)
		}
		dbSchema.Tables = append(dbSchema.Tables, *table)
	}

	return dbSchema, nil
}

// convertModelToTable converts an AST model to a database table
func convertModelToTable(model *ast.Model, parsed *ast.SchemaAst, provider string) (*introspect.Table, error) {
	tableName := toSnakeCase(model.Name.Name)
	table := &introspect.Table{
		Name:        tableName,
		Schema:      "public", // Default schema
		Columns:     []introspect.Column{},
		PrimaryKey:  nil,
		Indexes:     []introspect.Index{},
		ForeignKeys: []introspect.ForeignKey{},
	}

	// Convert fields to columns
	var primaryKeyColumns []string
	scalarTypes := map[string]bool{
		"int": true, "bigint": true, "string": true, "boolean": true, "bool": true,
		"datetime": true, "float": true, "decimal": true, "json": true, "bytes": true,
	}

	for _, field := range model.Fields {
		// field is *ast.Field because model.Fields is []*ast.Field

		// Skip relation fields (they don't become columns)
		if field.Arity.IsList() {
			// List fields are relations, skip them
			continue
		}

		// Check if field type is a model name (relation)
		typeName := ""
		if field.Type != nil {
			typeName = strings.ToLower(field.Type.Name)
		}

		if !scalarTypes[typeName] {
			// This is likely a relation field (model name), skip it
			// But check if it has @relation attribute - if so, it's definitely a relation
			hasRelationAttr := false
			for _, attr := range field.Attributes {
				if attr.Name.Name == "relation" {
					hasRelationAttr = true
					break
				}
			}
			if hasRelationAttr {
				continue
			}
			// If no @relation but not a scalar, still skip (it's a relation field)
			continue
		}

		column, err := convertFieldToColumn(field, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", field.Name.Name, err)
		}
		table.Columns = append(table.Columns, *column)

		// Check for primary key
		if hasAttribute(field, "id") {
			primaryKeyColumns = append(primaryKeyColumns, column.Name)
		}

		// Check for unique index
		if hasAttribute(field, "unique") {
			indexName := fmt.Sprintf("%s_%s_unique", tableName, column.Name)
			table.Indexes = append(table.Indexes, introspect.Index{
				Name:     indexName,
				Columns:  []string{column.Name},
				IsUnique: true,
			})
		}
	}

	// Set primary key
	if len(primaryKeyColumns) > 0 {
		table.PrimaryKey = &introspect.PrimaryKey{
			Name:    fmt.Sprintf("%s_pkey", tableName),
			Columns: primaryKeyColumns,
		}
	}

	// Extract foreign keys from relation attributes
	foreignKeys := extractForeignKeys(model, tableName)
	table.ForeignKeys = append(table.ForeignKeys, foreignKeys...)

	return table, nil
}

// convertFieldToColumn converts an AST field to a database column
func convertFieldToColumn(field *ast.Field, provider string) (*introspect.Column, error) {
	column := &introspect.Column{
		Name:          toSnakeCase(field.Name.Name),
		Nullable:      field.Arity.IsOptional(),
		DefaultValue:  nil,
		AutoIncrement: false,
	}

	typeName := ""
	if field.Type != nil {
		typeName = field.Type.Name
	}

	// Convert Prisma type to database type
	dbType, err := mapPrismaTypeToDB(typeName, provider)
	if err != nil {
		return nil, err
	}
	column.Type = dbType

	// Check for auto-increment
	if hasAttribute(field, "default") {
		if isAutoIncrement(field) {
			column.AutoIncrement = true
		} else if defaultValue := extractDefaultValue(field); defaultValue != nil {
			column.DefaultValue = defaultValue
		}
	}

	return column, nil
}

// mapPrismaTypeToDB maps Prisma types to database types
func mapPrismaTypeToDB(prismaType string, provider string) (string, error) {
	switch strings.ToLower(prismaType) {
	case "int":
		switch provider {
		case "postgresql", "postgres":
			return "INTEGER", nil
		case "mysql":
			return "INT", nil
		case "sqlite":
			return "INTEGER", nil
		}
	case "string":
		switch provider {
		case "postgresql", "postgres":
			return "TEXT", nil
		case "mysql":
			return "VARCHAR(255)", nil
		case "sqlite":
			return "TEXT", nil
		}
	case "boolean", "bool":
		switch provider {
		case "postgresql", "postgres":
			return "BOOLEAN", nil
		case "mysql":
			return "TINYINT(1)", nil
		case "sqlite":
			return "INTEGER", nil
		}
	case "datetime":
		switch provider {
		case "postgresql", "postgres":
			return "TIMESTAMP", nil
		case "mysql":
			return "DATETIME", nil
		case "sqlite":
			return "DATETIME", nil
		}
	case "float":
		switch provider {
		case "postgresql", "postgres":
			return "DOUBLE PRECISION", nil
		case "mysql":
			return "DOUBLE", nil
		case "sqlite":
			return "REAL", nil
		}
	case "json":
		switch provider {
		case "postgresql", "postgres":
			return "JSONB", nil
		case "mysql":
			return "JSON", nil
		case "sqlite":
			return "TEXT", nil
		}
	default:
		// For unknown types, default to TEXT
		return "TEXT", nil
	}
	return "", fmt.Errorf("unsupported provider: %s", provider)
}

// extractForeignKeys extracts foreign keys from relation attributes
func extractForeignKeys(model *ast.Model, tableName string) []introspect.ForeignKey {
	var foreignKeys []introspect.ForeignKey

	// Build a map of relation fields to their foreign keys
	relationMap := make(map[string]string) // relation field name -> foreign key field name

	for _, field := range model.Fields {
		// Look for @relation attributes to find foreign keys
		for _, attr := range field.Attributes {
			if attr.Name.Name == "relation" {
				// Extract fields argument (the foreign key field)
				fieldsArg := findAttributeArgument(attr, "fields")
				if fieldsArg != nil {
					fieldNames := extractStringArray(fieldsArg)
					if len(fieldNames) > 0 {
						relationMap[field.Name.Name] = fieldNames[0]
					}
				}
			}
		}
	}

	// Now look for foreign key fields (ending in Id or _id) that match relations
	for _, field := range model.Fields {
		// Skip if it's a relation field itself
		if field.Arity.IsList() {
			continue
		}

		typeName := ""
		if field.Type != nil {
			typeName = field.Type.Name
		}

		scalarTypes := map[string]bool{
			"int": true, "bigint": true, "string": true, "boolean": true, "bool": true,
			"datetime": true, "float": true, "decimal": true, "json": true, "bytes": true,
		}
		if !scalarTypes[strings.ToLower(typeName)] {
			continue
		}

		// Look for fields that might be foreign keys (ending in Id or _id)
		fieldNameLower := strings.ToLower(field.Name.Name)
		if strings.HasSuffix(fieldNameLower, "id") || strings.HasSuffix(fieldNameLower, "_id") {
			// Check if this field is used in a relation
			// For now, skip creating foreign keys automatically
			// They should be created explicitly via @relation attributes
		}
	}

	return foreignKeys
}

// Helper functions

func hasAttribute(field *ast.Field, attrName string) bool {
	for _, attr := range field.Attributes {
		if attr.Name.Name == attrName {
			return true
		}
	}
	return false
}

func isAutoIncrement(field *ast.Field) bool {
	for _, attr := range field.Attributes {
		if attr.Name.Name == "default" {
			if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
				// Check if default value is autoincrement()
				if funcCall, ok := attr.Arguments.Arguments[0].Value.(*ast.FunctionCall); ok {
					if funcCall.Name == "autoincrement" {
						return true
					}
				}
			}
		}
	}
	return false
}

func extractDefaultValue(field *ast.Field) *string {
	for _, attr := range field.Attributes {
		if attr.Name.Name == "default" {
			if attr.Arguments != nil && len(attr.Arguments.Arguments) > 0 {
				arg := attr.Arguments.Arguments[0]
				if strLit, ok := arg.Value.(*ast.StringValue); ok {
					val := strLit.GetValue()
					return &val
				}
				if numVal, ok := arg.Value.(*ast.NumericValue); ok {
					// NumericValue stores parsed number as string in Value? No, float64 or int64?
					// AST definition: type NumericValue struct { Value string } usually for raw token
					val := numVal.Value
					return &val
				}
				if constVal, ok := arg.Value.(*ast.ConstantValue); ok {
					val := constVal.Value
					return &val
				}
			}
		}
	}
	return nil
}

func findAttributeArgument(attr *ast.Attribute, argName string) ast.Expression {
	if attr.Arguments == nil {
		return nil
	}
	for _, arg := range attr.Arguments.Arguments {
		if arg.Name != nil && arg.Name.Name == argName {
			return arg.Value
		}
	}
	return nil
}

func extractStringArray(expr ast.Expression) []string {
	if arr, ok := expr.(*ast.ArrayExpression); ok {
		var result []string
		for _, elem := range arr.Elements {
			if strLit, ok := elem.(*ast.StringValue); ok {
				result = append(result, strLit.GetValue())
			} else if ident, ok := elem.(*ast.ConstantValue); ok {
				result = append(result, ident.Value)
			}
		}
		return result
	}
	return []string{}
}

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
