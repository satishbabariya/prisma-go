// Package parserdatabase provides type aliases for AST identifiers with file IDs.
package database

import "github.com/satishbabariya/prisma-go/psl/diagnostics"

// InFile represents an AST identifier with the accompanying file ID.
type InFile struct {
	FileID diagnostics.FileID
	ID     uint32
}

// ModelId represents a model identifier with file ID.
type ModelId = InFile

// EnumId represents an enum identifier with file ID.
type EnumId = InFile

// CompositeTypeId represents a composite type identifier with file ID.
type CompositeTypeId = InFile

// TopId represents a top-level identifier with file ID.
type TopId = InFile

// AttributeContainer represents an attribute container with file ID.
// The ID field encodes the container type and index:
// - For models: ID is the model index
// - For fields: ID encodes (model_index << 16) | field_index
// - For enums: ID is the enum index
// - For enum values: ID encodes (enum_index << 16) | value_index
// - For composite types: ID is the composite type index
// - For composite type fields: ID encodes (composite_type_index << 16) | field_index
type AttributeContainer = InFile

// AttributeId represents an attribute identifier.
// It contains the container and the index of the attribute within that container.
type AttributeId struct {
	FileID    diagnostics.FileID
	Container AttributeContainer
	Index     uint32 // Index within the container's attributes slice
}
