// Package schemaast provides the AST data structure for Prisma schemas.
package ast

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/internal/debug"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// SchemaAst represents the AST of a Prisma schema.
type SchemaAst struct {
	// All models, enums, composite types, datasources, generators and type aliases.
	Tops []Top
}

// Sources returns all datasource blocks in the schema.
func (ast *SchemaAst) Sources() []*SourceConfig {
	debug.Debug("Extracting datasource blocks from AST", "total_tops", len(ast.Tops))
	var sources []*SourceConfig
	for _, top := range ast.Tops {
		if source := top.AsSource(); source != nil {
			sources = append(sources, source)
			debug.Debug("Found datasource", "name", source.Name.Name)
		}
	}
	debug.Debug("Completed extracting datasources", "count", len(sources))
	return sources
}

// Generators returns all generator blocks in the schema.
func (ast *SchemaAst) Generators() []*GeneratorConfig {
	debug.Debug("Extracting generator blocks from AST", "total_tops", len(ast.Tops))
	var generators []*GeneratorConfig
	for _, top := range ast.Tops {
		if generator := top.AsGenerator(); generator != nil {
			generators = append(generators, generator)
			debug.Debug("Found generator", "name", generator.Name.Name)
		}
	}
	debug.Debug("Completed extracting generators", "count", len(generators))
	return generators
}

// IterTops returns all top-level items in the schema.
func (ast *SchemaAst) IterTops() []Top {
	debug.Debug("Iterating over top-level items", "count", len(ast.Tops))
	// Return a copy to avoid external modification
	tops := make([]Top, len(ast.Tops))
	copy(tops, ast.Tops)
	for i, top := range tops {
		debug.Debug("Top-level item", "index", i, "type", top.GetType(), "name", top.TopName())
	}
	return tops
}

// Top represents a top-level item in a Prisma schema.
type Top interface {
	// Span returns the span of this top-level item.
	Span() diagnostics.Span
	// AsModel returns this as a Model if it is one, nil otherwise.
	AsModel() *Model
	// AsEnum returns this as an Enum if it is one, nil otherwise.
	AsEnum() *Enum
	// AsSource returns this as a SourceConfig if it is one, nil otherwise.
	AsSource() *SourceConfig
	// AsGenerator returns this as a GeneratorConfig if it is one, nil otherwise.
	AsGenerator() *GeneratorConfig
	// AsCompositeType returns this as a CompositeType if it is one, nil otherwise.
	AsCompositeType() *CompositeType
	// GetType returns a string saying what kind of item this is.
	GetType() string
	// Identifier returns the identifier of this top-level item.
	Identifier() *Identifier
	// TopName returns the name of this top-level item.
	TopName() string
}

// Model represents a model declaration in the schema.
type Model struct {
	Name          Identifier
	Fields        []Field
	Attributes    []Attribute
	Documentation *Comment
	IsView        bool
	ASTSpan       diagnostics.Span
}

// Enum represents an enum declaration in the schema.
type Enum struct {
	Name          Identifier
	Values        []EnumValue
	Attributes    []Attribute
	Documentation *Comment
	ASTSpan       diagnostics.Span
	InnerSpan     diagnostics.Span
}

// CompositeType represents a composite type declaration.
type CompositeType struct {
	Name          Identifier
	Fields        []Field
	Attributes    []Attribute
	Documentation *Comment
	ASTSpan       diagnostics.Span
	InnerSpan     diagnostics.Span
}

// SourceConfig represents a datasource configuration.
type SourceConfig struct {
	Name          Identifier
	Properties    []ConfigBlockProperty
	Documentation *Comment
	ASTSpan       diagnostics.Span
	InnerSpan     diagnostics.Span
}

// GeneratorConfig represents a generator configuration.
type GeneratorConfig struct {
	Name          Identifier
	Properties    []ConfigBlockProperty
	Documentation *Comment
	ASTSpan       diagnostics.Span
}

// Field represents a field in a model or composite type.
type Field struct {
	Name          Identifier
	FieldType     FieldType
	Arity         FieldArity
	Attributes    []Attribute
	Documentation *Comment
	ASTSpan       diagnostics.Span
}

// Span returns the span of this field.
func (f *Field) Span() diagnostics.Span {
	return f.ASTSpan
}

// FieldName returns the name of the field.
func (f *Field) FieldName() string {
	return f.Name.Name
}

