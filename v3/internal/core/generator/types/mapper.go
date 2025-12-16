// Package types provides Prisma to Go type mapping.
package types

import (
	"fmt"
	"strings"
)

// TypeMapper handles Prisma type to Go type conversion.
type TypeMapper struct {
	scalarMap map[string]string
}

// NewTypeMapper creates a new type mapper.
func NewTypeMapper() *TypeMapper {
	return &TypeMapper{
		scalarMap: map[string]string{
			"String":   "string",
			"Int":      "int64",
			"BigInt":   "int64",
			"Boolean":  "bool",
			"Float":    "float64",
			"Decimal":  "float64",
			"DateTime": "time.Time",
			"Json":     "json.RawMessage",
			"Bytes":    "[]byte",
		},
	}
}

// MapType maps a Prisma type to a Go type.
func (m *TypeMapper) MapType(prismaType string, isOptional bool, isArray bool) string {
	// Handle scalar types
	goType, ok := m.scalarMap[prismaType]
	if !ok {
		// Assume it's a model or enum type
		goType = prismaType
	}

	// Handle arrays
	if isArray {
		goType = "[]" + goType
	}

	// Handle optional (pointer for nullable)
	if isOptional && !isArray {
		goType = "*" + goType
	}

	return goType
}

// IsScalarType checks if a type is a Prisma scalar.
func (m *TypeMapper) IsScalarType(prismaType string) bool {
	_, ok := m.scalarMap[prismaType]
	return ok
}

// ToPascalCase converts a string to PascalCase.
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle snake_case and camelCase
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	result := strings.Join(parts, "")

	// Ensure first letter is uppercase
	if len(result) > 0 {
		result = strings.ToUpper(result[:1]) + result[1:]
	}

	return result
}

// ToCamelCase converts a string to camelCase.
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

// GetImportsForType returns the imports needed for a Go type.
func GetImportsForType(goType string) []string {
	var imports []string

	if strings.Contains(goType, "time.Time") {
		imports = append(imports, "time")
	}
	if strings.Contains(goType, "json.RawMessage") {
		imports = append(imports, "encoding/json")
	}

	return imports
}

// BuildStructTag builds a struct tag string.
func BuildStructTag(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}

	var parts []string
	for key, value := range tags {
		parts = append(parts, fmt.Sprintf(`%s:"%s"`, key, value))
	}

	return strings.Join(parts, " ")
}
