// Package schemaast provides trait implementations for Prisma schema ASTs.
package ast

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// WithSpan represents an AST node with a span.
type WithSpan interface {
	Span() diagnostics.Span
}

// WithName represents an AST node with a name.
type WithName interface {
	GetName() string
}

// WithIdentifier represents an AST node with an identifier.
type WithIdentifier interface {
	GetIdentifier() Identifier
}

// WithAttributes represents an AST node with attributes.
type WithAttributes interface {
	GetAttributes() []Attribute
}

// WithDocumentation represents an AST node with documentation.
type WithDocumentation interface {
	GetDocumentation() *Comment
}

// Model implements WithName
func (m *Model) GetName() string {
	return m.Name.Name
}

// Model implements WithIdentifier
func (m *Model) GetIdentifier() Identifier {
	return m.Name
}

// Model implements WithAttributes
func (m *Model) GetAttributes() []Attribute {
	return m.Attributes
}

// Model implements WithDocumentation
func (m *Model) GetDocumentation() *Comment {
	return m.Documentation
}

// Enum implements WithName
func (e *Enum) GetName() string {
	return e.Name.Name
}

// Enum implements WithIdentifier
func (e *Enum) GetIdentifier() Identifier {
	return e.Name
}

// Enum implements WithAttributes
func (e *Enum) GetAttributes() []Attribute {
	return e.Attributes
}

// Enum implements WithDocumentation
func (e *Enum) GetDocumentation() *Comment {
	return e.Documentation
}

// CompositeType implements WithName
func (ct *CompositeType) GetName() string {
	return ct.Name.Name
}

// CompositeType implements WithIdentifier
func (ct *CompositeType) GetIdentifier() Identifier {
	return ct.Name
}

// CompositeType implements WithAttributes
func (ct *CompositeType) GetAttributes() []Attribute {
	return ct.Attributes
}

// CompositeType implements WithDocumentation
func (ct *CompositeType) GetDocumentation() *Comment {
	return ct.Documentation
}

// Field implements WithName
func (f *Field) GetName() string {
	return f.Name.Name
}

// Field implements WithIdentifier
func (f *Field) GetIdentifier() Identifier {
	return f.Name
}

// Field implements WithAttributes
func (f *Field) GetAttributes() []Attribute {
	return f.Attributes
}

// Field implements WithDocumentation
func (f *Field) GetDocumentation() *Comment {
	return f.Documentation
}

// EnumValue implements WithName
func (ev *EnumValue) GetName() string {
	return ev.Name.Name
}

// EnumValue implements WithIdentifier
func (ev *EnumValue) GetIdentifier() Identifier {
	return ev.Name
}

// EnumValue implements WithAttributes
func (ev *EnumValue) GetAttributes() []Attribute {
	return ev.Attributes
}

// SourceConfig implements WithName
func (sc *SourceConfig) GetName() string {
	return sc.Name.Name
}

// SourceConfig implements WithIdentifier
func (sc *SourceConfig) GetIdentifier() Identifier {
	return sc.Name
}

// GeneratorConfig implements WithName
func (gc *GeneratorConfig) GetName() string {
	return gc.Name.Name
}

// GeneratorConfig implements WithIdentifier
func (gc *GeneratorConfig) GetIdentifier() Identifier {
	return gc.Name
}

// Attribute implements WithName
func (a *Attribute) GetName() string {
	return a.Name.Name
}

// Attribute implements WithIdentifier
func (a *Attribute) GetIdentifier() Identifier {
	return a.Name
}

// Identifier implements WithName
func (i Identifier) GetName() string {
	return i.Name
}

// Identifier implements WithIdentifier
func (i Identifier) GetIdentifier() Identifier {
	return i
}
