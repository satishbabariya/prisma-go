// Package builder provides select builder functionality.
package builder

// SelectBuilder builds select clauses for fields
type SelectBuilder struct {
	fields map[string]bool
}

// NewSelectBuilder creates a new select builder
func NewSelectBuilder() *SelectBuilder {
	return &SelectBuilder{
		fields: make(map[string]bool),
	}
}

// Field adds a field to select
func (s *SelectBuilder) Field(field string) *SelectBuilder {
	s.fields[field] = true
	return s
}

// GetFields returns the selected fields
func (s *SelectBuilder) GetFields() map[string]bool {
	return s.fields
}

// HasFields returns true if there are any fields selected
func (s *SelectBuilder) HasFields() bool {
	return len(s.fields) > 0
}
