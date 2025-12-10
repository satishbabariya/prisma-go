// Package parserdatabase provides validation context functionality.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// Context is a validation context that contains the database itself,
// as well as context that is discarded after validation.
//
// The Context also acts as a state machine for attribute validation.
// The goal is to avoid manual work validating things that are valid
// for every attribute set, and every argument set inside an attribute:
// multiple unnamed arguments are not valid, attributes we do not use
// in parser-database are not valid, multiple arguments with the same
// name are not valid, etc.
type Context struct {
	asts           *Files
	interner       *StringInterner
	names          *Names
	types          *Types
	relations      *Relations
	diagnostics    *diagnostics.Diagnostics
	extensionTypes ExtensionTypes

	// Attribute validation state machine
	attributes AttributesValidationState

	// @map'ed names indexes. These are not in the db because they are only used for validation.
	mappedModelScalarFieldNames map[ModelFieldKeyByID]uint32         // (ModelId, StringId) -> FieldId
	mappedCompositeTypeNames    map[CompositeTypeFieldKeyByID]uint32 // (CompositeTypeId, StringId) -> FieldId
	mappedEnumNames             map[StringId]EnumId
	mappedEnumValueNames        map[EnumValueKey]uint32 // (EnumId, StringId) -> EnumValueId
}

// ModelFieldKeyByID represents a key for model field lookup by ID.
type ModelFieldKeyByID struct {
	ModelID ModelId
	NameID  StringId
}

// EnumValueKey represents a key for enum value lookup.
type EnumValueKey struct {
	EnumID EnumId
	NameID StringId
}

// AttributesValidationState represents the state of attribute validation.
type AttributesValidationState struct {
	// The attribute container currently being validated
	attributes *AttributeContainer
	// The attribute currently being validated
	attribute *AttributeId
	// Unused attributes (by AttributeId)
	unusedAttributes map[AttributeId]bool
	// Arguments by name (nil key = unnamed argument)
	args map[*StringId]int
}

// NewContext creates a new validation context.
func NewContext(
	asts *Files,
	interner *StringInterner,
	names *Names,
	types *Types,
	relations *Relations,
	diags *diagnostics.Diagnostics,
	extensionTypes ExtensionTypes,
) *Context {
	return &Context{
		asts:           asts,
		interner:       interner,
		names:          names,
		types:          types,
		relations:      relations,
		diagnostics:    diags,
		extensionTypes: extensionTypes,
		attributes: AttributesValidationState{
			unusedAttributes: make(map[AttributeId]bool),
			args:             make(map[*StringId]int),
		},
		mappedModelScalarFieldNames: make(map[ModelFieldKeyByID]uint32),
		mappedCompositeTypeNames:    make(map[CompositeTypeFieldKeyByID]uint32),
		mappedEnumNames:             make(map[StringId]EnumId),
		mappedEnumValueNames:        make(map[EnumValueKey]uint32),
	}
}

// PushError adds an error to the diagnostics.
func (ctx *Context) PushError(err diagnostics.DatamodelError) {
	ctx.diagnostics.PushError(err)
}

// ExtensionTypes returns the extension types.
func (ctx *Context) ExtensionTypes() ExtensionTypes {
	return ctx.extensionTypes
}

// FindModelField finds a specific field in a specific model by name.
func (ctx *Context) FindModelField(modelID ModelId, fieldName string) *uint32 {
	nameID, ok := ctx.interner.Lookup(fieldName)
	if !ok {
		return nil
	}
	key := ModelFieldKey{
		ModelID: modelID,
		NameID:  nameID,
	}
	fieldID, ok := ctx.names.ModelFields[key]
	if !ok {
		return nil
	}
	return &fieldID
}

// FindCompositeTypeField finds a specific field in a specific composite type by name.
func (ctx *Context) FindCompositeTypeField(compositeTypeID CompositeTypeId, fieldName string) *uint32 {
	nameID, ok := ctx.interner.Lookup(fieldName)
	if !ok {
		return nil
	}
	key := CompositeTypeFieldKey{
		CompositeTypeID: compositeTypeID,
		NameID:          nameID,
	}
	fieldID, ok := ctx.names.CompositeTypeFields[key]
	if !ok {
		return nil
	}
	return &fieldID
}

// IterTops returns an iterator over all top-level items.
func (ctx *Context) IterTops() []TopEntry {
	var result []TopEntry
	for _, file := range ctx.asts.files {
		for i, top := range file.AST.Tops {
			topID := TopId{
				FileID: file.FileID,
				ID:     uint32(i),
			}
			result = append(result, TopEntry{
				TopID: topID,
				Top:   top,
			})
		}
	}
	return result
}

// TopEntry represents a top-level entry.
type TopEntry struct {
	TopID TopId
	Top   ast.Top
}