// SpanForArgument finds the position span of the argument in the given field attribute.
func (f *Field) SpanForArgument(attribute string, argument string) *diagnostics.Span {
	for _, attr := range f.Attributes {
		if attr.Name.Name == attribute {
			for _, arg := range attr.Arguments.Arguments {
				if arg.Name != nil && arg.Name.Name == argument {
					return &arg.Span
				}
			}
		}
	}
	return nil
}

// SpanForAttribute finds the position span of the given attribute.
func (f *Field) SpanForAttribute(attribute string) *diagnostics.Span {
	for _, attr := range f.Attributes {
		if attr.Name.Name == attribute {
			return &attr.Span
		}
	}
	return nil
}

// EnumValue represents a value in an enum.
type EnumValue struct {
	Name          Identifier
	Attributes    []Attribute
	Documentation *Comment
	ASTSpan       diagnostics.Span
}

// Span returns the span of this enum value.
func (ev *EnumValue) Span() diagnostics.Span {
	return ev.ASTSpan
}

// Identifier represents an identifier in the schema.
type Identifier struct {
	Name    string
	ASTSpan diagnostics.Span
}

// Span implements the Expression interface.
func (i Identifier) Span() diagnostics.Span {
	return i.ASTSpan
}

// AsFunction implements the Expression interface.
func (i Identifier) AsFunction() *FunctionCall {
	return nil
}

// AsArray implements the Expression interface.
func (i Identifier) AsArray() *ArrayLiteral {
	return nil
}

// String implements the Expression interface.
func (i Identifier) String() string {
	return i.Name
}

// AsStringValue implements the Expression interface.
func (i Identifier) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}

// AsConstantValue implements the Expression interface.
func (i Identifier) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return &ConstantValue{Value: i.Name, ASTSpan: i.ASTSpan}, i.ASTSpan
}

// AsNumericValue implements the Expression interface.
func (i Identifier) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}

// IsEnvExpression implements the Expression interface.
func (i Identifier) IsEnvExpression() bool {
	return false
}

// DescribeValueType implements the Expression interface.
func (i Identifier) DescribeValueType() string {
	return "literal"
}

// IsFunction implements the Expression interface.
func (i Identifier) IsFunction() bool {
	return false
}

// IsArray implements the Expression interface.
func (i Identifier) IsArray() bool {
	return false
}

// IsString implements the Expression interface.
func (i Identifier) IsString() bool {
	return false
}

// Comment represents a documentation comment.
type Comment struct {
	Text string
	Span diagnostics.Span
}

// Attribute represents an attribute like @id, @unique, etc.
type Attribute struct {
	Name      Identifier
	Arguments ArgumentsList
	Span      diagnostics.Span
}

// SpanForArgument finds the position span of the given argument.
func (a *Attribute) SpanForArgument(argument string) *diagnostics.Span {
	for _, arg := range a.Arguments.Arguments {
		if arg.Name != nil && arg.Name.Name == argument {
			return &arg.Span
		}
	}
	return nil
}

// ArgumentsList represents a list of arguments inside parentheses.
type ArgumentsList struct {
	// The arguments themselves.
	Arguments []Argument
	// The arguments without a value (for autocompletion).
	EmptyArguments []EmptyArgument
	// The trailing comma at the end of the arguments list.
	TrailingComma *diagnostics.Span
}

// Iter returns an iterator over the arguments.
func (al *ArgumentsList) Iter() []Argument {
	return al.Arguments
}

// EmptyArgument represents an argument with a name but no value.
// This is invalid syntax, but we parse it for better diagnostics and autocompletion.
type EmptyArgument struct {
	Name Identifier
}

// Argument represents an argument to an attribute.
type Argument struct {
	Name  *Identifier
	Value Expression
	Span  diagnostics.Span
}

// IsUnnamed returns true if the argument has no name.
func (a *Argument) IsUnnamed() bool {
	return a.Name == nil
}

// ArgumentName returns the argument name if it has one.
func (a *Argument) ArgumentName() *string {
	if a.Name != nil {
		return &a.Name.Name
	}
	return nil
}

