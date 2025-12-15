package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// ExtendedType represents an `extend type` declaration that adds fields to an existing type.
type ExtendedType struct {
	Pos           lexer.Position
	Documentation *CommentBlock `@@?`
	ExtendKeyword string        `@"extend"`
	TypeKeyword   string        `@"type"`
	Name          *Identifier   `@@`
	Fields        []*Field      `"{" @@* "}"`
}

// GetName returns the name of the extended type.
func (e *ExtendedType) GetName() string {
	if e.Name == nil {
		return ""
	}
	return e.Name.Name
}

// GetDocumentation returns the documentation for the extended type.
func (e *ExtendedType) GetDocumentation() string {
	if e.Documentation == nil {
		return ""
	}
	return e.Documentation.GetText()
}

// TopPos returns the position for this top-level item.
func (e *ExtendedType) TopPos() lexer.Position {
	return e.Pos
}

// isTop implements the Top interface marker method.
func (e *ExtendedType) isTop() {}
