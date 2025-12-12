package ast

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
)

// FieldArity represents the arity/cardinality of a field.
type FieldArity int

const (
	// FieldArityRequired means the field must have a value.
	FieldArityRequired FieldArity = iota
	// FieldArityOptional means the field can be null (Type?).
	FieldArityOptional
	// FieldArityList means the field is an array (Type[]).
	FieldArityList
)

// String returns the string representation of the arity.
func (a FieldArity) String() string {
	switch a {
	case FieldArityOptional:
		return "?"
	case FieldArityList:
		return "[]"
	default:
		return ""
	}
}

// IsRequired returns true if the field is required.
func (a FieldArity) IsRequired() bool { return a == FieldArityRequired }

// IsOptional returns true if the field is optional.
func (a FieldArity) IsOptional() bool { return a == FieldArityOptional }

// IsList returns true if the field is a list.
func (a FieldArity) IsList() bool { return a == FieldArityList }

// FieldType represents the type of a field.
type FieldType struct {
	Pos         lexer.Position
	Name        string  `@Ident`
	Unsupported *string // For Unsupported("type") - parsed separately
}

// String returns the string representation of the field type.
func (f *FieldType) String() string {
	if f.Unsupported != nil {
		return fmt.Sprintf("Unsupported(%q)", *f.Unsupported)
	}
	return f.Name
}

// IsUnsupported returns true if this is an Unsupported type.
func (f *FieldType) IsUnsupported() bool {
	return f.Unsupported != nil
}

// Field represents a field in a model or composite type.
type Field struct {
	Pos           lexer.Position
	Documentation *CommentBlock `@@?`
	Name          *Identifier   `@@`
	Colon         *string       `@":"?` // Legacy colon syntax
	Type          *FieldType    `@@?`
	Arity         FieldArity    // Determined during parsing
	ListSuffix    *string       `@("[" "]")?`
	OptionalMark  *string       `@"?"?`
	Attributes    []*Attribute  `@@*`
}

// GetName returns the field name.
func (f *Field) GetName() string {
	if f.Name == nil {
		return ""
	}
	return f.Name.Name
}

// GetTypeName returns the type name.
func (f *Field) GetTypeName() string {
	if f.Type == nil {
		return ""
	}
	return f.Type.Name
}

// String returns a string representation of the field.
func (f *Field) String() string {
	typeName := ""
	if f.Type != nil {
		typeName = f.Type.String()
	}
	return fmt.Sprintf("%s %s%s", f.GetName(), typeName, f.Arity.String())
}

// SpanForAttribute finds the span of a specific attribute.
func (f *Field) SpanForAttribute(name string) *lexer.Position {
	for _, attr := range f.Attributes {
		if attr.GetName() == name {
			return &attr.Pos
		}
	}
	return nil
}

// SpanForArgument finds the span of an argument within an attribute.
func (f *Field) SpanForArgument(attrName, argName string) *lexer.Position {
	for _, attr := range f.Attributes {
		if attr.GetName() == attrName {
			return attr.SpanForArgument(argName)
		}
	}
	return nil
}

// FieldID is an opaque identifier for a field within a model.
type FieldID int

// FieldIDMin is the minimum field ID.
const FieldIDMin FieldID = 0

// FieldIDMax is the maximum field ID.
const FieldIDMax FieldID = 1<<31 - 1
