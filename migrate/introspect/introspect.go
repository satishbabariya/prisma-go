// Package introspect provides database introspection capabilities.
package introspect

import (
	"context"
	"database/sql"
)

// Introspector reads database schema and converts it to Prisma schema
type Introspector interface {
	Introspect(ctx context.Context) (*DatabaseSchema, error)
}

// DatabaseSchema represents the introspected database schema
type DatabaseSchema struct {
	Tables           []Table
	Enums            []Enum
	Views            []View
	Sequences        []Sequence
	CheckConstraints []CheckConstraint
	Triggers         []Trigger
	StoredProcedures []StoredProcedure
}

// Table represents a database table
type Table struct {
	Name        string
	Schema      string
	Columns     []Column
	PrimaryKey  *PrimaryKey
	Indexes     []Index
	ForeignKeys []ForeignKey
}

// Column represents a table column
type Column struct {
	Name          string
	Type          string
	Nullable      bool
	DefaultValue  *string
	AutoIncrement bool
}

// PrimaryKey represents a primary key constraint
type PrimaryKey struct {
	Name    string
	Columns []string
}

// Index represents a database index
type Index struct {
	Name     string
	Columns  []string
	IsUnique bool
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name              string
	Columns           []string
	ReferencedTable   string
	ReferencedColumns []string
	OnDelete          string
	OnUpdate          string
}

// Enum represents a database enum type
type Enum struct {
	Name   string
	Values []string
}

// View represents a database view
type View struct {
	Name       string
	Definition string
}

// Sequence represents a database sequence
type Sequence struct {
	Name string
}

// CheckConstraint represents a check constraint
type CheckConstraint struct {
	Name       string
	TableName  string
	Definition string
}

// Trigger represents a database trigger
type Trigger struct {
	Name       string
	TableName  string
	Event      string // INSERT, UPDATE, DELETE, etc.
	Timing     string // BEFORE, AFTER, INSTEAD OF
	Definition string
}

// StoredProcedure represents a stored procedure
type StoredProcedure struct {
	Name       string
	Definition string
	Parameters []ProcedureParameter
}

// ProcedureParameter represents a stored procedure parameter
type ProcedureParameter struct {
	Name     string
	Type     string
	Mode     string // IN, OUT, INOUT
	Position int
}

// NewIntrospector creates a new introspector for the given database
func NewIntrospector(db *sql.DB, provider string) (Introspector, error) {
	switch provider {
	case "postgresql", "postgres":
		return &PostgresIntrospector{db: db}, nil
	case "mysql":
		return &MySQLIntrospector{db: db}, nil
	case "sqlite":
		return &SQLiteIntrospector{db: db}, nil
	case "sqlserver", "mssql":
		return &SQLServerIntrospector{db: db}, nil
	case "cockroachdb":
		return NewCockroachDBIntrospector(db), nil
	default:
		return nil, ErrUnsupportedProvider
	}
}

// MySQLIntrospector and SQLiteIntrospector are defined in their respective files
