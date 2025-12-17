// Package domain contains the core business entities and interfaces for the Migration domain.
package domain

import (
	"context"
	"time"
)

// Migration represents a migration aggregate root.
type Migration struct {
	ID          string
	Name        string
	CreatedAt   time.Time
	AppliedAt   *time.Time
	Changes     []Change
	SQL         []string
	RollbackSQL []string // SQL statements to reverse the migration
	Checksum    string
	Status      MigrationStatus
}

// MigrationStatus represents the status of a migration.
type MigrationStatus string

const (
	// Pending indicates the migration has not been applied.
	Pending MigrationStatus = "Pending"
	// Applied indicates the migration has been successfully applied.
	Applied MigrationStatus = "Applied"
	// Failed indicates the migration failed to apply.
	Failed MigrationStatus = "Failed"
	// RolledBack indicates the migration has been rolled back.
	RolledBack MigrationStatus = "RolledBack"
)

// Change represents a database schema change.
type Change interface {
	// Type returns the type of change.
	Type() ChangeType

	// Description returns a human-readable description.
	Description() string

	// ToSQL converts the change to SQL statements.
	ToSQL(dialect SQLDialect) ([]string, error)

	// IsDestructive returns true if the change may cause data loss.
	IsDestructive() bool
}

// ChangeType represents the type of schema change.
type ChangeType string

const (
	// CreateTable creates a new table.
	CreateTable ChangeType = "CreateTable"
	// DropTable drops a table.
	DropTable ChangeType = "DropTable"
	// AlterTable alters a table.
	AlterTable ChangeType = "AlterTable"
	// AddColumn adds a column.
	AddColumn ChangeType = "AddColumn"
	// DropColumn drops a column.
	DropColumn ChangeType = "DropColumn"
	// AlterColumn alters a column.
	AlterColumn ChangeType = "AlterColumn"
	// CreateIndex creates an index.
	CreateIndex ChangeType = "CreateIndex"
	// DropIndex drops an index.
	DropIndex ChangeType = "DropIndex"
	// AddConstraint adds a constraint.
	AddConstraint ChangeType = "AddConstraint"
	// DropConstraint drops a constraint.
	DropConstraint ChangeType = "DropConstraint"
)

// SQLDialect represents a SQL dialect.
type SQLDialect string

const (
	// PostgreSQL dialect.
	PostgreSQL SQLDialect = "postgres"
	// MySQL dialect.
	MySQL SQLDialect = "mysql"
	// SQLite dialect.
	SQLite SQLDialect = "sqlite"
)

// DatabaseState represents the current state of a database.
type DatabaseState struct {
	Tables []Table
}

// Table represents a database table.
type Table struct {
	Name        string
	Columns     []Column
	Indexes     []Index
	Constraints []Constraint
}

// Column represents a database column.
type Column struct {
	Name         string
	Type         string
	IsNullable   bool
	DefaultValue interface{}
	IsPrimaryKey bool
	IsUnique     bool
}

// Index represents a database index.
type Index struct {
	Name     string
	Columns  []string
	IsUnique bool
}

// Constraint represents a database constraint.
type Constraint struct {
	Name              string
	Type              ConstraintType
	Columns           []string
	ReferencedTable   string
	ReferencedColumns []string
	OnDelete          ReferentialAction
	OnUpdate          ReferentialAction
}

// ConstraintType represents the type of constraint.
type ConstraintType string

const (
	// PrimaryKey constraint.
	PrimaryKey ConstraintType = "PrimaryKey"
	// ForeignKey constraint.
	ForeignKey ConstraintType = "ForeignKey"
	// UniqueKey constraint.
	UniqueKey ConstraintType = "UniqueKey"
	// CheckKey constraint.
	CheckKey ConstraintType = "CheckKey"
)

// ReferentialAction represents a referential action.
type ReferentialAction string

const (
	// Cascade deletes/updates related records.
	Cascade ReferentialAction = "Cascade"
	// Restrict prevents deletion/update.
	Restrict ReferentialAction = "Restrict"
	// NoAction similar to Restrict.
	NoAction ReferentialAction = "NoAction"
	// SetNull sets to NULL.
	SetNull ReferentialAction = "SetNull"
	// SetDefault sets to default value.
	SetDefault ReferentialAction = "SetDefault"
)

// MigrationPlan represents a plan for applying changes.
type MigrationPlan struct {
	Changes []Change
	SQL     []string
}

// Introspector defines the interface for database introspection.
type Introspector interface {
	// IntrospectDatabase introspects the entire database.
	IntrospectDatabase(ctx context.Context) (*DatabaseState, error)

	// IntrospectTable introspects a specific table.
	IntrospectTable(ctx context.Context, tableName string) (*Table, error)

	// ListTables lists all tables in the database.
	ListTables(ctx context.Context) ([]string, error)
}

// Differ defines the interface for schema comparison.
type Differ interface {
	// Compare compares two database states and returns changes.
	Compare(ctx context.Context, from, to *DatabaseState) ([]Change, error)

	// CompareTables compares two tables.
	CompareTables(ctx context.Context, from, to *Table) ([]Change, error)

	// CompareColumns compares two columns.
	CompareColumns(ctx context.Context, from, to *Column) ([]Change, error)
}

// Planner defines the interface for migration planning.
type Planner interface {
	// CreatePlan creates a migration plan from changes.
	CreatePlan(ctx context.Context, changes []Change) (*MigrationPlan, error)

	// OptimizePlan optimizes a migration plan.
	OptimizePlan(ctx context.Context, plan *MigrationPlan) (*MigrationPlan, error)

	// ValidatePlan validates a migration plan.
	ValidatePlan(ctx context.Context, plan *MigrationPlan) error
}

// Executor defines the interface for migration execution.
type Executor interface {
	// Execute executes a migration.
	Execute(ctx context.Context, migration *Migration) error

	// Rollback rolls back a migration.
	Rollback(ctx context.Context, migration *Migration) error

	// ExecuteSQL executes raw SQL statements.
	ExecuteSQL(ctx context.Context, sql []string) error
}
