package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// CompositeType represents a type (composite type) declaration.
type CompositeType struct {
	Pos           lexer.Position
	Documentation *CommentBlock `@@?`
	Keyword       string        `@"type"`
	Name          *Identifier   `@@`
	Fields        []*Field      `"{" @@* "}"`
}

// GetName returns the composite type name.
func (c *CompositeType) GetName() string {
	if c.Name == nil {
		return ""
	}
	return c.Name.Name
}

// IterFields returns an iterator over fields with their IDs.
func (c *CompositeType) IterFields() []FieldWithID {
	result := make([]FieldWithID, len(c.Fields))
	for i, f := range c.Fields {
		result[i] = FieldWithID{ID: FieldID(i), Field: f}
	}
	return result
}

// GetField returns a field by ID.
func (c *CompositeType) GetField(id FieldID) *Field {
	if int(id) < 0 || int(id) >= len(c.Fields) {
		return nil
	}
	return c.Fields[id]
}

// GetDocumentation returns the composite type documentation.
func (c *CompositeType) GetDocumentation() string {
	if c.Documentation == nil {
		return ""
	}
	return c.Documentation.GetText()
}

// CompositeTypeID is an opaque identifier for a composite type in the schema.
type CompositeTypeID int