// Expression represents an expression in the schema.
type Expression interface {
	Span() diagnostics.Span
	// AsFunction returns this as a FunctionCall if it is one, nil otherwise.
	AsFunction() *FunctionCall
	// AsArray returns this as an ArrayLiteral if it is one, nil otherwise.
	AsArray() *ArrayLiteral
	// AsStringValue returns this as a StringLiteral if it is one, nil otherwise.
	AsStringValue() (*StringLiteral, diagnostics.Span)
	// AsConstantValue returns this as a ConstantValue if it is one, nil otherwise.
	AsConstantValue() (*ConstantValue, diagnostics.Span)
	// AsNumericValue returns this as a NumericValue if it is one, nil otherwise.
	AsNumericValue() (*NumericValue, diagnostics.Span)
	// IsEnvExpression returns true if this is an env() function call.
	IsEnvExpression() bool
	// DescribeValueType returns a friendly readable representation for a value's type.
	DescribeValueType() string
	// IsString returns true if this is a string literal.
	IsString() bool
}

// FieldTypeKind represents the kind of field type.
type FieldTypeKind interface {
	Name() string
	IsSupported() bool
}

// SupportedFieldType represents a supported field type.
type SupportedFieldType struct {
	Identifier Identifier
}

func (s SupportedFieldType) Name() string {
	return s.Identifier.Name
}

func (s SupportedFieldType) IsSupported() bool {
	return true
}

// UnsupportedFieldType represents an unsupported field type.
type UnsupportedFieldType struct {
	TypeName string
}

func (u UnsupportedFieldType) Name() string {
	return u.TypeName
}

func (u UnsupportedFieldType) IsSupported() bool {
	return false
}

// Name returns the type name of the field.
func (ft FieldType) Name() string {
	return ft.Type.Name()
}

// FieldTypeSpan returns the span of the field type.
func (ft FieldType) FieldTypeSpan() diagnostics.Span {
	return ft.ASTSpan
}

// AsUnsupported returns the unsupported type information if this is unsupported, nil otherwise.
func (ft FieldType) AsUnsupported() (*UnsupportedFieldType, *diagnostics.Span) {
	if unsupported, ok := ft.Type.(UnsupportedFieldType); ok {
		return &unsupported, &ft.ASTSpan
	}
	return nil, nil
}

// TypeName returns the type name of the field.
func (ft FieldType) TypeName() string {
	return ft.Type.Name()
}

// Span returns the span of the field type.
func (ft FieldType) Span() diagnostics.Span {
	return ft.ASTSpan
}

// IsOptional returns whether the field type is optional (needs to be checked with field arity).
func (ft FieldType) IsOptional() bool {
	// This method should be used in conjunction with FieldArity
	// The actual optional status is determined by the field's arity, not the type itself
	return false
}

// IsArray returns whether the field type is an array (needs to be checked with field arity).
func (ft FieldType) IsArray() bool {
	// This method should be used in conjunction with FieldArity
	// The actual array status is determined by the field's arity, not the type itself
	return false
}

// FieldArity represents the arity of a field (required, optional, list).
type FieldArity int

const (
	Required FieldArity = iota
	Optional
	List
)

// IsList returns true if the field arity is List.
func (fa FieldArity) IsList() bool {
	return fa == List
}

// IsOptional returns true if the field arity is Optional.
func (fa FieldArity) IsOptional() bool {
	return fa == Optional
}

// IsRequired returns true if the field arity is Required.
func (fa FieldArity) IsRequired() bool {
	return fa == Required
}

// Expression types

// NumericValue represents a numeric literal expression (int or float).
type NumericValue struct {
	Value   string
	ASTSpan diagnostics.Span
}

func (n NumericValue) Span() diagnostics.Span    { return n.ASTSpan }
func (n NumericValue) AsFunction() *FunctionCall { return nil }
func (n NumericValue) AsArray() *ArrayLiteral    { return nil }
func (n NumericValue) String() string            { return n.Value }
func (n NumericValue) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (n NumericValue) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (n NumericValue) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return &n, n.ASTSpan
}
func (n NumericValue) IsEnvExpression() bool     { return false }
func (n NumericValue) DescribeValueType() string { return "numeric" }
func (n NumericValue) IsFunction() bool          { return false }
func (n NumericValue) IsArray() bool             { return false }
func (n NumericValue) IsString() bool            { return false }

// ConstantValue represents a constant value expression (like boolean or enum).
type ConstantValue struct {
	Value   string
	ASTSpan diagnostics.Span
}

