package ast

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
)

// Expression represents a value expression in the schema.
// This is a union type that can be one of several expression types.
type Expression interface {
	isExpression()
	Span() lexer.Position
	String() string

	AsStringValue() (*StringValue, bool)
	AsNumericValue() (*NumericValue, bool)
	AsConstantValue() (*ConstantValue, bool)
	AsFunction() (*FunctionCall, bool)
	AsArray() (*ArrayExpression, bool)
	AsBooleanValue() (bool, bool)
}

// StringValue represents a quoted string literal.
type StringValue struct {
	Pos   lexer.Position
	Value string `@String`
}

func (s *StringValue) isExpression() {}

// Span returns the source position.
func (s *StringValue) Span() lexer.Position { return s.Pos }

// String returns the string representation.
func (s *StringValue) String() string {
	return fmt.Sprintf("%q", s.Value)
}

// GetValue returns the unquoted string value.
func (s *StringValue) GetValue() string {
	// Remove surrounding quotes and unescape
	val := s.Value
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
		val = val[1 : len(val)-1]
	}
	// Handle escape sequences
	val = strings.ReplaceAll(val, `\\`, `\`)
	val = strings.ReplaceAll(val, `\"`, `"`)
	val = strings.ReplaceAll(val, `\n`, "\n")
	val = strings.ReplaceAll(val, `\r`, "\r")
	val = strings.ReplaceAll(val, `\t`, "\t")
	return val
}

// NumericValue represents a numeric literal (int or float).
type NumericValue struct {
	Pos   lexer.Position
	Value string `@Number`
}

func (n *NumericValue) isExpression() {}

// Span returns the source position.
func (n *NumericValue) Span() lexer.Position { return n.Pos }

// String returns the string representation.
func (n *NumericValue) String() string { return n.Value }

// ConstantValue represents a constant/identifier value (true, false, enum values, field references).
type ConstantValue struct {
	Pos   lexer.Position
	Value string `@Ident`
}

func (c *ConstantValue) isExpression() {}

// Span returns the source position.
func (c *ConstantValue) Span() lexer.Position { return c.Pos }

// String returns the string representation.
func (c *ConstantValue) String() string { return c.Value }

// PathValue represents a dotted path like "db.User" or just "User".
type PathValue struct {
	Pos   lexer.Position
	Parts []string `@Ident ("." @Ident)*`
}

func (p *PathValue) isExpression() {}

// Span returns the source position.
func (p *PathValue) Span() lexer.Position { return p.Pos }

// String returns the string representation.
func (p *PathValue) String() string {
	return strings.Join(p.Parts, ".")
}

// FunctionCall represents a function call expression like env("DATABASE_URL").
type FunctionCall struct {
	Pos       lexer.Position
	Name      string         `@Ident`
	Arguments *ArgumentsList `"(" @@? ")"`
}

func (f *FunctionCall) isExpression() {}

// Span returns the source position.
func (f *FunctionCall) Span() lexer.Position { return f.Pos }

// String returns the string representation.
func (f *FunctionCall) String() string {
	args := ""
	if f.Arguments != nil {
		args = f.Arguments.String()
	}
	return fmt.Sprintf("%s(%s)", f.Name, args)
}

// ArrayExpression represents an array literal like [1, 2, 3] or [field1, field2].
type ArrayExpression struct {
	Pos      lexer.Position
	Elements []Expression `"[" (@@ ("," @@)*)? "]"`
}

func (a *ArrayExpression) isExpression() {}

// Span returns the source position.
func (a *ArrayExpression) Span() lexer.Position { return a.Pos }

// String returns the string representation.
func (a *ArrayExpression) String() string {
	parts := make([]string, len(a.Elements))
	for i, elem := range a.Elements {
		parts[i] = elem.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// AsStringValue returns the StringValue if the expression is one.
func (s *StringValue) AsStringValue() (*StringValue, bool)     { return s, true }
func (n *NumericValue) AsStringValue() (*StringValue, bool)    { return nil, false }
func (c *ConstantValue) AsStringValue() (*StringValue, bool)   { return nil, false }
func (p *PathValue) AsStringValue() (*StringValue, bool)       { return nil, false }
func (f *FunctionCall) AsStringValue() (*StringValue, bool)    { return nil, false }
func (a *ArrayExpression) AsStringValue() (*StringValue, bool) { return nil, false }

// AsNumericValue returns the NumericValue if the expression is one.
func (s *StringValue) AsNumericValue() (*NumericValue, bool)     { return nil, false }
func (n *NumericValue) AsNumericValue() (*NumericValue, bool)    { return n, true }
func (c *ConstantValue) AsNumericValue() (*NumericValue, bool)   { return nil, false }
func (p *PathValue) AsNumericValue() (*NumericValue, bool)       { return nil, false }
func (f *FunctionCall) AsNumericValue() (*NumericValue, bool)    { return nil, false }
func (a *ArrayExpression) AsNumericValue() (*NumericValue, bool) { return nil, false }

// AsConstantValue returns the ConstantValue if the expression is one.
func (s *StringValue) AsConstantValue() (*ConstantValue, bool)     { return nil, false }
func (n *NumericValue) AsConstantValue() (*ConstantValue, bool)    { return nil, false }
func (c *ConstantValue) AsConstantValue() (*ConstantValue, bool)   { return c, true }
func (p *PathValue) AsConstantValue() (*ConstantValue, bool)       { return nil, false }
func (f *FunctionCall) AsConstantValue() (*ConstantValue, bool)    { return nil, false }
func (a *ArrayExpression) AsConstantValue() (*ConstantValue, bool) { return nil, false }

// AsFunction returns the FunctionCall if the expression is one.
func (s *StringValue) AsFunction() (*FunctionCall, bool)     { return nil, false }
func (n *NumericValue) AsFunction() (*FunctionCall, bool)    { return nil, false }
func (c *ConstantValue) AsFunction() (*FunctionCall, bool)   { return nil, false }
func (p *PathValue) AsFunction() (*FunctionCall, bool)       { return nil, false }
func (f *FunctionCall) AsFunction() (*FunctionCall, bool)    { return f, true }
func (a *ArrayExpression) AsFunction() (*FunctionCall, bool) { return nil, false }

// AsArray returns the ArrayExpression if the expression is one.
func (s *StringValue) AsArray() (*ArrayExpression, bool)     { return nil, false }
func (n *NumericValue) AsArray() (*ArrayExpression, bool)    { return nil, false }
func (c *ConstantValue) AsArray() (*ArrayExpression, bool)   { return nil, false }
func (p *PathValue) AsArray() (*ArrayExpression, bool)       { return nil, false }
func (f *FunctionCall) AsArray() (*ArrayExpression, bool)    { return nil, false }
func (a *ArrayExpression) AsArray() (*ArrayExpression, bool) { return a, true }

// AsBooleanValue returns the boolean value if the expression is a constant "true" or "false".
func (c *ConstantValue) AsBooleanValue() (bool, bool) {
	if c.Value == "true" {
		return true, true
	}
	if c.Value == "false" {
		return false, true
	}
	return false, false
}

// Default implementations for AsBooleanValue
func (s *StringValue) AsBooleanValue() (bool, bool)     { return false, false }
func (n *NumericValue) AsBooleanValue() (bool, bool)    { return false, false }
func (p *PathValue) AsBooleanValue() (bool, bool)       { return false, false }
func (f *FunctionCall) AsBooleanValue() (bool, bool)    { return false, false }
func (a *ArrayExpression) AsBooleanValue() (bool, bool) { return false, false }
