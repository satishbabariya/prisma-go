// Package parserdatabase provides RelationFieldWalker for accessing relation field data.
package database

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// RelationFieldWalker provides access to a relation field in a model.
type RelationFieldWalker struct {
	db *ParserDatabase
	id RelationFieldId
}

// Name returns the name of the field.
func (w *RelationFieldWalker) Name() string {
	astField := w.AstField()
	if astField == nil {
		return ""
	}
	return astField.GetName()
}

// FieldID returns the field ID in the AST.
func (w *RelationFieldWalker) FieldID() uint32 {
	rf := w.attributes()
	if rf == nil {
		return 0
	}
	return rf.FieldID
}

// AstField returns the AST node for the field.
func (w *RelationFieldWalker) AstField() *v2ast.Field {
	rf := w.attributes()
	if rf == nil {
		return nil
	}

	astModel := w.db.getModelFromID(rf.ModelID)
	if astModel == nil || int(rf.FieldID) >= len(astModel.Fields) {
		return nil
	}
	return astModel.Fields[rf.FieldID]
}

// Model returns the parent model walker.
func (w *RelationFieldWalker) Model() *ModelWalker {
	rf := w.attributes()
	if rf == nil {
		return nil
	}
	return w.db.WalkModel(rf.ModelID)
}

// ReferencedModel returns the model this field references.
func (w *RelationFieldWalker) ReferencedModel() *ModelWalker {
	rf := w.attributes()
	if rf == nil {
		return nil
	}
	return w.db.WalkModel(rf.ReferencedModel)
}

// IsIgnored returns whether the field has an @ignore attribute.
func (w *RelationFieldWalker) IsIgnored() bool {
	rf := w.attributes()
	if rf == nil {
		return false
	}
	return rf.IsIgnored
}

// IsRequired returns whether the field is required (not optional).
func (w *RelationFieldWalker) IsRequired() bool {
	astField := w.AstField()
	if astField == nil {
		return false
	}
	return !astField.Arity.IsOptional()
}

// ReferencingFields returns the scalar fields that reference the related model (fields argument).
func (w *RelationFieldWalker) ReferencingFields() []*ScalarFieldWalker {
	rf := w.attributes()
	if rf == nil || rf.Fields == nil {
		return nil
	}

	var result []*ScalarFieldWalker
	for _, fieldID := range *rf.Fields {
		result = append(result, w.db.WalkScalarField(fieldID))
	}
	return result
}

// ReferencedFields returns the scalar fields that are referenced in the related model (references argument).
func (w *RelationFieldWalker) ReferencedFields() []*ScalarFieldWalker {
	rf := w.attributes()
	if rf == nil || rf.References == nil {
		return nil
	}

	var result []*ScalarFieldWalker
	for _, fieldID := range *rf.References {
		result = append(result, w.db.WalkScalarField(fieldID))
	}
	return result
}

// attributes returns the relation field attributes.
func (w *RelationFieldWalker) attributes() *RelationField {
	if int(w.id) >= len(w.db.types.RelationFields) {
		return nil
	}
	return &w.db.types.RelationFields[w.id]
}

// OnDelete returns the onDelete referential action if specified.
func (w *RelationFieldWalker) OnDelete() *ReferentialActionInfo {
	rf := w.attributes()
	if rf == nil {
		return nil
	}
	return rf.OnDelete
}

// OnUpdate returns the onUpdate referential action if specified.
func (w *RelationFieldWalker) OnUpdate() *ReferentialActionInfo {
	rf := w.attributes()
	if rf == nil {
		return nil
	}
	return rf.OnUpdate
}

// Relation returns the RelationWalker for this relation field.
func (w *RelationFieldWalker) Relation() *RelationWalker {
	rf := w.attributes()
	if rf == nil {
		return nil
	}
	// Use the fields map to efficiently find the relation ID
	relationID, ok := w.db.relations.GetRelationID(w.id)
	if !ok {
		return nil
	}
	return w.db.WalkRelation(relationID)
}

// MappedName returns the mapped name from the @map attribute on the relation field.
func (w *RelationFieldWalker) MappedName() *string {
	rf := w.attributes()
	if rf == nil || rf.MappedName == nil {
		return nil
	}
	name := w.db.interner.Get(*rf.MappedName)
	return &name
}

// ExplicitOnDelete returns whether onDelete was explicitly specified.
func (w *RelationFieldWalker) ExplicitOnDelete() bool {
	return w.OnDelete() != nil
}

// ExplicitOnUpdate returns whether onUpdate was explicitly specified.
func (w *RelationFieldWalker) ExplicitOnUpdate() bool {
	return w.OnUpdate() != nil
}