func (c ConstantValue) Span() diagnostics.Span    { return c.ASTSpan }
func (c ConstantValue) AsFunction() *FunctionCall { return nil }
func (c ConstantValue) AsArray() *ArrayLiteral    { return nil }
func (c ConstantValue) String() string            { return c.Value }
func (c ConstantValue) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (c ConstantValue) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return &c, c.ASTSpan
}
func (c ConstantValue) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (c ConstantValue) IsEnvExpression() bool     { return false }
func (c ConstantValue) DescribeValueType() string { return "literal" }
func (c ConstantValue) IsFunction() bool          { return false }
func (c ConstantValue) IsArray() bool             { return false }
func (c ConstantValue) IsString() bool            { return false }

// ConfigBlockProperty represents a property in a config block.
type ConfigBlockProperty struct {
	Name    Identifier
	Value   Expression // Optional - can be nil if expression is missing
	ASTSpan diagnostics.Span
}

func (c ConfigBlockProperty) Span() diagnostics.Span {
	return c.ASTSpan
}

// FieldType represents the type of a field.
type FieldType struct {
	// Type represents the field type - either supported or unsupported
	Type    FieldTypeKind
	ASTSpan diagnostics.Span
}

// StringLiteral represents a string literal expression.
type StringLiteral struct {
	Value   string
	ASTSpan diagnostics.Span
}

func (s StringLiteral) Span() diagnostics.Span    { return s.ASTSpan }
func (s StringLiteral) AsFunction() *FunctionCall { return nil }
func (s StringLiteral) AsArray() *ArrayLiteral    { return nil }
func (s StringLiteral) String() string            { return s.Value }
func (s StringLiteral) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return &s, s.ASTSpan
}
func (s StringLiteral) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (s StringLiteral) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (s StringLiteral) IsEnvExpression() bool     { return false }
func (s StringLiteral) DescribeValueType() string { return "string" }
func (s StringLiteral) IsFunction() bool          { return false }
func (s StringLiteral) IsArray() bool             { return false }
func (s StringLiteral) IsString() bool            { return true }

// IntLiteral represents an integer literal expression.
type IntLiteral struct {
	Value   int
	ASTSpan diagnostics.Span
}

func (i IntLiteral) Span() diagnostics.Span    { return i.ASTSpan }
func (i IntLiteral) AsFunction() *FunctionCall { return nil }
func (i IntLiteral) AsArray() *ArrayLiteral    { return nil }
func (i IntLiteral) String() string            { return fmt.Sprintf("%d", i.Value) }
func (i IntLiteral) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (i IntLiteral) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (i IntLiteral) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return &NumericValue{Value: fmt.Sprintf("%d", i.Value), ASTSpan: i.ASTSpan}, i.ASTSpan
}
func (i IntLiteral) IsEnvExpression() bool     { return false }
func (i IntLiteral) DescribeValueType() string { return "numeric" }
func (i IntLiteral) IsFunction() bool          { return false }
func (i IntLiteral) IsArray() bool             { return false }
func (i IntLiteral) IsString() bool            { return false }

// FloatLiteral represents a float literal expression.
type FloatLiteral struct {
	Value   float64
	ASTSpan diagnostics.Span
}

func (f FloatLiteral) Span() diagnostics.Span    { return f.ASTSpan }
func (f FloatLiteral) AsFunction() *FunctionCall { return nil }
func (f FloatLiteral) AsArray() *ArrayLiteral    { return nil }
func (f FloatLiteral) String() string            { return fmt.Sprintf("%f", f.Value) }
func (f FloatLiteral) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (f FloatLiteral) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (f FloatLiteral) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return &NumericValue{Value: fmt.Sprintf("%f", f.Value), ASTSpan: f.ASTSpan}, f.ASTSpan
}
func (f FloatLiteral) IsEnvExpression() bool     { return false }
func (f FloatLiteral) DescribeValueType() string { return "numeric" }
func (f FloatLiteral) IsFunction() bool          { return false }
func (f FloatLiteral) IsArray() bool             { return false }
func (f FloatLiteral) IsString() bool            { return false }

// BooleanLiteral represents a boolean literal expression.
type BooleanLiteral struct {
	Value   bool
	ASTSpan diagnostics.Span
}

