// Package parserdatabase provides FieldWalker for accessing field data.
package database

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// FieldWalker provides access to a model field (scalar or relation).
type FieldWalker struct {
	db      *ParserDatabase
	modelID ModelId
	fieldID uint32
}

// Name returns the name of the field.
func (w *FieldWalker) Name() string {
	astField := w.AstField()
	if astField == nil {
		return ""
	}
	return astField.GetName()
}

// AstField returns the AST node for the field.
func (w *FieldWalker) AstField() *v2ast.Field {
	astModel := w.Model().AstModel()
	if astModel == nil || int(w.fieldID) >= len(astModel.Fields) {
		return nil
	}
	return astModel.Fields[w.fieldID]
}

// Model returns the parent model walker.
func (w *FieldWalker) Model() *ModelWalker {
	return w.db.WalkModel(w.modelID)
}

// Refine determines whether this field is a scalar field or relation field.
func (w *FieldWalker) Refine() *RefinedFieldWalker {
	variant := w.db.types.RefineField(w.modelID, w.fieldID)
	switch variant {
	case RefinedFieldVariantScalar:
		sfid := w.db.types.FindModelScalarField(w.modelID, w.fieldID)
		if sfid != nil {
			return &RefinedFieldWalker{
				IsScalar: true,
				Scalar:   w.db.WalkScalarField(*sfid),
			}
		}
	case RefinedFieldVariantRelation:
		// Find relation field ID
		for _, entry := range w.db.types.RangeModelRelationFields(w.modelID) {
			if entry.Field.FieldID == w.fieldID {
				return &RefinedFieldWalker{
					IsScalar: false,
					Relation: w.db.WalkRelationField(entry.ID),
				}
			}
		}
	}
	return nil
}

// RefinedFieldWalker represents a field that has been identified as scalar or relation.
type RefinedFieldWalker struct {
	IsScalar bool
	Scalar   *ScalarFieldWalker
	Relation *RelationFieldWalker
}
