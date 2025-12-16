// Package ir defines the intermediate representation for code generation.
package ir

// IR represents the intermediate representation of a Prisma schema.
type IR struct {
	Models    []Model
	Enums     []Enum
	Config    Config
	Relations []Relation
}

// Model represents a Prisma model in IR.
type Model struct {
	Name       string
	TableName  string // From @@map or name
	Fields     []Field
	PrimaryKey *PrimaryKey
	Indexes    []Index
	IsView     bool
}

// Field represents a model field.
type Field struct {
	Name         string
	GoName       string // PascalCase Go name
	Type         FieldType
	IsOptional   bool
	IsArray      bool
	DefaultValue interface{}
	IsUnique     bool
	IsID         bool
	Relation     *RelationInfo
	Tags         map[string]string // For struct tags
}

// FieldType represents a field's type information.
type FieldType struct {
	PrismaType string // Original Prisma type
	GoType     string // Mapped Go type
	IsScalar   bool
	IsModel    bool
	IsEnum     bool
}

// RelationInfo contains relation metadata.
type RelationInfo struct {
	Name         string
	RelatedModel string
	RelationType RelationType
	ForeignKey   string
	References   string
	OnDelete     string
	OnUpdate     string
}

// RelationType represents the type of relation.
type RelationType string

const (
	OneToOne   RelationType = "OneToOne"
	OneToMany  RelationType = "OneToMany"
	ManyToMany RelationType = "ManyToMany"
)

// PrimaryKey represents a primary key.
type PrimaryKey struct {
	Fields []string
	Name   string
}

// Index represents an index.
type Index struct {
	Name   string
	Fields []string
	Unique bool
}

// Enum represents a Prisma enum.
type Enum struct {
	Name   string
	Values []EnumValue
}

// EnumValue represents an enum value.
type EnumValue struct {
	Name   string
	GoName string
}

// Config holds generator configuration.
type Config struct {
	PackageName string
	OutputPath  string
}

// Relation represents a relation between models.
type Relation struct {
	Name      string
	FromModel string
	ToModel   string
	Type      RelationType
	FromField string
	ToField   string
}
