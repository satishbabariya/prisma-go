// Package schemaast provides rendering functionality for Prisma schema ASTs.
package formatting

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

	"fmt"
	"strings"
)

// Renderer renders a Prisma schema AST to a string.
type Renderer struct {
	builder strings.Builder
}

// NewRenderer creates a new renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Render renders the entire schema AST to a string.
func (r *Renderer) Render(ast *ast.SchemaAst) string {
	r.builder.Reset()

	for _, top := range ast.Tops {
		r.renderTop(top)
		r.builder.WriteString("\n\n")
	}

	return r.builder.String()
}

// renderTop renders a top-level element.
func (r *Renderer) renderTop(top ast.Top) {
	switch t := top.(type) {
	case *ast.Model:
		r.renderModel(t)
	case *ast.Enum:
		r.renderEnum(t)
	case *ast.CompositeType:
		r.renderCompositeType(t)
	case *ast.SourceConfig:
		r.renderSourceConfig(t)
	case *ast.GeneratorConfig:
		r.renderGeneratorConfig(t)
	}
}

// renderModel renders a model.
func (r *Renderer) renderModel(model *ast.Model) {
	r.builder.WriteString("model ")
	r.renderIdentifier(&model.Name)
	r.builder.WriteString(" {\n")

	// Render fields
	for _, field := range model.Fields {
		r.renderField(&field, "  ")
	}

	// Render attributes
	for _, attr := range model.Attributes {
		r.renderAttribute(&attr, "  ")
	}

	r.builder.WriteString("}")
}

// renderEnum renders an enum.
func (r *Renderer) renderEnum(enum *ast.Enum) {
	r.builder.WriteString("enum ")
	r.renderIdentifier(&enum.Name)
	r.builder.WriteString(" {\n")

	// Render values
	for _, value := range enum.Values {
		r.renderEnumValue(&value, "  ")
	}

	// Render attributes
	for _, attr := range enum.Attributes {
		r.renderAttribute(&attr, "  ")
	}

	r.builder.WriteString("}")
}

// renderCompositeType renders a composite type.
func (r *Renderer) renderCompositeType(compositeType *ast.CompositeType) {
	r.builder.WriteString("type ")
	r.renderIdentifier(&compositeType.Name)
	r.builder.WriteString(" {\n")

	// Render fields
	for _, field := range compositeType.Fields {
		r.renderField(&field, "  ")
	}

	// Render attributes
	for _, attr := range compositeType.Attributes {
		r.renderAttribute(&attr, "  ")
	}

	r.builder.WriteString("}")
}

// renderSourceConfig renders a datasource configuration.
func (r *Renderer) renderSourceConfig(source *ast.SourceConfig) {
	r.builder.WriteString("datasource ")
	r.renderIdentifier(&source.Name)
	r.builder.WriteString(" {\n")

	for _, prop := range source.Properties {
		r.renderConfigProperty(&prop, "  ")
	}

	r.builder.WriteString("}")
}

// renderGeneratorConfig renders a generator configuration.
func (r *Renderer) renderGeneratorConfig(generator *ast.GeneratorConfig) {
	r.builder.WriteString("generator ")
	r.renderIdentifier(&generator.Name)
	r.builder.WriteString(" {\n")

	for _, prop := range generator.Properties {
		r.renderConfigProperty(&prop, "  ")
	}

	r.builder.WriteString("}")
}

// renderField renders a field.
func (r *Renderer) renderField(field *ast.Field, indent string) {
	r.builder.WriteString(indent)
	r.renderIdentifier(&field.Name)
	r.builder.WriteString("  ")
	r.renderFieldType(&field.FieldType)

	// Render arity modifiers
	if field.Arity.IsList() {
		r.builder.WriteString("[]")
	} else if field.Arity.IsOptional() {
		r.builder.WriteString("?")
	}

	// Render attributes
	for _, attr := range field.Attributes {
		r.builder.WriteString(" ")
		r.renderAttribute(&attr, "")
	}

	r.builder.WriteString("\n")
}