func (b BooleanLiteral) Span() diagnostics.Span    { return b.ASTSpan }
func (b BooleanLiteral) AsFunction() *FunctionCall { return nil }
func (b BooleanLiteral) AsArray() *ArrayLiteral    { return nil }
func (b BooleanLiteral) String() string            { return fmt.Sprintf("%t", b.Value) }
func (b BooleanLiteral) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (b BooleanLiteral) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return &ConstantValue{Value: fmt.Sprintf("%t", b.Value), ASTSpan: b.ASTSpan}, b.ASTSpan
}
func (b BooleanLiteral) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (b BooleanLiteral) IsEnvExpression() bool     { return false }
func (b BooleanLiteral) DescribeValueType() string { return "literal" }
func (b BooleanLiteral) IsFunction() bool          { return false }
func (b BooleanLiteral) IsArray() bool             { return false }
func (b BooleanLiteral) IsString() bool            { return false }

// ArrayLiteral represents an array literal expression.
type ArrayLiteral struct {
	Elements []Expression
	ASTSpan  diagnostics.Span
}

// Expressions returns the elements of the array.
func (a ArrayLiteral) Expressions() []Expression {
	return a.Elements
}

func (a ArrayLiteral) Span() diagnostics.Span    { return a.ASTSpan }
func (a ArrayLiteral) AsFunction() *FunctionCall { return nil }
func (a ArrayLiteral) AsArray() *ArrayLiteral    { return &a }
func (a ArrayLiteral) String() string {
	// For now, just return a placeholder
	return "array"
}
func (a ArrayLiteral) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (a ArrayLiteral) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (a ArrayLiteral) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (a ArrayLiteral) IsEnvExpression() bool     { return false }
func (a ArrayLiteral) DescribeValueType() string { return "array" }
func (a ArrayLiteral) IsFunction() bool          { return false }
func (a ArrayLiteral) IsArray() bool             { return true }
func (a ArrayLiteral) IsString() bool            { return false }

// FunctionCall represents a function call expression.
type FunctionCall struct {
	Name      Identifier
	Arguments []Expression
	ASTSpan   diagnostics.Span
}

