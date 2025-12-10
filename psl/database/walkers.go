// Package parserdatabase provides walkers for convenient access to parsed schema data.
package database

// Walker is a generic walker that provides access to parsed schema elements.
// It holds a reference to the ParserDatabase and an identifier.
type Walker struct {
	db *ParserDatabase
	id interface{} // The identifier (ModelId, EnumId, ScalarFieldId, etc.)
}

// Walk creates a new walker for the given ID.
func (pd *ParserDatabase) Walk(id interface{}) *Walker {
	return &Walker{
		db: pd,
		id: id,
	}
}

// WalkModel creates a ModelWalker for the given ModelId.
func (pd *ParserDatabase) WalkModel(id ModelId) *ModelWalker {
	return &ModelWalker{
		db: pd,
		id: id,
	}
}

// WalkEnum creates an EnumWalker for the given EnumId.
func (pd *ParserDatabase) WalkEnum(id EnumId) *EnumWalker {
	return &EnumWalker{
		db: pd,
		id: id,
	}
}

// WalkScalarField creates a ScalarFieldWalker for the given ScalarFieldId.
func (pd *ParserDatabase) WalkScalarField(id ScalarFieldId) *ScalarFieldWalker {
	return &ScalarFieldWalker{
		db: pd,
		id: id,
	}
}

// WalkRelationField creates a RelationFieldWalker for the given RelationFieldId.
func (pd *ParserDatabase) WalkRelationField(id RelationFieldId) *RelationFieldWalker {
	return &RelationFieldWalker{
		db: pd,
		id: id,
	}
}

// WalkCompositeType creates a CompositeTypeWalker for the given CompositeTypeId.
func (pd *ParserDatabase) WalkCompositeType(id CompositeTypeId) *CompositeTypeWalker {
	return &CompositeTypeWalker{
		db: pd,
		id: id,
	}
}

// FindModel finds a model by name.
func (pd *ParserDatabase) FindModel(name string) *ModelWalker {
	nameID, found := pd.interner.Lookup(name)
	if !found {
		return nil
	}

	topID, exists := pd.names.Tops[nameID]
	if !exists {
		return nil
	}

	// Check if it's a model by looking at the AST
	file := pd.asts.Get(topID.FileID)
	if file == nil || int(topID.ID) >= len(file.AST.Tops) {
		return nil
	}
	top := file.AST.Tops[topID.ID]
	if top.AsModel() == nil {
		return nil
	}

	modelID := ModelId{
		FileID: topID.FileID,
		ID:     topID.ID,
	}

	return pd.WalkModel(modelID)
}

// FindEnum finds an enum by name.
func (pd *ParserDatabase) FindEnum(name string) *EnumWalker {
	nameID, found := pd.interner.Lookup(name)
	if !found {
		return nil
	}

	topID, exists := pd.names.Tops[nameID]
	if !exists {
		return nil
	}

	// Check if it's an enum by looking at the AST
	file := pd.asts.Get(topID.FileID)
	if file == nil || int(topID.ID) >= len(file.AST.Tops) {
		return nil
	}
	top := file.AST.Tops[topID.ID]
	if top.AsEnum() == nil {
		return nil
	}

	enumID := EnumId{
		FileID: topID.FileID,
		ID:     topID.ID,
	}

	return pd.WalkEnum(enumID)
}

// FindCompositeType finds a composite type by name.
func (pd *ParserDatabase) FindCompositeType(name string) *CompositeTypeWalker {
	nameID, found := pd.interner.Lookup(name)
	if !found {
		return nil
	}

	topID, exists := pd.names.Tops[nameID]
	if !exists {
		return nil
	}

	// Check if it's a composite type by looking at the AST
	file := pd.asts.Get(topID.FileID)
	if file == nil || int(topID.ID) >= len(file.AST.Tops) {
		return nil
	}
	top := file.AST.Tops[topID.ID]
	if top.AsCompositeType() == nil {
		return nil
	}

	ctID := CompositeTypeId{
		FileID: topID.FileID,
		ID:     topID.ID,
	}

	return pd.WalkCompositeType(ctID)
}

// WalkModels returns an iterator over all models in the schema.
func (pd *ParserDatabase) WalkModels() []*ModelWalker {
	var result []*ModelWalker

	for _, file := range pd.asts.files {
		modelCount := 0
		for _, top := range file.AST.Tops {
			if model := top.AsModel(); model != nil {
				modelID := ModelId{
					FileID: file.FileID,
					ID:     uint32(modelCount),
				}
				result = append(result, pd.WalkModel(modelID))
				modelCount++
			}
		}
	}

	return result
}

// WalkEnums returns an iterator over all enums in the schema.
func (pd *ParserDatabase) WalkEnums() []*EnumWalker {
	var result []*EnumWalker

	for _, file := range pd.asts.files {
		enumCount := 0
		for _, top := range file.AST.Tops {
			if enum := top.AsEnum(); enum != nil {
				enumID := EnumId{
					FileID: file.FileID,
					ID:     uint32(enumCount),
				}
				result = append(result, pd.WalkEnum(enumID))
				enumCount++
			}
		}
	}

	return result
}

// WalkCompositeTypes returns an iterator over all composite types in the schema.
func (pd *ParserDatabase) WalkCompositeTypes() []*CompositeTypeWalker {
	var result []*CompositeTypeWalker

	for _, file := range pd.asts.files {
		ctCount := 0
		for _, top := range file.AST.Tops {
			if ct := top.AsCompositeType(); ct != nil {
				ctID := CompositeTypeId{
					FileID: file.FileID,
					ID:     uint32(ctCount),
				}
				result = append(result, pd.WalkCompositeType(ctID))
				ctCount++
			}
		}
	}

	return result
}
