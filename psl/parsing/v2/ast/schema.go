package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Top is a union interface for all top-level schema declarations.
type Top interface {
	isTop()
	GetName() string
	GetDocumentation() string
	TopPos() lexer.Position
}

// Make all top-level types implement Top interface
func (m *Model) isTop()                 {}
func (m *Model) TopPos() lexer.Position { return m.Pos }

func (e *Enum) isTop()                 {}
func (e *Enum) TopPos() lexer.Position { return e.Pos }

func (c *CompositeType) isTop()                 {}
func (c *CompositeType) TopPos() lexer.Position { return c.Pos }

func (s *SourceConfig) isTop()                 {}
func (s *SourceConfig) TopPos() lexer.Position { return s.Pos }

func (g *GeneratorConfig) isTop()                 {}
func (g *GeneratorConfig) TopPos() lexer.Position { return g.Pos }

// SchemaAst represents the complete parsed Prisma schema.
type SchemaAst struct {
	Tops []Top
}

// IterTops returns an iterator over all top-level items with their IDs.
func (s *SchemaAst) IterTops() []TopWithID {
	result := make([]TopWithID, len(s.Tops))
	for i, top := range s.Tops {
		result[i] = TopWithID{ID: topIdxToTopID(i, top), Top: top}
	}
	return result
}

// Sources returns all datasource blocks.
func (s *SchemaAst) Sources() []*SourceConfig {
	var result []*SourceConfig
	for _, top := range s.Tops {
		if src, ok := top.(*SourceConfig); ok {
			result = append(result, src)
		}
	}
	return result
}

// Generators returns all generator blocks.
func (s *SchemaAst) Generators() []*GeneratorConfig {
	var result []*GeneratorConfig
	for _, top := range s.Tops {
		if gen, ok := top.(*GeneratorConfig); ok {
			result = append(result, gen)
		}
	}
	return result
}

// Models returns all model declarations.
func (s *SchemaAst) Models() []*Model {
	var result []*Model
	for _, top := range s.Tops {
		if model, ok := top.(*Model); ok {
			result = append(result, model)
		}
	}
	return result
}

// Enums returns all enum declarations.
func (s *SchemaAst) Enums() []*Enum {
	var result []*Enum
	for _, top := range s.Tops {
		if enum, ok := top.(*Enum); ok {
			result = append(result, enum)
		}
	}
	return result
}

// CompositeTypes returns all composite type declarations.
func (s *SchemaAst) CompositeTypes() []*CompositeType {
	var result []*CompositeType
	for _, top := range s.Tops {
		if ct, ok := top.(*CompositeType); ok {
			result = append(result, ct)
		}
	}
	return result
}

// GetModel returns a model by ID.
func (s *SchemaAst) GetModel(id ModelID) *Model {
	if int(id) < 0 || int(id) >= len(s.Tops) {
		return nil
	}
	if m, ok := s.Tops[id].(*Model); ok {
		return m
	}
	return nil
}

// GetEnum returns an enum by ID.
func (s *SchemaAst) GetEnum(id EnumID) *Enum {
	if int(id) < 0 || int(id) >= len(s.Tops) {
		return nil
	}
	if e, ok := s.Tops[id].(*Enum); ok {
		return e
	}
	return nil
}

// TopID is an identifier for a top-level item in the schema.
type TopID interface {
	isTopID()
}

// Make all ID types implement TopID
func (ModelID) isTopID()         {}
func (EnumID) isTopID()          {}
func (CompositeTypeID) isTopID() {}
func (SourceID) isTopID()        {}
func (GeneratorID) isTopID()     {}

// TopWithID pairs a top-level item with its ID.
type TopWithID struct {
	ID  TopID
	Top Top
}

// topIdxToTopID converts an index and top item to the appropriate TopID.
func topIdxToTopID(idx int, top Top) TopID {
	switch top.(type) {
	case *Model:
		return ModelID(idx)
	case *Enum:
		return EnumID(idx)
	case *CompositeType:
		return CompositeTypeID(idx)
	case *SourceConfig:
		return SourceID(idx)
	case *GeneratorConfig:
		return GeneratorID(idx)
	default:
		return ModelID(idx) // Fallback
	}
}
