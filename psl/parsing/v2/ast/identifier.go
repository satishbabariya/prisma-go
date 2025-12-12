package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Identifier represents a named identifier in the schema.
type Identifier struct {
	Pos  lexer.Position
	Name string `@Ident`
}

// String returns the identifier name.
func (i *Identifier) String() string {
	if i == nil {
		return ""
	}
	return i.Name
}
