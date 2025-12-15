// Package parserdatabase provides CompositeTypeWalker for accessing composite type data.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// CompositeTypeWalker provides access to a composite type declaration in the Prisma schema.
type CompositeTypeWalker struct {
	db *ParserDatabase
	id CompositeTypeId
}

// Name returns the name of the composite type.
func (w *CompositeTypeWalker) Name() string {
	astCT := w.AstCompositeType()
	if astCT == nil {
		return ""
	}
	return astCT.GetName()
}

// FileID returns the ID of the file containing the composite type.
func (w *CompositeTypeWalker) FileID() diagnostics.FileID {
	return w.id.FileID
}

// AstCompositeType returns the AST node for the composite type.
func (w *CompositeTypeWalker) AstCompositeType() *v2ast.CompositeType {
	file := w.db.asts.Get(w.id.FileID)
	if file == nil {
		return nil
	}

	ctCount := 0
	for _, top := range file.AST.Tops {
		if ct, ok := top.(*v2ast.CompositeType); ok {
			if uint32(ctCount) == w.id.ID {
				return ct
			}
			ctCount++
		}
	}

	return nil
}

// Fields returns all fields in the composite type.
func (w *CompositeTypeWalker) Fields() []*CompositeTypeFieldWalker {
	astCT := w.AstCompositeType()
	if astCT == nil {
		return nil
	}

	var result []*CompositeTypeFieldWalker
	for i := range astCT.Fields {
		fieldID := uint32(i)
		key := CompositeTypeFieldKeyByID{
			CompositeTypeID: w.id,
			FieldID:         fieldID,
		}
		if _, exists := w.db.types.CompositeTypeFields[key]; exists {
			result = append(result, &CompositeTypeFieldWalker{
				db:      w.db,
				ctID:    w.id,
				fieldID: fieldID,
			})
		}
	}
	return result
}

// Field returns a specific field by field ID.
func (w *CompositeTypeWalker) Field(fieldID uint32) *CompositeTypeFieldWalker {
	return &CompositeTypeFieldWalker{
		db:      w.db,
		ctID:    w.id,
		fieldID: fieldID,
	}
}

// IsDefinedInFile returns whether the composite type is defined in the given file.
func (w *CompositeTypeWalker) IsDefinedInFile(fileID diagnostics.FileID) bool {
	return w.id.FileID == fileID
}

// CompositeTypeFieldWalker provides access to a field in a composite type.
type CompositeTypeFieldWalker struct {
	db      *ParserDatabase
	ctID    CompositeTypeId
	fieldID uint32
}

// Name returns the name of the field.
func (w *CompositeTypeFieldWalker) Name() string {
	astCT := w.db.WalkCompositeType(w.ctID).AstCompositeType()
	if astCT == nil || int(w.fieldID) >= len(astCT.Fields) {
		return ""
	}
	field := astCT.Fields[w.fieldID]
	if field == nil {
		return ""
	}
	return field.GetName()
}

// AstField returns the AST node for the field.
func (w *CompositeTypeFieldWalker) AstField() *v2ast.Field {
	astCT := w.db.WalkCompositeType(w.ctID).AstCompositeType()
	if astCT == nil || int(w.fieldID) >= len(astCT.Fields) {
		return nil
	}
	return astCT.Fields[w.fieldID]
}

// Type returns the type of the field.
func (w *CompositeTypeFieldWalker) Type() ScalarFieldType {
	key := CompositeTypeFieldKeyByID{
		CompositeTypeID: w.ctID,
		FieldID:         w.fieldID,
	}
	ctf, exists := w.db.types.CompositeTypeFields[key]
	if !exists {
		return ScalarFieldType{}
	}
	return ctf.Type
}

// DatabaseName returns the database name of the field.
func (w *CompositeTypeFieldWalker) DatabaseName() string {
	key := CompositeTypeFieldKeyByID{
		CompositeTypeID: w.ctID,
		FieldID:         w.fieldID,
	}
	ctf, exists := w.db.types.CompositeTypeFields[key]
	if !exists || ctf.MappedName == nil {
		return w.Name()
	}
	return w.db.interner.Get(*ctf.MappedName)
}

// DefaultValue returns the default value expression if @default is present.
// For composite type fields, we return the expression directly from AST.
func (w *CompositeTypeFieldWalker) DefaultValue() v2ast.Expression {
	key := CompositeTypeFieldKeyByID{
		CompositeTypeID: w.ctID,
		FieldID:         w.fieldID,
	}
	ctf, exists := w.db.types.CompositeTypeFields[key]
	if !exists || ctf.Default == nil {
		return nil
	}

	astCT := w.db.WalkCompositeType(w.ctID).AstCompositeType()
	if astCT == nil || int(w.fieldID) >= len(astCT.Fields) {
		return nil
	}

	astField := astCT.Fields[w.fieldID]
	if astField == nil {
		return nil
	}
	if int(ctf.Default.ArgumentIdx) >= len(astField.Attributes) {
		return nil
	}

	// Find the @default attribute
	for _, attr := range astField.Attributes {
		if attr != nil && attr.GetName() == "default" {
			if attr.Arguments != nil && int(ctf.Default.ArgumentIdx) < len(attr.Arguments.Arguments) {
				arg := attr.Arguments.Arguments[ctf.Default.ArgumentIdx]
				if arg != nil {
					return arg.Value
				}
			}
		}
	}

	return nil
}

// Span returns the span of the field from the AST.
func (w *CompositeTypeFieldWalker) Span() diagnostics.Span {
	astField := w.AstField()
	if astField == nil {
		return diagnostics.EmptySpan()
	}
	pos := astField.Pos
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(astField.GetName()), diagnostics.FileIDZero)
	return span
}