func (f FunctionCall) Span() diagnostics.Span    { return f.ASTSpan }
func (f FunctionCall) AsFunction() *FunctionCall { return &f }
func (f FunctionCall) AsArray() *ArrayLiteral    { return nil }
func (f FunctionCall) String() string {
	// For now, just return a placeholder
	return f.Name.Name
}
func (f FunctionCall) AsStringValue() (*StringLiteral, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (f FunctionCall) AsConstantValue() (*ConstantValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (f FunctionCall) AsNumericValue() (*NumericValue, diagnostics.Span) {
	return nil, diagnostics.Span{}
}
func (f FunctionCall) IsEnvExpression() bool {
	return f.Name.Name == "env"
}
func (f FunctionCall) DescribeValueType() string { return "functional" }
func (f FunctionCall) IsFunction() bool          { return true }
func (f FunctionCall) IsArray() bool             { return false }
func (f FunctionCall) IsString() bool            { return false }

// FunctionName returns the function name.
func (f FunctionCall) FunctionName() string {
	return f.Name.Name
}

// Implement Top interface methods

func (m *Model) Span() diagnostics.Span          { return m.ASTSpan }
func (m *Model) AsModel() *Model                 { return m }
func (m *Model) AsEnum() *Enum                   { return nil }
func (m *Model) AsSource() *SourceConfig         { return nil }
func (m *Model) AsGenerator() *GeneratorConfig   { return nil }
func (m *Model) AsCompositeType() *CompositeType { return nil }
func (m *Model) GetType() string {
	if m.IsView {
		return "view"
	}
	return "model"
}
func (m *Model) Identifier() *Identifier {
	return &m.Name
}
func (m *Model) TopName() string {
	return m.Name.Name
}

// ModelIsView returns whether this model is a view.
func (m *Model) ModelIsView() bool {
	return m.IsView
}

// IterFields returns an iterator over the fields with their indices.
func (m *Model) IterFields() []FieldWithId {
	debug.Debug("Iterating over model fields", "model", m.Name.Name, "field_count", len(m.Fields))
	result := make([]FieldWithId, len(m.Fields))
	for i := range m.Fields {
		result[i] = FieldWithId{Id: FieldId(i), Field: &m.Fields[i]}
		debug.Debug("Model field", "index", i, "name", m.Fields[i].Name.Name, "type", m.Fields[i].FieldType.Name())
	}
	return result
}

// FieldId represents an opaque identifier for a field in a model.
type FieldId int

// FieldWithId represents a field with its identifier.
type FieldWithId struct {
	Id    FieldId
	Field *Field
}

func (e *Enum) Span() diagnostics.Span          { return e.ASTSpan }
func (e *Enum) AsModel() *Model                 { return nil }
func (e *Enum) AsEnum() *Enum                   { return e }
func (e *Enum) AsSource() *SourceConfig         { return nil }
func (e *Enum) AsGenerator() *GeneratorConfig   { return nil }
func (e *Enum) AsCompositeType() *CompositeType { return nil }
func (e *Enum) GetType() string                 { return "enum" }
func (e *Enum) Identifier() *Identifier         { return &e.Name }
func (e *Enum) TopName() string                 { return e.Name.Name }

// IterValues returns an iterator over the enum values with their indices.
func (e *Enum) IterValues() []EnumValueWithId {
	debug.Debug("Iterating over enum values", "enum", e.Name.Name, "value_count", len(e.Values))
	result := make([]EnumValueWithId, len(e.Values))
	for i, value := range e.Values {
		result[i] = EnumValueWithId{Id: EnumValueId(i), Value: &value}
		debug.Debug("Enum value", "index", i, "name", value.Name.Name)
	}
	return result
}

// EnumValueId represents an opaque identifier for a value in an enum.
type EnumValueId int

// EnumValueWithId represents an enum value with its identifier.
type EnumValueWithId struct {
	Id    EnumValueId
	Value *EnumValue
}

func (s *SourceConfig) Span() diagnostics.Span          { return s.ASTSpan }
func (s *SourceConfig) AsModel() *Model                 { return nil }
func (s *SourceConfig) AsEnum() *Enum                   { return nil }
func (s *SourceConfig) AsSource() *SourceConfig         { return s }
func (s *SourceConfig) AsGenerator() *GeneratorConfig   { return nil }
func (s *SourceConfig) AsCompositeType() *CompositeType { return nil }
func (s *SourceConfig) GetType() string                 { return "source" }
func (s *SourceConfig) Identifier() *Identifier         { return &s.Name }
func (s *SourceConfig) TopName() string                 { return s.Name.Name }

func (g *GeneratorConfig) Span() diagnostics.Span          { return g.ASTSpan }
func (g *GeneratorConfig) AsModel() *Model                 { return nil }
func (g *GeneratorConfig) AsEnum() *Enum                   { return nil }
func (g *GeneratorConfig) AsSource() *SourceConfig         { return nil }
func (g *GeneratorConfig) AsGenerator() *GeneratorConfig   { return g }
func (g *GeneratorConfig) AsCompositeType() *CompositeType { return nil }
func (g *GeneratorConfig) GetType() string                 { return "generator" }
func (g *GeneratorConfig) Identifier() *Identifier         { return &g.Name }
func (g *GeneratorConfig) TopName() string                 { return g.Name.Name }

func (c *CompositeType) Span() diagnostics.Span          { return c.ASTSpan }
func (c *CompositeType) AsModel() *Model                 { return nil }
func (c *CompositeType) AsEnum() *Enum                   { return nil }
func (c *CompositeType) AsSource() *SourceConfig         { return nil }
func (c *CompositeType) AsGenerator() *GeneratorConfig   { return nil }
func (c *CompositeType) AsCompositeType() *CompositeType { return c }
func (c *CompositeType) GetType() string                 { return "composite type" }
func (c *CompositeType) Identifier() *Identifier         { return &c.Name }
func (c *CompositeType) TopName() string                 { return c.Name.Name }

// IsCommentedOut returns whether this composite type is commented out.
func (c *CompositeType) IsCommentedOut() bool {
	return false
}

// IterFields returns an iterator over the fields with their indices.
func (c *CompositeType) IterFields() []FieldWithId {
	debug.Debug("Iterating over composite type fields", "type", c.Name.Name, "field_count", len(c.Fields))
	result := make([]FieldWithId, len(c.Fields))
	for i, field := range c.Fields {
		result[i] = FieldWithId{Id: FieldId(i), Field: &field}
		debug.Debug("Composite type field", "index", i, "name", field.Name.Name, "type", field.FieldType.Name())
	}
	return result
}
