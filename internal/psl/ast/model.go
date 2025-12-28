package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Model represents a model or view declaration.
type Model struct {
	Pos             lexer.Position
	Documentation   *CommentBlock     `@@?`
	Keyword         string            `@("model" | "view")`
	Name            *Identifier       `@@`
	Fields          []*Field          `"{" @@* `
	BlockAttributes []*BlockAttribute `@@* "}"`
}

// IsView returns true if this is a view declaration.
func (m *Model) IsView() bool {
	return m.Keyword == "view"
}

// GetName returns the model name.
func (m *Model) GetName() string {
	if m.Name == nil {
		return ""
	}
	return m.Name.Name
}

// IterFields returns an iterator over fields with their IDs.
func (m *Model) IterFields() []FieldWithID {
	result := make([]FieldWithID, len(m.Fields))
	for i, f := range m.Fields {
		result[i] = FieldWithID{ID: FieldID(i), Field: f}
	}
	return result
}

// FieldWithID pairs a field with its ID.
type FieldWithID struct {
	ID    FieldID
	Field *Field
}

// GetField returns a field by ID.
func (m *Model) GetField(id FieldID) *Field {
	if int(id) < 0 || int(id) >= len(m.Fields) {
		return nil
	}
	return m.Fields[id]
}

// GetDocumentation returns the model documentation.
func (m *Model) GetDocumentation() string {
	if m.Documentation == nil {
		return ""
	}
	return m.Documentation.GetText()
}

// ModelID is an opaque identifier for a model in the schema.
type ModelID int

// ModelIDZero is the zero value for ModelID.
const ModelIDZero ModelID = 0

// ModelIDMax is the maximum ModelID value.
const ModelIDMax ModelID = 1<<31 - 1
