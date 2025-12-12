package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Identifier represents a named identifier in the schema.
type Identifier struct {
	Pos  lexer.Position
	Name string `(@Ident | @Keyword) ("." (@Ident | @Keyword))*`
}

// String returns the identifier name.
func (i *Identifier) String() string {
	if i == nil {
		return ""
	}
	return i.Name
}

// FieldName represents a field name that can be any identifier including keywords.
type FieldName struct {
	Pos  lexer.Position
	Name string `(@Ident | @Keyword)`
}

// String returns the field name.
func (f *FieldName) String() string {
	if f == nil {
		return ""
	}
	return f.Name
}

// GetName returns the field name (for interface compatibility).
func (f *FieldName) GetName() string {
	return f.String()
}
