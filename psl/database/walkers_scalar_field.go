// Package parserdatabase provides ScalarFieldWalker for accessing scalar field data.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// ScalarFieldWalker provides access to a scalar field in a model.
type ScalarFieldWalker struct {
	db *ParserDatabase
	id ScalarFieldId
}

// Name returns the name of the field.
func (w *ScalarFieldWalker) Name() string {
	astField := w.AstField()
	if astField == nil {
		return ""
	}
	return astField.Name.Name
}

// FieldID returns the field ID in the AST.
func (w *ScalarFieldWalker) FieldID() uint32 {
	sf := w.attributes()
	if sf == nil {
		return 0
	}
	return sf.FieldID
}

// AstField returns the AST node for the field.
func (w *ScalarFieldWalker) AstField() *ast.Field {
	sf := w.attributes()
	if sf == nil {
		return nil
	}

	astModel := getModelFromID(sf.ModelID, &Context{asts: &w.db.asts})
	if astModel == nil || int(sf.FieldID) >= len(astModel.Fields) {
		return nil
	}
	return &astModel.Fields[sf.FieldID]
}

// Model returns the parent model walker.
func (w *ScalarFieldWalker) Model() *ModelWalker {
	sf := w.attributes()
	if sf == nil {
		return nil
	}
	return w.db.WalkModel(sf.ModelID)
}

// attributes returns the scalar field attributes.
func (w *ScalarFieldWalker) attributes() *ScalarField {
	if int(w.id) >= len(w.db.types.ScalarFields) {
		return nil
	}
	return &w.db.types.ScalarFields[w.id]
}

// DatabaseName returns the final database name of the field.
func (w *ScalarFieldWalker) DatabaseName() string {
	sf := w.attributes()
	if sf == nil {
		return w.Name()
	}
	if sf.MappedName != nil {
		return w.db.interner.Get(*sf.MappedName)
	}
	return w.Name()
}

// IsIgnored returns whether the field has an @ignore attribute.
func (w *ScalarFieldWalker) IsIgnored() bool {
	sf := w.attributes()
	if sf == nil {
		return false
	}
	return sf.IsIgnored
}

// IsUpdatedAt returns whether the field has an @updatedAt attribute.
func (w *ScalarFieldWalker) IsUpdatedAt() bool {
	sf := w.attributes()
	if sf == nil {
		return false
	}
	return sf.IsUpdatedAt
}

// IsOptional returns whether the field is optional/nullable.
func (w *ScalarFieldWalker) IsOptional() bool {
	astField := w.AstField()
	if astField == nil {
		return false
	}
	return astField.FieldType.IsOptional()
}

// IsList returns whether the field is a list.
func (w *ScalarFieldWalker) IsList() bool {
	astField := w.AstField()
	if astField == nil {
		return false
	}
	return astField.FieldType.IsArray()
}

// ScalarFieldType returns the type of the scalar field.
func (w *ScalarFieldWalker) ScalarFieldType() ScalarFieldType {
	sf := w.attributes()
	if sf == nil {
		return ScalarFieldType{}
	}
	return sf.Type
}

// ScalarType returns the scalar type if the field is a built-in scalar.
func (w *ScalarFieldWalker) ScalarType() *ScalarType {
	sf := w.attributes()
	if sf == nil {
		return nil
	}
	return sf.Type.BuiltInScalar
}

// DefaultValue returns the default value walker if @default is present.
func (w *ScalarFieldWalker) DefaultValue() *DefaultValueWalker {
	sf := w.attributes()
	if sf == nil || sf.Default == nil {
		return nil
	}
	return &DefaultValueWalker{
		db:          w.db,
		fieldID:     w.id,
		defaultAttr: sf.Default,
	}
}

// NativeType returns the native type information if present.
func (w *ScalarFieldWalker) NativeType() *NativeTypeInfo {
	sf := w.attributes()
	if sf == nil {
		return nil
	}
	return sf.NativeType
}

// IsSinglePK returns whether this field defines a primary key by itself.
func (w *ScalarFieldWalker) IsSinglePK() bool {
	model := w.Model()
	if model == nil {
		return false
	}
	pk := model.PrimaryKey()
	if pk == nil {
		return false
	}
	fields := pk.Fields()
	return len(fields) == 1 && fields[0].FieldID() == w.id
}

// IsPartOfCompoundPK returns whether the field is part of a compound primary key.
func (w *ScalarFieldWalker) IsPartOfCompoundPK() bool {
	model := w.Model()
	if model == nil {
		return false
	}
	pk := model.PrimaryKey()
	if pk == nil {
		return false
	}
	fields := pk.Fields()
	if len(fields) <= 1 {
		return false
	}
	for _, f := range fields {
		if f.FieldID() == w.id {
			return true
		}
	}
	return false
}
