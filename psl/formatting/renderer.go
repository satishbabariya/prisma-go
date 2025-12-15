// Package formatting provides rendering functionality for Prisma schema ASTs.
package formatting

import (
	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"

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
func (r *Renderer) Render(schema *ast.SchemaAst) string {
	r.builder.Reset()

	for i, top := range schema.Tops {
		if i > 0 {
			r.builder.WriteString("\n\n")
		}
		r.renderTop(top)
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
	r.builder.WriteString(model.Keyword + " ")
	r.renderIdentifier(model.Name)
	r.builder.WriteString(" {\n")

	// Render fields
	for _, field := range model.Fields {
		r.renderField(field, "  ")
	}

	// Render attributes
	for _, attr := range model.BlockAttributes {
		r.renderBlockAttribute(attr, "  ")
	}

	r.builder.WriteString("}")
}

// renderEnum renders an enum.
func (r *Renderer) renderEnum(enum *ast.Enum) {
	r.builder.WriteString("enum ")
	r.renderIdentifier(enum.Name)
	r.builder.WriteString(" {\n")

	// Render values
	for _, value := range enum.Values {
		r.renderEnumValue(value, "  ")
	}

	// Render attributes (block attributes for enums usually @@map but typically enum attributes are mostly on values or block level)
	for _, attr := range enum.BlockAttributes {
		r.renderBlockAttribute(attr, "  ")
	}

	r.builder.WriteString("}")
}

// renderCompositeType renders a composite type.
func (r *Renderer) renderCompositeType(compositeType *ast.CompositeType) {
	r.builder.WriteString("type ")
	r.renderIdentifier(compositeType.Name)
	r.builder.WriteString(" {\n")

	// Render fields
	for _, field := range compositeType.Fields {
		r.renderField(field, "  ")
	}

	r.builder.WriteString("}")
}

// renderSourceConfig renders a datasource configuration.
func (r *Renderer) renderSourceConfig(source *ast.SourceConfig) {
	r.builder.WriteString("datasource ")
	r.renderIdentifier(source.Name)
	r.builder.WriteString(" {\n")

	for _, prop := range source.Properties {
		r.renderConfigBlockProperty(prop, "  ")
	}

	r.builder.WriteString("}")
}

// renderGeneratorConfig renders a generator configuration.
func (r *Renderer) renderGeneratorConfig(generator *ast.GeneratorConfig) {
	r.builder.WriteString("generator ")
	r.renderIdentifier(generator.Name)
	r.builder.WriteString(" {\n")

	for _, prop := range generator.Properties {
		r.renderConfigBlockProperty(prop, "  ")
	}

	r.builder.WriteString("}")
}

// renderField renders a field.
func (r *Renderer) renderField(field *ast.Field, indent string) {
	r.builder.WriteString(indent)
	if field.Name != nil {
		r.builder.WriteString(field.Name.Name)
	}
	r.builder.WriteString(" ")
	r.renderFieldType(field.Type)

	// Render arity modifiers
	if field.Arity.IsList() {
		r.builder.WriteString("[]")
	} else if field.Arity.IsOptional() {
		r.builder.WriteString("?")
	}

	// Render attributes
	for _, attr := range field.Attributes {
		r.builder.WriteString(" ")
		r.renderAttribute(attr)
	}

	r.builder.WriteString("\n")
}

// renderEnumValue renders an enum value.
func (r *Renderer) renderEnumValue(value *ast.EnumValue, indent string) {
	r.builder.WriteString(indent)
	r.renderIdentifier(value.Name)

	// Render attributes
	for _, attr := range value.Attributes {
		r.builder.WriteString(" ")
		r.renderAttribute(attr)
	}

	r.builder.WriteString("\n")
}

// renderAttribute renders a field attribute.
func (r *Renderer) renderAttribute(attr *ast.Attribute) {
	r.builder.WriteString("@")
	r.renderIdentifier(attr.Name)
	r.renderArguments(attr.Arguments)
}

// renderBlockAttribute renders a block attribute (@@).
func (r *Renderer) renderBlockAttribute(attr *ast.BlockAttribute, indent string) {
	r.builder.WriteString(indent)
	r.builder.WriteString("@@")
	r.renderIdentifier(attr.Name)
	r.renderArguments(attr.Arguments)
	r.builder.WriteString("\n")
}

// renderArguments renders argument list.
func (r *Renderer) renderArguments(args *ast.ArgumentsList) {
	if args == nil {
		return
	}

	// We only have Arguments in V2 AST ArgumentsList struct
	hasArgs := len(args.Arguments) > 0
	if hasArgs {
		r.builder.WriteString("(")

		for i, arg := range args.Arguments {
			if i > 0 {
				r.builder.WriteString(", ")
			}
			r.renderArgument(arg)
		}

		if args.TrailingComma {
			r.builder.WriteString(",")
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

// renderConfigBlockProperty renders a config property.
func (r *Renderer) renderConfigBlockProperty(prop *ast.ConfigBlockProperty, indent string) {
	r.builder.WriteString(indent)
	r.renderIdentifier(prop.Name)
	if prop.Value != nil {
		r.builder.WriteString(" = ")
		r.renderExpression(prop.Value)
	}
	r.builder.WriteString("\n")
}

// renderFieldType renders a field type.
func (r *Renderer) renderFieldType(fieldType *ast.FieldType) {
	if fieldType == nil {
		return
	}
	r.builder.WriteString(fieldType.Name)
}

// renderIdentifier renders an identifier.
func (r *Renderer) renderIdentifier(ident *ast.Identifier) {
	if ident != nil {
		r.builder.WriteString(ident.Name)
	}
}

// renderExpression renders an expression.
func (r *Renderer) renderExpression(expr ast.Expression) {
	switch e := expr.(type) {
	case *ast.StringValue:
		// e.Value includes quotes
		r.builder.WriteString(e.Value)
	case *ast.NumericValue:
		r.builder.WriteString(e.Value)
	case *ast.ConstantValue:
		// Includes booleans (true/false) and identifiers
		r.builder.WriteString(e.Value)
	case *ast.FunctionCall:
		r.builder.WriteString(e.Name)
		r.renderArguments(e.Arguments)
	case *ast.ArrayExpression:
		r.builder.WriteString("[")
		for i, elem := range e.Elements {
			if i > 0 {
				r.builder.WriteString(", ")
			}
			r.renderExpression(elem)
		}
		r.builder.WriteString("]")
	case *ast.PathValue:
		r.builder.WriteString(strings.Join(e.Parts, "."))
	default:
		// Fallback for types that might not match directly or if interface handling differs
		// But in V2 AST, Expression IS the interface implemented by these pointers.
		// So this should cover it.
		r.builder.WriteString("/* unknown expression */")
	}
}
