// Package parserdatabase provides type resolution functionality.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// ScalarFieldId represents an identifier for a scalar field.
type ScalarFieldId uint32

// RelationFieldId represents an identifier for a relation field.
type RelationFieldId uint32

// ScalarType represents a built-in Prisma scalar type.
type ScalarType string

const (
	ScalarTypeString   ScalarType = "String"
	ScalarTypeInt      ScalarType = "Int"
	ScalarTypeFloat    ScalarType = "Float"
	ScalarTypeBoolean  ScalarType = "Boolean"
	ScalarTypeDateTime ScalarType = "DateTime"
	ScalarTypeJson     ScalarType = "Json"
	ScalarTypeBytes    ScalarType = "Bytes"
	ScalarTypeBigInt   ScalarType = "BigInt"
	ScalarTypeDecimal  ScalarType = "Decimal"
)

// ScalarFieldType represents the type of a scalar field.
type ScalarFieldType struct {
	// One of these will be set:
	CompositeTypeID *CompositeTypeId
	EnumID          *EnumId
	ExtensionID     *ExtensionTypeId
	BuiltInScalar   *ScalarType
	Unsupported     *UnsupportedType
}

// UnsupportedType represents an unsupported type.
type UnsupportedType struct {
	Name StringId
}

// ScalarField represents a scalar field in a model.
type ScalarField struct {
	ModelID     ModelId
	FieldID     uint32 // ast.FieldId equivalent
	Type        ScalarFieldType
	IsIgnored   bool
	IsUpdatedAt bool
	Default     *DefaultAttribute
	MappedName  *StringId // @map
	// Native type: (attribute scope, native type name, arguments, span)
	// For example: @db.Text would translate to ("db", "Text", [], span)
	NativeType *NativeTypeInfo
}

// NativeTypeInfo represents native type information.
type NativeTypeInfo struct {
	Scope     StringId
	TypeName  StringId
	Arguments []string
	Span      diagnostics.Span
}

// RelationField represents a relation field in a model.
type RelationField struct {
	ModelID         ModelId
	FieldID         uint32 // ast.FieldId equivalent
	ReferencedModel ModelId
	OnDelete        *ReferentialActionInfo
	OnUpdate        *ReferentialActionInfo
	// The fields explicitly present in the AST
	Fields *[]ScalarFieldId
	// The references fields explicitly present in the AST
	References *[]ScalarFieldId
	// The name explicitly present in the AST
	Name      *StringId
	IsIgnored bool
	// The foreign key name explicitly present in the AST through the @map attribute
	MappedName        *StringId
	RelationAttribute *uint32 // ast.AttributeId equivalent
}

// ReferentialActionInfo represents referential action information.
type ReferentialActionInfo struct {
	Action ReferentialAction
	Span   diagnostics.Span
}

// ReferentialAction represents a referential action.
type ReferentialAction string

const (
	ReferentialActionCascade    ReferentialAction = "Cascade"
	ReferentialActionRestrict   ReferentialAction = "Restrict"
	ReferentialActionNoAction   ReferentialAction = "NoAction"
	ReferentialActionSetNull    ReferentialAction = "SetNull"
	ReferentialActionSetDefault ReferentialAction = "SetDefault"
)

// DefaultAttribute represents a @default attribute.
type DefaultAttribute struct {
	MappedName       *StringId
	ArgumentIdx      int
	DefaultAttribute uint32 // AttributeId equivalent
}

// CompositeTypeField represents a field in a composite type.
type CompositeTypeField struct {
	Type       ScalarFieldType
	MappedName *StringId
	Default    *DefaultAttribute
	NativeType *NativeTypeInfo
}

// CompositeTypeFieldKeyByID represents a key for composite type field lookup in Types.
// This uses FieldID (index) instead of NameID.
type CompositeTypeFieldKeyByID struct {
	CompositeTypeID CompositeTypeId
	FieldID         uint32
}

// RefinedFieldVariant represents the variant of a refined field.
type RefinedFieldVariant int

const (
	RefinedFieldVariantRelation RefinedFieldVariant = iota
	RefinedFieldVariantScalar
	RefinedFieldVariantUnknown
)

// Types holds resolved type information.
type Types struct {
	// Composite type fields: (CompositeTypeId, FieldId) -> CompositeTypeField
	CompositeTypeFields map[CompositeTypeFieldKeyByID]CompositeTypeField
	// Scalar fields sorted by (model_id, field_id) for efficient lookup
	ScalarFields []ScalarField
	// Relation fields sorted by (model_id, field_id) for efficient lookup
	RelationFields []RelationField
	// Enum attributes: EnumId -> EnumAttributes
	EnumAttributes map[EnumId]EnumAttributes
	// Model attributes: ModelId -> ModelAttributes
	ModelAttributes map[ModelId]ModelAttributes
	// Sorted array of scalar fields that have an @default() attribute with a function
	// that is not part of the base Prisma ones
	UnknownFunctionDefaults []ScalarFieldId
}

