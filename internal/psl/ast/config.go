package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// ConfigBlockProperty represents a key-value property in a config block.
type ConfigBlockProperty struct {
	Pos   lexer.Position
	Name  *Identifier `@@`
	Value Expression  `"=" @@`
}

// GetName returns the property name.
func (p *ConfigBlockProperty) GetName() string {
	if p.Name == nil {
		return ""
	}
	return p.Name.Name
}

// SourceConfig represents a datasource block.
type SourceConfig struct {
	Pos           lexer.Position
	Documentation *CommentBlock          `@@?`
	Keyword       string                 `@"datasource"`
	Name          *Identifier            `@@`
	Properties    []*ConfigBlockProperty `"{" @@* "}"`
}

// GetName returns the datasource name.
func (s *SourceConfig) GetName() string {
	if s.Name == nil {
		return ""
	}
	return s.Name.Name
}

// GetProperty finds a property by name.
func (s *SourceConfig) GetProperty(name string) *ConfigBlockProperty {
	for _, prop := range s.Properties {
		if prop.GetName() == name {
			return prop
		}
	}
	return nil
}

// GetDocumentation returns the datasource documentation.
func (s *SourceConfig) GetDocumentation() string {
	if s.Documentation == nil {
		return ""
	}
	return s.Documentation.GetText()
}

// GeneratorConfig represents a generator block.
type GeneratorConfig struct {
	Pos           lexer.Position
	Documentation *CommentBlock          `@@?`
	Keyword       string                 `@"generator"`
	Name          *Identifier            `@@`
	Properties    []*ConfigBlockProperty `"{" @@* "}"`
}

// GetName returns the generator name.
func (g *GeneratorConfig) GetName() string {
	if g.Name == nil {
		return ""
	}
	return g.Name.Name
}

// GetProperty finds a property by name.
func (g *GeneratorConfig) GetProperty(name string) *ConfigBlockProperty {
	for _, prop := range g.Properties {
		if prop.GetName() == name {
			return prop
		}
	}
	return nil
}

// GetDocumentation returns the generator documentation.
func (g *GeneratorConfig) GetDocumentation() string {
	if g.Documentation == nil {
		return ""
	}
	return g.Documentation.GetText()
}

// SourceID is an opaque identifier for a datasource in the schema.
type SourceID int

// GeneratorID is an opaque identifier for a generator in the schema.
type GeneratorID int
