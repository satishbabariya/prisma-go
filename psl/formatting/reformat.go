package formatting

import (
	"github.com/satishbabariya/prisma-go/psl/parsing"

	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

	"fmt"
	"strings"
)

// Reformat reformats a Prisma schema string.
// indentWidth specifies the number of spaces for indentation (defaults to 2 if 0).
func Reformat(input string, indentWidth int) (string, error) {
	// Parse the input first
	ast, diags := parsing.ParseSchema(input)
	if diags.HasErrors() {
		return "", fmt.Errorf("cannot reformat invalid schema: %v", diags.ToResult())
	}

	// Default indent width to 2 if not specified
	if indentWidth == 0 {
		indentWidth = 2
	}

	// Render the AST back to formatted string
	return renderSchema(ast, indentWidth), nil
}

// renderSchema renders a ast.SchemaAst to formatted string.
func renderSchema(schema *ast.SchemaAst, indentWidth int) string {
	var builder strings.Builder

	for i, top := range schema.Tops {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(renderTop(top, indentWidth))
	}

	return builder.String()
}

// renderTop renders a top-level element to string.
func renderTop(top ast.Top, indentWidth int) string {
	switch {
	case top.AsModel() != nil:
		return renderModel(top.AsModel(), indentWidth)
	case top.AsEnum() != nil:
		return renderEnum(top.AsEnum(), indentWidth)
	case top.AsSource() != nil:
		return renderSourceConfig(top.AsSource(), indentWidth)
	case top.AsGenerator() != nil:
		return renderGeneratorConfig(top.AsGenerator(), indentWidth)
	case top.AsCompositeType() != nil:
		return renderCompositeType(top.AsCompositeType(), indentWidth)
	default:
		return ""
	}
}

// renderModel renders a model to string.
func renderModel(model *ast.Model, indentWidth int) string {
	var builder strings.Builder
	indent := strings.Repeat(" ", indentWidth)

	builder.WriteString(fmt.Sprintf("model %s {\n", model.Name.Name))

	for i, field := range model.Fields {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(indent)
		builder.WriteString(renderField(&field))
	}

	if len(model.Attributes) > 0 {
		if len(model.Fields) > 0 {
			builder.WriteString("\n")
		}
		for i, attr := range model.Attributes {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(indent)
			builder.WriteString(renderAttribute(&attr))
		}
	}

	builder.WriteString("\n}")
	return builder.String()
}

// renderEnum renders an enum to string.
func renderEnum(enum *ast.Enum, indentWidth int) string {
	var builder strings.Builder
	indent := strings.Repeat(" ", indentWidth)

	builder.WriteString(fmt.Sprintf("enum %s {\n", enum.Name.Name))

	for i, value := range enum.Values {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(indent)
		builder.WriteString(value.Name.Name)

		if len(value.Attributes) > 0 {
			for _, attr := range value.Attributes {
				builder.WriteString(" ")
				builder.WriteString(renderAttribute(&attr))
			}
		}
	}

	builder.WriteString("\n}")
	return builder.String()
}

// renderSourceConfig renders a datasource config to string.
func renderSourceConfig(source *ast.SourceConfig, indentWidth int) string {
	var builder strings.Builder
	indent := strings.Repeat(" ", indentWidth)

	builder.WriteString(fmt.Sprintf("datasource %s {\n", source.Name.Name))

	for i, prop := range source.Properties {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(indent)
		builder.WriteString(renderConfigProperty(&prop))
	}

	builder.WriteString("\n}")
	return builder.String()
}

// renderGeneratorConfig renders a generator config to string.
func renderGeneratorConfig(generator *ast.GeneratorConfig, indentWidth int) string {
	var builder strings.Builder
	indent := strings.Repeat(" ", indentWidth)

	builder.WriteString(fmt.Sprintf("generator %s {\n", generator.Name.Name))

	for i, prop := range generator.Properties {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(indent)
		builder.WriteString(renderConfigProperty(&prop))
	}

	builder.WriteString("\n}")
	return builder.String()
}