// renderEnumValue renders an enum value.
func (r *Renderer) renderEnumValue(value *ast.EnumValue, indent string) {
	r.builder.WriteString(indent)
	r.renderIdentifier(&value.Name)

	// Render attributes
	for _, attr := range value.Attributes {
		r.builder.WriteString(" ")
		r.renderAttribute(&attr, "")
	}

	r.builder.WriteString("\n")
}

// renderAttribute renders an attribute.
func (r *Renderer) renderAttribute(attr *ast.Attribute, indent string) {
	r.builder.WriteString(indent)
	r.builder.WriteString("@")
	r.renderIdentifier(&attr.Name)

	hasArgs := len(attr.Arguments.Arguments) > 0 || len(attr.Arguments.EmptyArguments) > 0
	if hasArgs {
		r.builder.WriteString("(")

		// Render regular arguments
		for i, arg := range attr.Arguments.Arguments {
			if i > 0 {
				r.builder.WriteString(", ")
			}
			r.renderArgument(&arg)
		}

		// Render empty arguments (for autocompletion/invalid syntax)
		if len(attr.Arguments.EmptyArguments) > 0 {
			if len(attr.Arguments.Arguments) > 0 {
				r.builder.WriteString(", ")
			}
			for i, emptyArg := range attr.Arguments.EmptyArguments {
				if i > 0 {
					r.builder.WriteString(", ")
				}
				r.renderIdentifier(&emptyArg.Name)
				r.builder.WriteString(": ")
			}
		}

		// Render trailing comma if present
		if attr.Arguments.TrailingComma != nil {
			if hasArgs {
				r.builder.WriteString(", ")
			}
		}

		r.builder.WriteString(")")
	}
}

// renderArgument renders an attribute argument.
func (r *Renderer) renderArgument(arg *ast.Argument) {
	if arg.Name != nil {
		r.renderIdentifier(arg.Name)
		r.builder.WriteString(": ")
	}

	if arg.Value != nil {
		r.renderExpression(arg.Value)
	}
}

// renderConfigProperty renders a config property.
func (r *Renderer) renderConfigProperty(prop *ast.ConfigBlockProperty, indent string) {
	r.builder.WriteString(indent)
	r.renderIdentifier(&prop.Name)
	if prop.Value != nil {
		r.builder.WriteString(" = ")
		r.renderExpression(*prop.Value)
	}
	r.builder.WriteString("\n")
}

// renderFieldType renders a field type.
func (r *Renderer) renderFieldType(fieldType *ast.FieldType) {
	r.builder.WriteString(fieldType.Name())

	// Note: Arity is now stored separately on ast.Field, not on ast.FieldType
	// This method should be called with the ast.Field's Arity information
	// For now, we'll render based on the type name only
}

// renderIdentifier renders an identifier.
func (r *Renderer) renderIdentifier(ident *ast.Identifier) {
	r.builder.WriteString(ident.Name)
}

// renderExpression renders an expression.
func (r *Renderer) renderExpression(expr ast.Expression) {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		r.builder.WriteString(fmt.Sprintf(`"%s"`, e.Value))
	case *ast.IntLiteral:
		r.builder.WriteString(fmt.Sprintf("%d", e.Value))
	case *ast.FloatLiteral:
		r.builder.WriteString(fmt.Sprintf("%f", e.Value))
	case *ast.BooleanLiteral:
		r.builder.WriteString(fmt.Sprintf("%t", e.Value))
	case *ast.ArrayLiteral:
		r.builder.WriteString("[")
		for i, elem := range e.Elements {
			if i > 0 {
				r.builder.WriteString(", ")
			}
			r.renderExpression(elem)
		}
		r.builder.WriteString("]")
	case *ast.FunctionCall:
		r.renderIdentifier(&e.Name)
		r.builder.WriteString("(")
		for i, arg := range e.Arguments {
			if i > 0 {
				r.builder.WriteString(", ")
			}
			r.renderExpression(arg)
		}
		r.builder.WriteString(")")
	default:
		r.builder.WriteString("/* unknown expression */")
	}
}
