// Package parserdatabase provides DefaultValueWalker for accessing default value data.
package database

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// DefaultValueWalker provides access to a @default() attribute on a field.
type DefaultValueWalker struct {
	db          *ParserDatabase
	fieldID     ScalarFieldId
	defaultAttr *DefaultAttribute
}

// Value returns the default value expression.
func (w *DefaultValueWalker) Value() v2ast.Expression {
	sf := w.db.types.ScalarFields[w.fieldID]
	astModel := w.db.getModelFromID(sf.ModelID)
	if astModel == nil || int(sf.FieldID) >= len(astModel.Fields) {
		return nil
	}

	astField := astModel.Fields[sf.FieldID]
	if w.defaultAttr == nil || astField == nil {
		return nil
	}

	// Find the @default attribute
	for _, attr := range astField.Attributes {
		if attr != nil && attr.GetName() == "default" {
			if attr.Arguments != nil && int(w.defaultAttr.ArgumentIdx) < len(attr.Arguments.Arguments) {
				return attr.Arguments.Arguments[w.defaultAttr.ArgumentIdx].Value
			}
		}
	}

	return nil
}

// MappedName returns the mapped name if @default(map: "...") is present.
func (w *DefaultValueWalker) MappedName() *string {
	if w.defaultAttr == nil || w.defaultAttr.MappedName == nil {
		return nil
	}
	name := w.db.interner.Get(*w.defaultAttr.MappedName)
	return &name
}

// IsAutoIncrement returns whether this is an autoincrement default.
func (w *DefaultValueWalker) IsAutoIncrement() bool {
	value := w.Value()
	if value == nil {
		return false
	}

	// Check if it's a function call to autoincrement()
	if funcCall, ok := value.(*v2ast.FunctionCall); ok {
		return funcCall.Name == "autoincrement"
	}

	return false
}
