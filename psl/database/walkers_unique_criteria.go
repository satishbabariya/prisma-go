// Package parserdatabase provides UniqueCriteriaWalker for accessing unique criteria data.
package database

// UniqueCriteriaWalker describes any unique criteria in a model.
// Can either be a primary key, or a unique index.
type UniqueCriteriaWalker struct {
	db      *ParserDatabase
	modelID ModelId
	fields  []FieldWithArgs
}

// Fields returns all fields that are part of the unique criteria.
func (w *UniqueCriteriaWalker) Fields() []*IndexFieldWalker {
	var result []*IndexFieldWalker
	for _, fieldWithArgs := range w.fields {
		result = append(result, &IndexFieldWalker{
			db:            w.db,
			modelID:       w.modelID,
			fieldWithArgs: fieldWithArgs,
		})
	}
	return result
}

// IsStrictCriteria returns whether the criteria has only required fields
// and no unsupported fields.
func (w *UniqueCriteriaWalker) IsStrictCriteria() bool {
	return !w.HasOptionalFields() && !w.HasUnsupportedFields()
}

// HasOptionalFields returns whether any field in the criteria is optional.
func (w *UniqueCriteriaWalker) HasOptionalFields() bool {
	for _, field := range w.Fields() {
		scalarField := field.ScalarField()
		if scalarField != nil && scalarField.IsOptional() {
			return true
		}
	}
	return false
}

// ContainsExactlyFields returns whether the criteria contains exactly
// the given scalar fields (by field ID).
func (w *UniqueCriteriaWalker) ContainsExactlyFields(fields []*ScalarFieldWalker) bool {
	if len(w.fields) != len(fields) {
		return false
	}

	criteriaFields := w.Fields()
	for i := range criteriaFields {
		if criteriaFields[i].FieldID() != fields[i].id {
			return false
		}
	}

	return true
}

// HasUnsupportedFields returns whether any field in the criteria is unsupported.
func (w *UniqueCriteriaWalker) HasUnsupportedFields() bool {
	for _, field := range w.Fields() {
		scalarField := field.ScalarField()
		if scalarField != nil {
			sf := scalarField.attributes()
			if sf != nil && sf.Type.Unsupported != nil {
				return true
			}
		}
	}
	return false
}
