// Package domain contains the core business entities and interfaces for the Schema domain.
package domain

import "context"

// Schema represents the Prisma schema aggregate root.
type Schema struct {
	Datasources []Datasource
	Generators  []Generator
	Models      []Model
	Enums       []Enum
}

// Datasource represents a database connection configuration.
type Datasource struct {
	Name     string
	Provider string
	URL      string
}

// Generator represents a code generator configuration.
type Generator struct {
	Name     string
	Provider string
	Output   string
}

// Model represents a Prisma model entity.
type Model struct {
	Name       string
	Fields     []Field
	Indexes    []Index
	Attributes []Attribute
	Comments   []string
}

// Field represents a model field value object.
type Field struct {
	Name         string
	Type         FieldType
	IsRequired   bool
	IsList       bool
	IsUnique     bool
	DefaultValue interface{}
	Attributes   []Attribute
}

// FieldType represents field type information.
type FieldType struct {
	Name      string
	IsBuiltin bool
	IsModel   bool
	IsEnum    bool
}

// Enum represents an enum entity.
type Enum struct {
	Name   string
	Values []string
}

// Index represents an index value object.
type Index struct {
	Name   string
	Fields []string
	Unique bool
	Type   IndexType
}

// IndexType represents the type of index.
type IndexType string

const (
	// BTreeIndex is a B-Tree index.
	BTreeIndex IndexType = "BTree"
	// HashIndex is a Hash index.
	HashIndex IndexType = "Hash"
)

// Attribute represents a field or model attribute.
type Attribute struct {
	Name      string
	Arguments []interface{}
}

// Relation represents a relation value object.
type Relation struct {
	Name         string
	FromModel    string
	ToModel      string
	FromFields   []string
	ToFields     []string
	RelationType RelationType
	OnDelete     ReferentialAction
	OnUpdate     ReferentialAction
}

// RelationType represents the type of relation.
type RelationType string

const (
	// OneToOne represents a one-to-one relation.
	OneToOne RelationType = "OneToOne"
	// OneToMany represents a one-to-many relation.
	OneToMany RelationType = "OneToMany"
	// ManyToOne represents a many-to-one relation.
	ManyToOne RelationType = "ManyToOne"
	// ManyToMany represents a many-to-many relation.
	ManyToMany RelationType = "ManyToMany"
)

// ReferentialAction represents a referential action for foreign keys.
type ReferentialAction string

const (
	// Cascade deletes/updates related records.
	Cascade ReferentialAction = "Cascade"
	// Restrict prevents deletion/update if related records exist.
	Restrict ReferentialAction = "Restrict"
	// NoAction similar to Restrict.
	NoAction ReferentialAction = "NoAction"
	// SetNull sets foreign key to NULL.
	SetNull ReferentialAction = "SetNull"
	// SetDefault sets foreign key to default value.
	SetDefault ReferentialAction = "SetDefault"
)

// SchemaParser defines the interface for parsing schemas.
type SchemaParser interface {
	// Parse parses schema content from a string.
	Parse(ctx context.Context, content string) (*Schema, error)

	// ParseFile parses schema from a file.
	ParseFile(ctx context.Context, path string) (*Schema, error)
}

// SchemaValidator defines the interface for validating schemas.
type SchemaValidator interface {
	// Validate validates the entire schema.
	Validate(ctx context.Context, schema *Schema) error

	// ValidateModel validates a single model.
	ValidateModel(ctx context.Context, model *Model) error

	// ValidateField validates a single field.
	ValidateField(ctx context.Context, field *Field) error

	// ValidateRelation validates a relation.
	ValidateRelation(ctx context.Context, relation *Relation) error
}

// SchemaFormatter defines the interface for formatting schemas.
type SchemaFormatter interface {
	// Format formats a schema to a string.
	Format(ctx context.Context, schema *Schema) (string, error)

	// FormatFile formats a schema file in place.
	FormatFile(ctx context.Context, path string) error
}