// renderCompositeType renders a composite type to string.
func renderCompositeType(composite *ast.CompositeType, indentWidth int) string {
	var builder strings.Builder
	indent := strings.Repeat(" ", indentWidth)

	builder.WriteString(fmt.Sprintf("type %s {\n", composite.Name.Name))

	for i, field := range composite.Fields {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(indent)
		builder.WriteString(renderField(&field))
	}

	builder.WriteString("\n}")
	return builder.String()
}

// renderField renders a field to string.
func renderField(field *ast.Field) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s %s", field.Name.Name, renderFieldType(&field.FieldType, field.Arity)))

	for _, attr := range field.Attributes {
		builder.WriteString(" ")
		builder.WriteString(renderAttribute(&attr))
	}

	return builder.String()
}

// renderFieldType renders a field type to string.
func renderFieldType(fieldType *ast.FieldType, arity ast.FieldArity) string {
	var builder strings.Builder

	builder.WriteString(fieldType.Name())

	if arity.IsList() {
		builder.WriteString("[]")
	} else if arity.IsOptional() {
		builder.WriteString("?")
	}

	return builder.String()
}

// renderAttribute renders an attribute to string.
func renderAttribute(attr *ast.Attribute) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("@%s", attr.Name.Name))

	hasArgs := len(attr.Arguments.Arguments) > 0 || len(attr.Arguments.EmptyArguments) > 0
	if hasArgs {
		builder.WriteString("(")

		// Render regular arguments
		for i, arg := range attr.Arguments.Arguments {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(renderArgument(&arg))
		}

		// Render empty arguments (for autocompletion/invalid syntax)
		if len(attr.Arguments.EmptyArguments) > 0 {
			if len(attr.Arguments.Arguments) > 0 {
				builder.WriteString(", ")
			}
			for i, emptyArg := range attr.Arguments.EmptyArguments {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(fmt.Sprintf("%s: ", emptyArg.Name.Name))
			}
		}

		// Render trailing comma if present
		if attr.Arguments.TrailingComma != nil {
			if hasArgs {
				builder.WriteString(", ")
			}
		}

		builder.WriteString(")")
	}

	return builder.String()
}

// renderArgument renders an argument to string.
func renderArgument(arg *ast.Argument) string {
	var builder strings.Builder

	if arg.Name != nil {
		builder.WriteString(fmt.Sprintf("%s: ", arg.Name.Name))
	}

	builder.WriteString(renderExpression(arg.Value))

	return builder.String()
}

// renderConfigProperty renders a config property to string.
func renderConfigProperty(prop *ast.ConfigBlockProperty) string {
	if prop.Value == nil {
		return prop.Name.Name
	}
	return fmt.Sprintf("%s = %s", prop.Name.Name, renderExpression(prop.Value))
}

// renderExpression renders an expression to string.
func renderExpression(expr ast.Expression) string {
	switch e := expr.(type) {
	case ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", e.Value)
	case ast.IntLiteral:
		return fmt.Sprintf("%d", e.Value)
	case ast.FloatLiteral:
		return fmt.Sprintf("%g", e.Value)
	case ast.BooleanLiteral:
		return fmt.Sprintf("%t", e.Value)
	case ast.ArrayLiteral:
		var builder strings.Builder
		builder.WriteString("[")
		for i, elem := range e.Elements {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(renderExpression(elem))
		}
		builder.WriteString("]")
		return builder.String()
	case ast.FunctionCall:
		var builder strings.Builder
		builder.WriteString(e.Name.Name)
		builder.WriteString("(")
		for i, expr := range e.Arguments {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(renderExpression(expr))
		}
		builder.WriteString(")")
		return builder.String()
	case ast.Identifier:
		return e.Name
	default:
		return ""
	}
}
