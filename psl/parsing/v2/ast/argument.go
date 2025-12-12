package ast

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// ArgumentsList represents a list of arguments in parentheses.
type ArgumentsList struct {
	Pos           lexer.Position
	Arguments     []*Argument `(@@ ("," @@)*)?`
	TrailingComma bool        `@","?`
}

// String returns the string representation of the arguments list.
func (a *ArgumentsList) String() string {
	if a == nil || len(a.Arguments) == 0 {
		return ""
	}
	parts := make([]string, len(a.Arguments))
	for i, arg := range a.Arguments {
		parts[i] = arg.String()
	}
	result := strings.Join(parts, ", ")
	if a.TrailingComma {
		result += ","
	}
	return result
}

// Iter returns an iterator over the arguments.
func (a *ArgumentsList) Iter() []*Argument {
	if a == nil {
		return nil
	}
	return a.Arguments
}

// Argument represents a single argument (named or positional).
type Argument struct {
	Pos   lexer.Position
	Name  *Identifier `(@@ ":")?`
	Value Expression  `@@`
}

// String returns the string representation of the argument.
func (a *Argument) String() string {
	if a.Name != nil {
		return fmt.Sprintf("%s: %s", a.Name.Name, a.Value.String())
	}
	return a.Value.String()
}

// IsNamed returns true if this is a named argument.
func (a *Argument) IsNamed() bool {
	return a.Name != nil
}

// GetName returns the argument name or empty string if positional.
func (a *Argument) GetName() string {
	if a.Name == nil {
		return ""
	}
	return a.Name.Name
}

// EmptyArgument represents an argument with a name but no value (for autocomplete).
type EmptyArgument struct {
	Pos  lexer.Position
	Name *Identifier `@@ ":"`
}
