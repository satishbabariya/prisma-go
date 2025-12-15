// Package parserdatabase provides EnumWalker for accessing enum data.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// EnumWalker provides access to an enum declaration in the Prisma schema.
type EnumWalker struct {
	db *ParserDatabase
	id EnumId
}

// Name returns the name of the enum.
func (w *EnumWalker) Name() string {
	astEnum := w.AstEnum()
	if astEnum == nil {
		return ""
	}
	return astEnum.GetName()
}

// FileID returns the ID of the file containing the enum.
func (w *EnumWalker) FileID() diagnostics.FileID {
	return w.id.FileID
}

// AstEnum returns the AST node for the enum.
func (w *EnumWalker) AstEnum() *v2ast.Enum {
	file := w.db.asts.Get(w.id.FileID)
	if file == nil {
		return nil
	}

	enumCount := 0
	for _, top := range file.AST.Tops {
		if enum, ok := top.(*v2ast.Enum); ok {
			if uint32(enumCount) == w.id.ID {
				return enum
			}
			enumCount++
		}
	}

	return nil
}

// Attributes returns the parsed attributes for the enum.
func (w *EnumWalker) Attributes() *EnumAttributes {
	attrs, exists := w.db.types.EnumAttributes[w.id]
	if !exists {
		return nil
	}
	return &attrs
}

// DatabaseName returns the name of the database enum the enum points to.
func (w *EnumWalker) DatabaseName() string {
	attrs := w.Attributes()
	if attrs == nil || attrs.MappedName == nil {
		return w.Name()
	}
	return w.db.interner.Get(*attrs.MappedName)
}

// Values returns all enum values.
func (w *EnumWalker) Values() []*EnumValueWalker {
	astEnum := w.AstEnum()
	if astEnum == nil {
		return nil
	}

	var result []*EnumValueWalker
	for i := range astEnum.Values {
		valueID := uint32(i)
		result = append(result, &EnumValueWalker{
			db:      w.db,
			enumID:  w.id,
			valueID: valueID,
		})
	}
	return result
}

// IsDefinedInFile returns whether the enum is defined in the given file.
func (w *EnumWalker) IsDefinedInFile(fileID diagnostics.FileID) bool {
	return w.id.FileID == fileID
}

// Schema returns the schema name if @@schema is present.
func (w *EnumWalker) Schema() *SchemaInfo {
	attrs := w.Attributes()
	if attrs == nil {
		return nil
	}
	return attrs.Schema
}

// EnumValueWalker provides access to an enum value.
type EnumValueWalker struct {
	db      *ParserDatabase
	enumID  EnumId
	valueID uint32
}

// Name returns the name of the enum value.
func (w *EnumValueWalker) Name() string {
	astEnum := w.db.WalkEnum(w.enumID).AstEnum()
	if astEnum == nil || int(w.valueID) >= len(astEnum.Values) {
		return ""
	}
	value := astEnum.Values[w.valueID]
	if value == nil {
		return ""
	}
	return value.GetName()
}

// DatabaseName returns the database name of the enum value if @map is present.
func (w *EnumValueWalker) DatabaseName() string {
	attrs := w.db.WalkEnum(w.enumID).Attributes()
	if attrs == nil || attrs.MappedValues == nil {
		return w.Name()
	}
	mappedNameID, exists := attrs.MappedValues[w.valueID]
	if !exists {
		return w.Name()
	}
	return w.db.interner.Get(mappedNameID)
}
