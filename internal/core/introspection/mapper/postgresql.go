// Package mapper provides type mapping from database types to Prisma types.
package mapper

import (
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/domain"
)

// PostgreSQLTypeMapper maps PostgreSQL types to Prisma types.
type PostgreSQLTypeMapper struct{}

// NewPostgreSQLTypeMapper creates a new PostgreSQL type mapper.
func NewPostgreSQLTypeMapper() *PostgreSQLTypeMapper {
	return &PostgreSQLTypeMapper{}
}

// MapToPrismaType converts a PostgreSQL type to Prisma type.
func (m *PostgreSQLTypeMapper) MapToPrismaType(dbType string, isNullable bool) string {
	// Normalize type (remove precision, scale, etc.)
	baseType := strings.ToLower(normalizeType(dbType))

	prismaType := m.getBasePrismaType(baseType)

	// Handle nullable types
	if isNullable {
		return prismaType + "?"
	}
	return prismaType
}

func (m *PostgreSQLTypeMapper) getBasePrismaType(dbType string) string {
	switch dbType {
	// String types
	case "character varying", "varchar", "character", "char", "text":
		return "String"
	case "uuid":
		return "String" // Prisma uses String for UUIDs with @db.Uuid

	// Integer types
	case "smallint", "integer", "int", "int4":
		return "Int"
	case "bigint", "int8", "bigserial", "serial8":
		return "BigInt"
	case "serial", "serial4":
		return "Int"

	// Decimal/Float types
	case "real", "float4":
		return "Float"
	case "double precision", "float8":
		return "Float"
	case "numeric", "decimal":
		return "Decimal"

	// Boolean
	case "boolean", "bool":
		return "Boolean"

	// Date/Time types
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		return "DateTime"
	case "date":
		return "DateTime"
	case "time", "time without time zone", "time with time zone":
		return "DateTime"

	// JSON types
	case "json", "jsonb":
		return "Json"

	// Binary types
	case "bytea":
		return "Bytes"

	// Default: treat as String
	default:
		return "String"
	}
}

// MapToGoType converts a PostgreSQL type to Go type.
func (m *PostgreSQLTypeMapper) MapToGoType(dbType string, isNullable bool) string {
	baseType := strings.ToLower(normalizeType(dbType))

	goType := m.getBaseGoType(baseType)

	// Handle nullable types with pointers
	if isNullable && !strings.HasPrefix(goType, "*") {
		return "*" + goType
	}
	return goType
}

func (m *PostgreSQLTypeMapper) getBaseGoType(dbType string) string {
	switch dbType {
	case "character varying", "varchar", "character", "char", "text", "uuid":
		return "string"
	case "smallint", "integer", "int", "int4", "serial", "serial4":
		return "int"
	case "bigint", "int8", "bigserial", "serial8":
		return "int64"
	case "real", "float4", "double precision", "float8":
		return "float64"
	case "numeric", "decimal":
		return "float64"
	case "boolean", "bool":
		return "bool"
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz", "date":
		return "time.Time"
	case "json", "jsonb":
		return "interface{}"
	case "bytea":
		return "[]byte"
	default:
		return "interface{}"
	}
}

// normalizeType removes precision, scale, and other modifiers from type name.
func normalizeType(dbType string) string {
	// Remove anything in parentheses (e.g., "varchar(255)" -> "varchar")
	if idx := strings.Index(dbType, "("); idx > 0 {
		return strings.TrimSpace(dbType[:idx])
	}
	return strings.TrimSpace(dbType)
}

// Ensure PostgreSQLTypeMapper implements domain.TypeMapper
var _ domain.TypeMapper = (*PostgreSQLTypeMapper)(nil)
