// Package schema provides type mapping between Prisma and Go types.
package schema

import (
	"fmt"
	"strings"
)

// TypeMapper handles conversions between Prisma types and Go types.
type TypeMapper struct {
	// customMappings allows overriding default type mappings
	customMappings map[string]string
	// enumTypes tracks defined enum types
	enumTypes map[string]bool
}

// NewTypeMapper creates a new type mapper.
func NewTypeMapper() *TypeMapper {
	return &TypeMapper{
		customMappings: make(map[string]string),
		enumTypes:      make(map[string]bool),
	}
}

// RegisterEnum registers an enum type name.
func (tm *TypeMapper) RegisterEnum(enumName string) {
	tm.enumTypes[enumName] = true
}

// IsEnum checks if a type is a registered enum.
func (tm *TypeMapper) IsEnum(typeName string) bool {
	return tm.enumTypes[typeName]
}

// RegisterCustomMapping adds a custom type mapping.
func (tm *TypeMapper) RegisterCustomMapping(prismaType, goType string) {
	tm.customMappings[prismaType] = goType
}

// PrismaToGo converts a Prisma type to its Go equivalent.
// Parameters:
//   - prismaType: The Prisma type name (e.g., "String", "Int", "DateTime")
//   - isOptional: Whether the field is optional (nullable)
//   - isList: Whether the field is an array
func (tm *TypeMapper) PrismaToGo(prismaType string, isOptional, isList bool) string {
	// Check custom mappings first
	if customType, exists := tm.customMappings[prismaType]; exists {
		return tm.applyModifiers(customType, isOptional, isList)
	}

	// Check if it's a registered enum
	if tm.IsEnum(prismaType) {
		return tm.applyModifiers(prismaType, isOptional, isList)
	}

	// Map standard Prisma types to Go types
	var goType string
	switch prismaType {
	// Scalar types
	case "String":
		goType = "string"
	case "Boolean":
		goType = "bool"
	case "Int":
		goType = "int64"
	case "BigInt":
		goType = "int64"
	case "Float":
		goType = "float64"
	case "Decimal":
		goType = "decimal.Decimal" // Using shopspring/decimal
	case "DateTime":
		goType = "time.Time"
	case "Json":
		goType = "json.RawMessage"
	case "Bytes":
		goType = "[]byte"

	// Unsupported types - these are special Prisma types
	case "Unsupported":
		goType = "interface{}"

	default:
		// Assume it's a model reference
		goType = prismaType
	}

	return tm.applyModifiers(goType, isOptional, isList)
}

// applyModifiers applies nullable and list modifiers to a Go type.
func (tm *TypeMapper) applyModifiers(goType string, isOptional, isList bool) string {
	result := goType

	// Apply list modifier (slice)
	if isList {
		result = "[]" + result
	}

	// Apply optional modifier (pointer) for non-slice types
	// Note: Slices are already nullable in Go (can be nil)
	if isOptional && !isList {
		// Special handling for certain types that shouldn't be pointers
		switch goType {
		case "[]byte", "json.RawMessage":
			// These are already reference types
			return result
		default:
			result = "*" + result
		}
	}

	return result
}

// GoToPrisma converts a Go type back to its Prisma equivalent.
// This is useful for introspection.
func (tm *TypeMapper) GoToPrisma(goType string) (prismaType string, isOptional bool, isList bool, err error) {
	original := goType

	// Check for slice (list) modifier
	if strings.HasPrefix(goType, "[]") {
		isList = true
		goType = strings.TrimPrefix(goType, "[]")
	}

	// Check for pointer (optional) modifier
	if strings.HasPrefix(goType, "*") {
		isOptional = true
		goType = strings.TrimPrefix(goType, "*")
	}

	// Map Go types back to Prisma types
	switch goType {
	case "string":
		prismaType = "String"
	case "bool":
		prismaType = "Boolean"
	case "int", "int32", "int64":
		prismaType = "Int"
	case "float32", "float64":
		prismaType = "Float"
	case "time.Time":
		prismaType = "DateTime"
	case "json.RawMessage":
		prismaType = "Json"
	case "byte":
		// Special case: []byte should map to Bytes
		if isList {
			prismaType = "Bytes"
			isList = false // Not a list of bytes, but a Bytes field
		} else {
			return "", false, false, fmt.Errorf("unsupported Go type: %s", original)
		}
	case "decimal.Decimal":
		prismaType = "Decimal"
	default:
		// Check if it's a registered enum
		if tm.IsEnum(goType) {
			prismaType = goType
		} else {
			// Assume it's a model reference
			prismaType = goType
		}
	}

	return prismaType, isOptional, isList, nil
}

