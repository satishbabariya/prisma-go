// Package formatter implements schema formatting.
package formatter

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// Formatter implements the SchemaFormatter interface.
type Formatter struct{}

// NewFormatter creates a new schema formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// Format formats a schema to a string.
func (f *Formatter) Format(ctx context.Context, schema *domain.Schema) (string, error) {
	var sb strings.Builder

	// Format datasources
	for _, ds := range schema.Datasources {
		sb.WriteString(f.formatDatasource(ds))
		sb.WriteString("\n\n")
	}

	// Format generators
	for _, gen := range schema.Generators {
		sb.WriteString(f.formatGenerator(gen))
		sb.WriteString("\n\n")
	}

	// Format models
	for _, model := range schema.Models {
		sb.WriteString(f.formatModel(model))
		sb.WriteString("\n\n")
	}

	// Format enums
	for _, enum := range schema.Enums {
		sb.WriteString(f.formatEnum(enum))
		sb.WriteString("\n\n")
	}

	return sb.String(), nil
}

// FormatFile formats a schema file in place.
func (f *Formatter) FormatFile(ctx context.Context, path string) error {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the schema (would use actual parser here)
	// For now, we'll write back the content as-is
	// TODO: Integrate with actual parser

	return os.WriteFile(path, content, 0644)
}

func (f *Formatter) formatDatasource(ds domain.Datasource) string {
	return fmt.Sprintf(`datasource %s {
  provider = "%s"
  url      = env("%s")
}`, ds.Name, ds.Provider, ds.URL)
}

func (f *Formatter) formatGenerator(gen domain.Generator) string {
	return fmt.Sprintf(`generator %s {
  provider = "%s"
  output   = "%s"
}`, gen.Name, gen.Provider, gen.Output)
}

func (f *Formatter) formatModel(model domain.Model) string {
	var sb strings.Builder

	// Model name and opening brace
	sb.WriteString(fmt.Sprintf("model %s {\n", model.Name))

	// Format fields
	for _, field := range model.Fields {
		sb.WriteString("  ")
		sb.WriteString(f.formatField(field))
		sb.WriteString("\n")
	}

	// Format indexes
	for _, index := range model.Indexes {
		sb.WriteString("  ")
		sb.WriteString(f.formatIndex(index))
		sb.WriteString("\n")
	}

	// Closing brace
	sb.WriteString("}")

	return sb.String()
}

func (f *Formatter) formatField(field domain.Field) string {
	var parts []string

	// Field name and type
	typeStr := field.Type.Name
	if field.IsList {
		typeStr += "[]"
	}
	if !field.IsRequired {
		typeStr += "?"
	}

	parts = append(parts, field.Name, typeStr)

	// Format attributes
	for _, attr := range field.Attributes {
		parts = append(parts, f.formatAttribute(attr))
	}

	return strings.Join(parts, " ")
}

func (f *Formatter) formatAttribute(attr domain.Attribute) string {
	if len(attr.Arguments) == 0 {
		return fmt.Sprintf("@%s", attr.Name)
	}

	var args []string
	for _, arg := range attr.Arguments {
		args = append(args, fmt.Sprintf("%v", arg))
	}

	return fmt.Sprintf("@%s(%s)", attr.Name, strings.Join(args, ", "))
}

func (f *Formatter) formatIndex(index domain.Index) string {
	fields := strings.Join(index.Fields, ", ")
	if index.Unique {
		return fmt.Sprintf("@@unique([%s])", fields)
	}
	return fmt.Sprintf("@@index([%s])", fields)
}

func (f *Formatter) formatEnum(enum domain.Enum) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("enum %s {\n", enum.Name))
	for _, value := range enum.Values {
		sb.WriteString(fmt.Sprintf("  %s\n", value))
	}
	sb.WriteString("}")

	return sb.String()
}

// Ensure Formatter implements SchemaFormatter interface.
var _ domain.SchemaFormatter = (*Formatter)(nil)
