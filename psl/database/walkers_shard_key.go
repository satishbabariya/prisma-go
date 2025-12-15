// Package parserdatabase provides ShardKeyWalker for accessing shard key data.
package database

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// ShardKeyWalker provides access to a @shardKey or @@shardKey attribute.
type ShardKeyWalker struct {
	db        *ParserDatabase
	modelID   ModelId
	attribute *ShardKeyAttribute
}

// AstAttribute returns the @(@)shardKey AST node.
func (w *ShardKeyWalker) AstAttribute() *v2ast.Attribute {
	if w.attribute == nil {
		return nil
	}

	model := w.db.WalkModel(w.modelID)
	if model == nil {
		return nil
	}

	astModel := model.AstModel()
	if astModel == nil {
		return nil
	}

	// If defined on a field, search field attributes
	if w.attribute.SourceField != nil {
		if int(*w.attribute.SourceField) < len(astModel.Fields) {
			field := astModel.Fields[*w.attribute.SourceField]
			if field != nil {
				for _, attr := range field.Attributes {
					if attr != nil && attr.GetName() == "shardKey" {
					return attr
					}
				}
			}
		}
		return nil
	}

	// Otherwise, search model block attributes
	for _, attr := range astModel.BlockAttributes {
		if attr != nil && attr.GetName() == "shardKey" {
			return &v2ast.Attribute{
				Pos:       attr.Pos,
				Name:      attr.Name,
				Arguments: attr.Arguments,
			}
		}
	}

	return nil
}

// IsDefinedOnField returns whether this is a @shardKey on a specific field,
// rather than @@shardKey on the model.
func (w *ShardKeyWalker) IsDefinedOnField() bool {
	if w.attribute == nil {
		return false
	}
	return w.attribute.SourceField != nil
}

// AttributeName returns "@shardKey" if defined on a field, otherwise "@@shardKey".
func (w *ShardKeyWalker) AttributeName() string {
	if w.IsDefinedOnField() {
		return "@shardKey"
	}
	return "@@shardKey"
}

// Model returns the model the shard key is defined on.
func (w *ShardKeyWalker) Model() *ModelWalker {
	return w.db.WalkModel(w.modelID)
}

// Fields returns the scalar fields used as the shard key.
func (w *ShardKeyWalker) Fields() []*ScalarFieldWalker {
	if w.attribute == nil {
		return nil
	}

	var result []*ScalarFieldWalker
	for _, fieldID := range w.attribute.Fields {
		result = append(result, w.db.WalkScalarField(fieldID))
	}
	return result
}
