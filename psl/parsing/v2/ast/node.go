// Package ast provides the abstract syntax tree types for Prisma schemas.
package ast

// Node represents a node in the AST.
type Node interface {
	// NodeSpan returns the source location of this node.
	NodeSpan() Span

	// Accept accepts a visitor.
	Accept(visitor Visitor)
}

// Visitor defines the visitor interface for AST traversal.
type Visitor interface {
	VisitSchemaAst(node *SchemaAst)
	VisitSourceConfig(node *SourceConfig)
	VisitGeneratorConfig(node *GeneratorConfig)
	VisitModel(node *Model)
	VisitField(node *Field)
	VisitEnum(node *Enum)
	VisitEnumValue(node *EnumValue)
	VisitAttribute(node *Attribute)
	VisitCompositeType(node *CompositeType)
}

// BaseVisitor provides a default implementation of Visitor.
type BaseVisitor struct{}

func (v *BaseVisitor) VisitSchemaAst(node *SchemaAst)             {}
func (v *BaseVisitor) VisitSourceConfig(node *SourceConfig)       {}
func (v *BaseVisitor) VisitGeneratorConfig(node *GeneratorConfig) {}
func (v *BaseVisitor) VisitModel(node *Model)                     {}
func (v *BaseVisitor) VisitField(node *Field)                     {}
func (v *BaseVisitor) VisitEnum(node *Enum)                       {}
func (v *BaseVisitor) VisitEnumValue(node *EnumValue)             {}
func (v *BaseVisitor) VisitAttribute(node *Attribute)             {}
func (v *BaseVisitor) VisitCompositeType(node *CompositeType)     {}
