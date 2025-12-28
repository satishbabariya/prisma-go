package ast

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// Attribute represents a field-level attribute (@attribute).
type Attribute struct {
	Pos       lexer.Position
	Name      *Identifier    `"@" @@`
	Arguments *ArgumentsList `("(" @@ ")")?`
}

// String returns the string representation of the attribute.
func (a *Attribute) String() string {
	args := ""
	if a.Arguments != nil && len(a.Arguments.Arguments) > 0 {
		args = "(" + a.Arguments.String() + ")"
	}
	return "@" + a.Name.Name + args
}

// GetName returns the attribute name.
func (a *Attribute) GetName() string {
	if a.Name == nil {
		return ""
	}
	return a.Name.Name
}

// SpanForArgument finds the position of a specific named argument.
func (a *Attribute) SpanForArgument(name string) *lexer.Position {
	if a.Arguments == nil {
		return nil
	}
	for _, arg := range a.Arguments.Arguments {
		if arg.Name != nil && arg.Name.Name == name {
			return &arg.Pos
		}
	}
	return nil
}

// BlockAttribute represents a block-level attribute (@@attribute).
type BlockAttribute struct {
	Pos       lexer.Position
	Name      *Identifier    `"@@" @@`
	Arguments *ArgumentsList `("(" @@ ")")?`
}

// String returns the string representation of the block attribute.
func (b *BlockAttribute) String() string {
	args := ""
	if b.Arguments != nil && len(b.Arguments.Arguments) > 0 {
		args = "(" + b.Arguments.String() + ")"
	}
	return "@@" + b.Name.Name + args
}

// GetName returns the block attribute name.
func (b *BlockAttribute) GetName() string {
	if b.Name == nil {
		return ""
	}
	return b.Name.Name
}

// AttributeList is a helper for parsing multiple attributes.
type AttributeList struct {
	Attributes []*Attribute `@@*`
}

// String returns the string representation of all attributes.
func (a *AttributeList) String() string {
	if a == nil || len(a.Attributes) == 0 {
		return ""
	}
	parts := make([]string, len(a.Attributes))
	for i, attr := range a.Attributes {
		parts[i] = attr.String()
	}
	return strings.Join(parts, " ")
}

// BlockAttributeList is a helper for parsing multiple block attributes.
type BlockAttributeList struct {
	Attributes []*BlockAttribute `@@*`
}

// String returns the string representation of all block attributes.
func (b *BlockAttributeList) String() string {
	if b == nil || len(b.Attributes) == 0 {
		return ""
	}
	parts := make([]string, len(b.Attributes))
	for i, attr := range b.Attributes {
		parts[i] = attr.String()
	}
	return strings.Join(parts, "\n")
}

// AttributeContainer identifies what kind of element contains the attribute.
type AttributeContainer int

const (
	AttributeContainerModel AttributeContainer = iota
	AttributeContainerModelField
	AttributeContainerEnum
	AttributeContainerEnumValue
	AttributeContainerCompositeTypeField
)

// AttributeID uniquely identifies an attribute in the schema.
type AttributeID struct {
	Container AttributeContainer
	ParentIdx int
	FieldIdx  int // Only for field-level containers
	AttrIdx   int
}

// String returns a debug string for the AttributeID.
func (a AttributeID) String() string {
	return fmt.Sprintf("AttributeID{Container: %d, ParentIdx: %d, FieldIdx: %d, AttrIdx: %d}",
		a.Container, a.ParentIdx, a.FieldIdx, a.AttrIdx)
}
