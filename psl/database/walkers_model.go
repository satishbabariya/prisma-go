// Package parserdatabase provides ModelWalker for accessing model data.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// ModelWalker provides access to a model declaration in the Prisma schema.
type ModelWalker struct {
	db *ParserDatabase
	id ModelId
}

// Name returns the name of the model.
func (w *ModelWalker) Name() string {
	model := w.AstModel()
	if model == nil {
		return ""
	}
	return model.GetName()
}

// FileID returns the ID of the file containing the model.
func (w *ModelWalker) FileID() diagnostics.FileID {
	return w.id.FileID
}

// AstModel returns the AST node for the model.
func (w *ModelWalker) AstModel() *v2ast.Model {
	file := w.db.asts.Get(w.id.FileID)
	if file == nil {
		return nil
	}

	modelCount := 0
	for _, top := range file.AST.Tops {
		if model, ok := top.(*v2ast.Model); ok {
			if uint32(modelCount) == w.id.ID {
				return model
			}
			modelCount++
		}
	}

	return nil
}

// Attributes returns the parsed attributes for the model.
func (w *ModelWalker) Attributes() *ModelAttributes {
	attrs, exists := w.db.types.ModelAttributes[w.id]
	if !exists {
		return nil
	}
	return &attrs
}

// IsIgnored returns whether the model has the @@ignore attribute.
func (w *ModelWalker) IsIgnored() bool {
	attrs := w.Attributes()
	if attrs == nil {
		return false
	}
	return attrs.IsIgnored
}

// DatabaseName returns the name of the database table the model points to.
func (w *ModelWalker) DatabaseName() string {
	attrs := w.Attributes()
	if attrs == nil || attrs.MappedName == nil {
		return w.Name()
	}
	return w.db.interner.Get(*attrs.MappedName)
}

// Schema returns the schema name if @@schema is present.
func (w *ModelWalker) Schema() *SchemaInfo {
	attrs := w.Attributes()
	if attrs == nil {
		return nil
	}
	return attrs.Schema
}

// PrimaryKey returns the primary key walker if @@id or @id is present.
func (w *ModelWalker) PrimaryKey() *PrimaryKeyWalker {
	attrs := w.Attributes()
	if attrs == nil || attrs.PrimaryKey == nil {
		return nil
	}
	return &PrimaryKeyWalker{
		db:      w.db,
		modelID: w.id,
		pk:      attrs.PrimaryKey,
	}
}

// Fields returns all fields in the model.
func (w *ModelWalker) Fields() []*FieldWalker {
	astModel := w.AstModel()
	if astModel == nil {
		return nil
	}

	var result []*FieldWalker
	for i := range astModel.Fields {
		fieldID := uint32(i)
		result = append(result, &FieldWalker{
			db:      w.db,
			modelID: w.id,
			fieldID: fieldID,
		})
	}
	return result
}

// Field returns a specific field by field ID.
func (w *ModelWalker) Field(fieldID uint32) *FieldWalker {
	return &FieldWalker{
		db:      w.db,
		modelID: w.id,
		fieldID: fieldID,
	}
}

// ScalarFields returns all scalar fields in the model.
func (w *ModelWalker) ScalarFields() []*ScalarFieldWalker {
	var result []*ScalarFieldWalker
	for _, entry := range w.db.types.RangeModelScalarFields(w.id) {
		result = append(result, w.db.WalkScalarField(entry.ID))
	}
	return result
}

// RelationFields returns all relation fields in the model.
func (w *ModelWalker) RelationFields() []*RelationFieldWalker {
	var result []*RelationFieldWalker
	for _, entry := range w.db.types.RangeModelRelationFields(w.id) {
		result = append(result, w.db.WalkRelationField(entry.ID))
	}
	return result
}

// Indexes returns all indexes on the model.
func (w *ModelWalker) Indexes() []*IndexWalker {
	attrs := w.Attributes()
	if attrs == nil {
		return nil
	}

	var result []*IndexWalker
	for _, entry := range attrs.AstIndexes {
		result = append(result, &IndexWalker{
			db:      w.db,
			modelID: w.id,
			index:   &entry.Index,
		})
	}
	return result
}

// IsDefinedInFile returns whether the model is defined in the given file.
func (w *ModelWalker) IsDefinedInFile(fileID diagnostics.FileID) bool {
	return w.id.FileID == fileID
}

// ShardKey returns the shard key walker if @shardKey or @@shardKey is present.
func (w *ModelWalker) ShardKey() *ShardKeyWalker {
	attrs := w.Attributes()
	if attrs == nil || attrs.ShardKey == nil {
		return nil
	}
	return &ShardKeyWalker{
		db:        w.db,
		modelID:   w.id,
		attribute: attrs.ShardKey,
	}
}

// UniqueCriterias returns all unique criterias of the model, consisting of
// the primary key and unique indexes, if set.
func (w *ModelWalker) UniqueCriterias() []*UniqueCriteriaWalker {
	var result []*UniqueCriteriaWalker

	// Add primary key if present
	pk := w.PrimaryKey()
	if pk != nil && pk.pk != nil {
		result = append(result, &UniqueCriteriaWalker{
			db:      w.db,
			modelID: w.id,
			fields:  pk.pk.Fields,
		})
	}

	// Add unique indexes
	for _, index := range w.Indexes() {
		if index.IsUnique() && index.index != nil {
			result = append(result, &UniqueCriteriaWalker{
				db:      w.db,
				modelID: w.id,
				fields:  index.index.Fields,
			})
		}
	}

	return result
}

// RequiredUniqueCriterias returns all required unique criterias of the model,
// consisting of the primary key and unique indexes that have only required fields.
func (w *ModelWalker) RequiredUniqueCriterias() []*UniqueCriteriaWalker {
	var result []*UniqueCriteriaWalker
	for _, criteria := range w.UniqueCriterias() {
		if !criteria.HasOptionalFields() {
			result = append(result, criteria)
		}
	}
	return result
}

// FieldIsIndexedForAutoincrement returns whether the field is indexed for autoincrement purposes.
// A field is considered indexed if it's part of any index (including primary key).
func (w *ModelWalker) FieldIsIndexedForAutoincrement(fieldID ScalarFieldId) bool {
	// Check if field is part of primary key
	pk := w.PrimaryKey()
	if pk != nil {
		for _, pkField := range pk.Fields() {
			if pkField.FieldID() == fieldID {
				return true
			}
		}
	}

	// Check if field is part of any index
	for _, index := range w.Indexes() {
		for _, indexField := range index.Fields() {
			if indexField.FieldID() == fieldID {
				return true
			}
		}
	}

	return false
}

// IsView returns whether this model is a view (as opposed to a table).
func (w *ModelWalker) IsView() bool {
	astModel := w.AstModel()
	if astModel == nil {
		return false
	}
	return astModel.IsView()
}