// NewTypes creates a new empty Types structure.
func NewTypes() Types {
	return Types{
		CompositeTypeFields:     make(map[CompositeTypeFieldKeyByID]CompositeTypeField),
		ScalarFields:            make([]ScalarField, 0),
		RelationFields:          make([]RelationField, 0),
		EnumAttributes:          make(map[EnumId]EnumAttributes),
		ModelAttributes:         make(map[ModelId]ModelAttributes),
		UnknownFunctionDefaults: make([]ScalarFieldId, 0),
	}
}

// EnumAttributes holds attributes for an enum.
type EnumAttributes struct {
	MappedName *StringId
}

// ModelAttributes holds attributes for a model.
type ModelAttributes struct {
	// @(@)id
	PrimaryKey *IdAttribute
	// @@ignore
	IsIgnored bool
	// @@index and @(@)unique explicitly written to the schema AST
	AstIndexes []IndexAttributeEntry
	// @@map
	MappedName *StringId
	// @@schema("public")
	Schema *SchemaInfo
	// @(@)shardKey
	ShardKey *ShardKeyAttribute
}

// IndexAttributeEntry represents an index attribute entry.
type IndexAttributeEntry struct {
	AttributeID uint32 // AttributeId equivalent
	Index       IndexAttribute
}

// IdAttribute represents an @id or @@id attribute.
type IdAttribute struct {
	Fields          []FieldWithArgs
	SourceField     *uint32 // FieldId equivalent
	SourceAttribute uint32  // AttributeId equivalent
	Name            *StringId
	MappedName      *StringId
	Clustered       *bool
}

// ShardKeyAttribute represents a @shardKey or @@shardKey attribute.
type ShardKeyAttribute struct {
	Fields          []ScalarFieldId
	SourceField     *uint32 // FieldId equivalent
	SourceAttribute uint32  // AttributeId equivalent
}

// SchemaInfo represents schema information.
type SchemaInfo struct {
	Name StringId
	Span diagnostics.Span
}

// IndexAttribute represents an index attribute.
type IndexAttribute struct {
	Type        IndexType
	Fields      []FieldWithArgs
	SourceField *ScalarFieldId
	Name        *StringId
	MappedName  *StringId
	Algorithm   *IndexAlgorithm
	Clustered   *bool
}

// IndexType represents the type of an index.
type IndexType int

const (
	IndexTypeNormal IndexType = iota
	IndexTypeUnique
	IndexTypeFulltext
)

// IndexAlgorithm represents the algorithm used for an index.
type IndexAlgorithm int

const (
	IndexAlgorithmBTree IndexAlgorithm = iota
	IndexAlgorithmHash
	IndexAlgorithmGist
	IndexAlgorithmGin
	IndexAlgorithmSpGist
	IndexAlgorithmBrin
)

// FieldWithArgs represents a field with arguments (for indexes, unique constraints, etc.).
type FieldWithArgs struct {
	Field         ScalarFieldId
	Path          []IndexFieldPathSegment // For composite type fields
	SortOrder     *SortOrder
	OperatorClass *OperatorClass
	Length        *int
}

// IndexFieldPathSegment represents a segment in an index field path.
type IndexFieldPathSegment struct {
	FieldID uint32 // FieldId in the composite type
}

// SortOrder represents the sort order for an index field.
type SortOrder int

const (
	SortOrderAsc SortOrder = iota
	SortOrderDesc
)

// OperatorClass represents an operator class for an index field.
type OperatorClass struct {
	Name StringId
}

// FindModelScalarField finds a scalar field in a model by field ID.
func (t *Types) FindModelScalarField(modelID ModelId, fieldID uint32) *ScalarFieldId {
	// Binary search since ScalarFields is sorted by (model_id, field_id)
	left := 0
	right := len(t.ScalarFields)

	for left < right {
		mid := (left + right) / 2
		sf := t.ScalarFields[mid]
		if sf.ModelID.FileID < modelID.FileID ||
			(sf.ModelID.FileID == modelID.FileID && sf.ModelID.ID < modelID.ID) ||
			(sf.ModelID == modelID && sf.FieldID < fieldID) {
			left = mid + 1
		} else if sf.ModelID == modelID && sf.FieldID == fieldID {
			id := ScalarFieldId(mid)
			return &id
		} else {
			right = mid
		}
	}
	return nil
}

// PushScalarField adds a scalar field to the types.
// The field must be added in sorted order by (model_id, field_id).
func (t *Types) PushScalarField(field ScalarField) ScalarFieldId {
	id := ScalarFieldId(len(t.ScalarFields))
	t.ScalarFields = append(t.ScalarFields, field)
	return id
}

// PushRelationField adds a relation field to the types.
// The field must be added in sorted order by (model_id, field_id).
func (t *Types) PushRelationField(field RelationField) RelationFieldId {
	id := RelationFieldId(len(t.RelationFields))
	t.RelationFields = append(t.RelationFields, field)
	return id
}

