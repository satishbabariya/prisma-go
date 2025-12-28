package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Enum represents an enum declaration.
type Enum struct {
	Pos             lexer.Position
	Documentation   *CommentBlock     `@@?`
	Keyword         string            `@"enum"`
	Name            *Identifier       `@@`
	Values          []*EnumValue      `"{" @@*`
	BlockAttributes []*BlockAttribute `@@* "}"`
}

// GetName returns the enum name.
func (e *Enum) GetName() string {
	if e.Name == nil {
		return ""
	}
	return e.Name.Name
}

// IterValues returns an iterator over enum values with their IDs.
func (e *Enum) IterValues() []EnumValueWithID {
	result := make([]EnumValueWithID, len(e.Values))
	for i, v := range e.Values {
		result[i] = EnumValueWithID{ID: EnumValueID(i), Value: v}
	}
	return result
}

// GetDocumentation returns the enum documentation.
func (e *Enum) GetDocumentation() string {
	if e.Documentation == nil {
		return ""
	}
	return e.Documentation.GetText()
}

// EnumValue represents a single enum value.
type EnumValue struct {
	Pos           lexer.Position
	Documentation *CommentBlock `@@?`
	Name          *Identifier   `@@`
	Attributes    []*Attribute  `@@*`
}

// GetName returns the enum value name.
func (v *EnumValue) GetName() string {
	if v.Name == nil {
		return ""
	}
	return v.Name.Name
}

// GetDocumentation returns the enum value documentation.
func (v *EnumValue) GetDocumentation() string {
	if v.Documentation == nil {
		return ""
	}
	return v.Documentation.GetText()
}

// EnumValueWithID pairs an enum value with its ID.
type EnumValueWithID struct {
	ID    EnumValueID
	Value *EnumValue
}

// EnumValueID is an opaque identifier for an enum value.
type EnumValueID int

// EnumValueIDMin is the minimum enum value ID.
const EnumValueIDMin EnumValueID = 0

// EnumValueIDMax is the maximum enum value ID.
const EnumValueIDMax EnumValueID = 1<<31 - 1

// EnumID is an opaque identifier for an enum in the schema.
type EnumID int
