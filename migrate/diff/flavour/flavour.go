// Package flavour provides provider-specific differ logic
package flavour

import (
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// DifferFlavour provides provider-specific logic for schema comparison
type DifferFlavour interface {
	// IndexesMatch checks if two indexes match by structure (ignoring name)
	IndexesMatch(prev, next *introspect.Index) bool

	// ForeignKeysMatch checks if two foreign keys match by structure (ignoring name)
	ForeignKeysMatch(prev, next *introspect.ForeignKey) bool

	// ColumnTypeChange detects if a column type has changed
	ColumnTypeChange(prev, next *introspect.Column) *ColumnTypeChange

	// ShouldRedefineTable determines if a table needs to be recreated
	ShouldRedefineTable(tableName string, changes []string) bool

	// CanRenameIndex returns whether index renames are supported
	CanRenameIndex() bool

	// CanRenameForeignKey returns whether foreign key renames are supported
	CanRenameForeignKey() bool

	// IndexShouldBeRenamed determines if an index should be renamed
	// Returns true if indexes match by structure but have different names
	IndexShouldBeRenamed(prev, next *introspect.Index) bool

	// LowerCasesTableNames returns true if table names should be lowercased for comparison
	LowerCasesTableNames() bool

	// TableShouldBeIgnored returns true if a table should be ignored during diffing
	TableShouldBeIgnored(tableName string) bool
}

// ColumnTypeChange represents a column type change
type ColumnTypeChange struct {
	FromType string
	ToType   string
	IsSafe   bool
}

// NewColumnTypeChange creates a new ColumnTypeChange
func NewColumnTypeChange(fromType, toType string, isSafe bool) *ColumnTypeChange {
	return &ColumnTypeChange{
		FromType: fromType,
		ToType:   toType,
		IsSafe:   isSafe,
	}
}
