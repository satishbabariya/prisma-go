// Package parserdatabase provides PrimaryKeyWalker for accessing primary key data.
package database

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// PrimaryKeyWalker provides access to a primary key (@id or @@id).
type PrimaryKeyWalker struct {
	db      *ParserDatabase
	modelID ModelId
	pk      *IdAttribute
}

// Fields returns all fields that are part of the primary key.
func (w *PrimaryKeyWalker) Fields() []*PrimaryKeyFieldWalker {
	if w.pk == nil {
		return nil
	}

	var result []*PrimaryKeyFieldWalker
	for _, fieldWithArgs := range w.pk.Fields {
		result = append(result, &PrimaryKeyFieldWalker{
			db:            w.db,
			modelID:       w.modelID,
			fieldWithArgs: fieldWithArgs,
		})
	}
	return result
}

// ContainsExactlyFieldsByID returns whether the primary key contains exactly the given field IDs.
func (w *PrimaryKeyWalker) ContainsExactlyFieldsByID(fieldIDs []ScalarFieldId) bool {
	if w.pk == nil {
		return false
	}
	if len(w.pk.Fields) != len(fieldIDs) {
		return false
	}
	for i, fieldWithArgs := range w.pk.Fields {
		if fieldWithArgs.Field != fieldIDs[i] {
			return false
		}
	}
	return true
}

// Name returns the name of the primary key if @@id(name: "...") is present.
func (w *PrimaryKeyWalker) Name() *string {
	if w.pk == nil || w.pk.Name == nil {
		return nil
	}
	name := w.db.interner.Get(*w.pk.Name)
	return &name
}

// MappedName returns the mapped name if @@id(map: "...") or @id(map: "...") is present.
func (w *PrimaryKeyWalker) MappedName() *string {
	if w.pk == nil || w.pk.MappedName == nil {
		return nil
	}
	name := w.db.interner.Get(*w.pk.MappedName)
	return &name
}

// SourceField returns the source field ID if this is an @id (not @@id).
func (w *PrimaryKeyWalker) SourceField() *uint32 {
	if w.pk == nil {
		return nil
	}
	return w.pk.SourceField
}

// Model returns the model this primary key belongs to.
func (w *PrimaryKeyWalker) Model() *ModelWalker {
	return w.db.WalkModel(w.modelID)
}

// AstAttribute returns the AST attribute for this primary key.
func (w *PrimaryKeyWalker) AstAttribute() *v2ast.Attribute {
	model := w.Model()
	if model == nil {
		return nil
	}
	astModel := model.AstModel()
	if astModel == nil {
		return nil
	}
	// Find the @@id or @id attribute
	// Check block attributes first (for @@id)
	for _, attr := range astModel.BlockAttributes {
		if attr != nil && attr.GetName() == "id" {
			return &v2ast.Attribute{
				Pos:       attr.Pos,
				Name:      attr.Name,
				Arguments: attr.Arguments,
			}
		}
	}
	// Check field attributes (for @id) - would need to search through fields
	// For now, return nil as field-level attributes are harder to locate
	return nil
}

// Clustered returns whether the primary key is clustered if specified.
func (w *PrimaryKeyWalker) Clustered() *bool {
	if w.pk == nil {
		return nil
	}
	return w.pk.Clustered
}

// AttributeName returns the attribute name based on whether it's @id or @@id.
func (w *PrimaryKeyWalker) AttributeName() string {
	if w.pk == nil {
		return "@id"
	}
	if w.pk.SourceField != nil {
		return "@id"
	}
	return "@@id"
}

// ScalarFieldAttributes returns scalar field attribute walkers for all fields in the primary key.
func (w *PrimaryKeyWalker) ScalarFieldAttributes() []*PrimaryKeyFieldWalker {
	return w.Fields()
}

// PrimaryKeyFieldWalker provides access to a field in a primary key.
type PrimaryKeyFieldWalker struct {
	db            *ParserDatabase
	modelID       ModelId
	fieldWithArgs FieldWithArgs
}

// FieldID returns the scalar field ID.
func (w *PrimaryKeyFieldWalker) FieldID() ScalarFieldId {
	return w.fieldWithArgs.Field
}

// ScalarField returns the scalar field walker.
func (w *PrimaryKeyFieldWalker) ScalarField() *ScalarFieldWalker {
	return w.db.WalkScalarField(w.fieldWithArgs.Field)
}

// SortOrder returns the sort order if specified.
func (w *PrimaryKeyFieldWalker) SortOrder() *SortOrder {
	return w.fieldWithArgs.SortOrder
}

// Length returns the length prefix if specified (for primary key fields).
func (w *PrimaryKeyFieldWalker) Length() *int {
	return w.fieldWithArgs.Length
}

// ScalarFieldType returns the scalar field type for this field.
func (w *PrimaryKeyFieldWalker) ScalarFieldType() ScalarFieldType {
	field := w.ScalarField()
	if field == nil {
		return ScalarFieldType{}
	}
	return field.ScalarFieldType()
}
