// Package domain contains domain models for database introspection.
package domain

import "time"

// IntrospectedDatabase represents a complete database schema.
type IntrospectedDatabase struct {
	Tables    []IntrospectedTable
	Enums     []IntrospectedEnum
	Sequences []IntrospectedSequence
	Views     []IntrospectedView
}

// IntrospectedTable represents a database table.
type IntrospectedTable struct {
	Name        string
	Schema      string // Schema name (e.g., "public" in PostgreSQL)
	Columns     []IntrospectedColumn
	Indexes     []IntrospectedIndex
	ForeignKeys []IntrospectedForeignKey
	PrimaryKey  *IntrospectedPrimaryKey
	Comment     *string
}

// IntrospectedColumn represents a table column.
type IntrospectedColumn struct {
	Name            string
	Type            string // Database-specific type (e.g., "VARCHAR", "INTEGER")
	IsNullable      bool
	DefaultValue    *string
	IsAutoIncrement bool
	IsPrimaryKey    bool
	IsUnique        bool
	Comment         *string
	OrdinalPosition int
}

// IntrospectedIndex represents a database index.
type IntrospectedIndex struct {
	Name     string
	Columns  []string
	IsUnique bool
	Type     string // e.g., "btree", "hash"
}

// IntrospectedForeignKey represents a foreign key constraint.
type IntrospectedForeignKey struct {
	Name              string
	ColumnNames       []string
	ReferencedTable   string
	ReferencedColumns []string
	OnDelete          ReferentialAction
	OnUpdate          ReferentialAction
}

// IntrospectedPrimaryKey represents a primary key constraint.
type IntrospectedPrimaryKey struct {
	Name    string
	Columns []string
}

// IntrospectedEnum represents an enum type.
type IntrospectedEnum struct {
	Name   string
	Values []string
}

// IntrospectedSequence represents a database sequence.
type IntrospectedSequence struct {
	Name      string
	Start     int64
	Increment int64
}

// IntrospectedView represents a database view.
type IntrospectedView struct {
	Name       string
	Definition string
	Columns    []IntrospectedColumn
}

// ReferentialAction represents a foreign key action.
type ReferentialAction string

const (
	// NoAction does nothing on delete/update.
	NoAction ReferentialAction = "NO ACTION"
	// Cascade deletes/updates related records.
	Cascade ReferentialAction = "CASCADE"
	// Restrict prevents deletion/update if related records exist.
	Restrict ReferentialAction = "RESTRICT"
	// SetNull sets the foreign key to NULL.
	SetNull ReferentialAction = "SET NULL"
	// SetDefault sets the foreign key to its default value.
	SetDefault ReferentialAction = "SET DEFAULT"
)

// IntrospectionContext holds metadata about the introspection process.
type IntrospectionContext struct {
	DatabaseName   string
	DatabaseType   string // "postgresql", "mysql", "sqlite"
	Version        string
	IntrospectedAt time.Time
}