// GetDefaultValue returns the appropriate zero/default value for a Go type.
func (tm *TypeMapper) GetDefaultValue(goType string) string {
	// Check for pointers and slices first (these are nil by default)
	if strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "[]") {
		return "nil"
	}

	// Handle base types
	switch goType {
	case "string":
		return `""`
	case "bool":
		return "false"
	case "int", "int32", "int64", "float32", "float64":
		return "0"
	case "time.Time":
		return "time.Time{}"
	case "json.RawMessage":
		return "nil"
	case "decimal.Decimal":
		return "decimal.Zero"
	default:
		// For complex types (models, custom types)
		return "nil"
	}
}

// IsBuiltinType checks if a type is a Prisma built-in scalar type.
func IsBuiltinType(typeName string) bool {
	builtins := map[string]bool{
		"String":   true,
		"Boolean":  true,
		"Int":      true,
		"BigInt":   true,
		"Float":    true,
		"Decimal":  true,
		"DateTime": true,
		"Json":     true,
		"Bytes":    true,
	}
	return builtins[typeName]
}

// GetSQLType returns the SQL column type for a Prisma type.
// This is useful for migrations and schema generation.
func GetSQLType(prismaType string, dialect string) string {
	switch dialect {
	case "postgres", "postgresql":
		return getPrismaToPostgresType(prismaType)
	case "mysql":
		return getPrismaToMySQLType(prismaType)
	case "sqlite":
		return getPrismaToSQLiteType(prismaType)
	default:
		return getPrismaToPostgresType(prismaType) // Default to Postgres
	}
}

func getPrismaToPostgresType(prismaType string) string {
	switch prismaType {
	case "String":
		return "TEXT"
	case "Boolean":
		return "BOOLEAN"
	case "Int":
		return "INTEGER"
	case "BigInt":
		return "BIGINT"
	case "Float":
		return "DOUBLE PRECISION"
	case "Decimal":
		return "DECIMAL(65,30)"
	case "DateTime":
		return "TIMESTAMP(3)"
	case "Json":
		return "JSONB"
	case "Bytes":
		return "BYTEA"
	default:
		return "TEXT"
	}
}

func getPrismaToMySQLType(prismaType string) string {
	switch prismaType {
	case "String":
		return "VARCHAR(191)"
	case "Boolean":
		return "BOOLEAN"
	case "Int":
		return "INT"
	case "BigInt":
		return "BIGINT"
	case "Float":
		return "DOUBLE"
	case "Decimal":
		return "DECIMAL(65,30)"
	case "DateTime":
		return "DATETIME(3)"
	case "Json":
		return "JSON"
	case "Bytes":
		return "LONGBLOB"
	default:
		return "VARCHAR(191)"
	}
}

func getPrismaToSQLiteType(prismaType string) string {
	switch prismaType {
	case "String":
		return "TEXT"
	case "Boolean":
		return "INTEGER" // SQLite uses 0/1 for booleans
	case "Int", "BigInt":
		return "INTEGER"
	case "Float", "Decimal":
		return "REAL"
	case "DateTime":
		return "TEXT" // SQLite stores dates as TEXT or INTEGER
	case "Json":
		return "TEXT" // SQLite doesn't have native JSON
	case "Bytes":
		return "BLOB"
	default:
		return "TEXT"
	}
}
