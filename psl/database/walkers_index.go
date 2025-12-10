// Package parserdatabase provides IndexWalker for accessing index data.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// IndexWalker provides access to an index (@@index, @@unique, or @@fulltext).
type IndexWalker struct {
	db      *ParserDatabase
	modelID ModelId
	index   *IndexAttribute
}

// Type returns the type of the index (Normal, Unique, or Fulltext).
func (w *IndexWalker) Type() IndexType {
	if w.index == nil {
		return IndexTypeNormal
	}
	return w.index.Type
}

// IsUnique returns whether this is a unique index.
func (w *IndexWalker) IsUnique() bool {
	return w.index != nil && w.index.Type == IndexTypeUnique
}

// IsFulltext returns whether this is a fulltext index.
func (w *IndexWalker) IsFulltext() bool {
	return w.index != nil && w.index.Type == IndexTypeFulltext
}

// Fields returns all fields that are part of the index.
func (w *IndexWalker) Fields() []*IndexFieldWalker {
	if w.index == nil {
		return nil
	}

	var result []*IndexFieldWalker
	for _, fieldWithArgs := range w.index.Fields {
		result = append(result, &IndexFieldWalker{
			db:            w.db,
			modelID:       w.modelID,
			fieldWithArgs: fieldWithArgs,
		})
	}
	return result
}

// Name returns the name of the index if @@unique(name: "...") is present.
func (w *IndexWalker) Name() *string {
	if w.index == nil || w.index.Name == nil {
		return nil
	}
	name := w.db.interner.Get(*w.index.Name)
	return &name
}

// MappedName returns the mapped name if @@index(map: "...") is present.
func (w *IndexWalker) MappedName() *string {
	if w.index == nil || w.index.MappedName == nil {
		return nil
	}
	name := w.db.interner.Get(*w.index.MappedName)
	return &name
}

// Algorithm returns the index algorithm if specified.
func (w *IndexWalker) Algorithm() *IndexAlgorithm {
	if w.index == nil {
		return nil
	}
	return w.index.Algorithm
}

// Model returns the model this index belongs to.
func (w *IndexWalker) Model() *ModelWalker {
	return w.db.WalkModel(w.modelID)
}

// Clustered returns whether the index is clustered if specified.
func (w *IndexWalker) Clustered() *bool {
	if w.index == nil {
		return nil
	}
	return w.index.Clustered
}

// IsNormal returns whether this is a normal index.
func (w *IndexWalker) IsNormal() bool {
	return w.index != nil && w.index.Type == IndexTypeNormal
}

// AttributeName returns the attribute name based on index type.
func (w *IndexWalker) AttributeName() string {
	if w.index == nil {
		return "@@index"
	}
	switch w.index.Type {
	case IndexTypeUnique:
		return "@@unique"
	case IndexTypeFulltext:
		return "@@fulltext"
	default:
		return "@@index"
	}
}

// AstAttribute returns the AST attribute for the index.
func (w *IndexWalker) AstAttribute() *ast.Attribute {
	// Find the attribute in the model's attributes
	model := w.db.WalkModel(w.modelID)
	if model == nil {
		return nil
	}

	astModel := model.AstModel()
	if astModel == nil {
		return nil
	}

	// Look through all attributes to find the matching one
	for _, attr := range astModel.Attributes {
		attrName := attr.Name.Name
		if (attrName == "index" && w.IsNormal()) ||
			(attrName == "unique" && w.IsUnique()) ||
			(attrName == "fulltext" && w.IsFulltext()) {
			return &attr
		}
	}

	return nil
}

// ScalarFieldAttributes returns scalar field attribute walkers for all fields in the index.
func (w *IndexWalker) ScalarFieldAttributes() []*IndexFieldWalker {
	return w.Fields()
}

// IndexFieldWalker provides access to a field in an index.
type IndexFieldWalker struct {
	db            *ParserDatabase
	modelID       ModelId
	fieldWithArgs FieldWithArgs
}

// FieldID returns the scalar field ID.
func (w *IndexFieldWalker) FieldID() ScalarFieldId {
	return w.fieldWithArgs.Field
}

// ScalarField returns the scalar field walker.
func (w *IndexFieldWalker) ScalarField() *ScalarFieldWalker {
	return w.db.WalkScalarField(w.fieldWithArgs.Field)
}

// SortOrder returns the sort order if specified.
func (w *IndexFieldWalker) SortOrder() *SortOrder {
	return w.fieldWithArgs.SortOrder
}

// Length returns the length prefix if specified.
func (w *IndexFieldWalker) Length() *int {
	return w.fieldWithArgs.Length
}

// OperatorClass returns the operator class if specified.
func (w *IndexFieldWalker) OperatorClass() *OperatorClass {
	return w.fieldWithArgs.OperatorClass
}

// ScalarFieldType returns the scalar field type for this field.
func (w *IndexFieldWalker) ScalarFieldType() ScalarFieldType {
	field := w.ScalarField()
	if field == nil {
		return ScalarFieldType{}
	}
	return field.ScalarFieldType()
}