// RefineField determines if a field is a relation field, scalar field, or unknown.
func (t *Types) RefineField(modelID ModelId, fieldID uint32) RefinedFieldVariant {
	// Check relation fields first (binary search)
	left := 0
	right := len(t.RelationFields)
	for left < right {
		mid := (left + right) / 2
		rf := t.RelationFields[mid]
		if rf.ModelID.FileID < modelID.FileID ||
			(rf.ModelID.FileID == modelID.FileID && rf.ModelID.ID < modelID.ID) ||
			(rf.ModelID == modelID && rf.FieldID < fieldID) {
			left = mid + 1
		} else if rf.ModelID == modelID && rf.FieldID == fieldID {
			return RefinedFieldVariantRelation
		} else {
			right = mid
		}
	}

	// Check scalar fields
	if t.FindModelScalarField(modelID, fieldID) != nil {
		return RefinedFieldVariantScalar
	}

	return RefinedFieldVariantUnknown
}

// RangeModelScalarFields returns all scalar fields for a given model.
// Returns a slice of (ScalarFieldId, ScalarField) pairs.
func (t *Types) RangeModelScalarFields(modelID ModelId) []ScalarFieldEntry {
	var result []ScalarFieldEntry
	start := t.findScalarFieldStart(modelID)

	for i := start; i < len(t.ScalarFields); i++ {
		sf := t.ScalarFields[i]
		if sf.ModelID != modelID {
			break
		}
		result = append(result, ScalarFieldEntry{
			ID:    ScalarFieldId(i),
			Field: &sf,
		})
	}
	return result
}

// ScalarFieldEntry represents a scalar field entry with its ID.
type ScalarFieldEntry struct {
	ID    ScalarFieldId
	Field *ScalarField
}

// RangeModelScalarFieldIDs returns all scalar field IDs for a given model.
func (t *Types) RangeModelScalarFieldIDs(modelID ModelId) []ScalarFieldId {
	var result []ScalarFieldId
	start := t.findScalarFieldStart(modelID)
	end := t.findScalarFieldEnd(modelID)

	for i := start; i < end; i++ {
		result = append(result, ScalarFieldId(i))
	}
	return result
}

// RangeModelRelationFields returns all relation fields for a given model.
// Returns a slice of (RelationFieldId, RelationField) pairs.
func (t *Types) RangeModelRelationFields(modelID ModelId) []RelationFieldEntry {
	var result []RelationFieldEntry
	start := t.findRelationFieldStart(modelID)

	for i := start; i < len(t.RelationFields); i++ {
		rf := t.RelationFields[i]
		if rf.ModelID != modelID {
			break
		}
		result = append(result, RelationFieldEntry{
			ID:    RelationFieldId(i),
			Field: &rf,
		})
	}
	return result
}

// RelationFieldEntry represents a relation field entry with its ID.
type RelationFieldEntry struct {
	ID    RelationFieldId
	Field *RelationField
}

// IterRelationFieldIDs returns all relation field IDs.
func (t *Types) IterRelationFieldIDs() []RelationFieldId {
	result := make([]RelationFieldId, len(t.RelationFields))
	for i := range t.RelationFields {
		result[i] = RelationFieldId(i)
	}
	return result
}

// IterRelationFields returns all relation fields with their IDs.
func (t *Types) IterRelationFields() []RelationFieldEntry {
	result := make([]RelationFieldEntry, len(t.RelationFields))
	for i := range t.RelationFields {
		result[i] = RelationFieldEntry{
			ID:    RelationFieldId(i),
			Field: &t.RelationFields[i],
		}
	}
	return result
}

// findScalarFieldStart finds the start index of scalar fields for a model using binary search.
func (t *Types) findScalarFieldStart(modelID ModelId) int {
	left := 0
	right := len(t.ScalarFields)

	for left < right {
		mid := (left + right) / 2
		sf := t.ScalarFields[mid]
		if sf.ModelID.FileID < modelID.FileID ||
			(sf.ModelID.FileID == modelID.FileID && sf.ModelID.ID < modelID.ID) {
			left = mid + 1
		} else {
			right = mid
		}
	}
	return left
}

// findScalarFieldEnd finds the end index of scalar fields for a model using binary search.
func (t *Types) findScalarFieldEnd(modelID ModelId) int {
	left := 0
	right := len(t.ScalarFields)

	for left < right {
		mid := (left + right) / 2
		sf := t.ScalarFields[mid]
		if sf.ModelID.FileID < modelID.FileID ||
			(sf.ModelID.FileID == modelID.FileID && sf.ModelID.ID <= modelID.ID) {
			left = mid + 1
		} else {
			right = mid
		}
	}
	return left
}

// findRelationFieldStart finds the start index of relation fields for a model using binary search.
func (t *Types) findRelationFieldStart(modelID ModelId) int {
	left := 0
	right := len(t.RelationFields)

	for left < right {
		mid := (left + right) / 2
		rf := t.RelationFields[mid]
		if rf.ModelID.FileID < modelID.FileID ||
			(rf.ModelID.FileID == modelID.FileID && rf.ModelID.ID < modelID.ID) {
			left = mid + 1
		} else {
			right = mid
		}
	}
	return left
}
