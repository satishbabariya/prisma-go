// Package flavour provides PostgreSQL-specific differ logic
package flavour

import (
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// PostgresFlavour implements DifferFlavour for PostgreSQL
type PostgresFlavour struct{}

// NewPostgresFlavour creates a new PostgreSQL flavour
func NewPostgresFlavour() DifferFlavour {
	return &PostgresFlavour{}
}

// IndexesMatch checks if two indexes match by structure
func (f *PostgresFlavour) IndexesMatch(prev, next *introspect.Index) bool {
	if len(prev.Columns) != len(next.Columns) {
		return false
	}
	if prev.IsUnique != next.IsUnique {
		return false
	}
	for i, col := range prev.Columns {
		if col != next.Columns[i] {
			return false
		}
	}
	return true
}

// ForeignKeysMatch checks if two foreign keys match by structure
func (f *PostgresFlavour) ForeignKeysMatch(prev, next *introspect.ForeignKey) bool {
	if len(prev.Columns) != len(next.Columns) {
		return false
	}
	if prev.ReferencedTable != next.ReferencedTable {
		return false
	}
	if len(prev.ReferencedColumns) != len(next.ReferencedColumns) {
		return false
	}
	for i, col := range prev.Columns {
		if col != next.Columns[i] {
			return false
		}
	}
	for i, col := range prev.ReferencedColumns {
		if col != next.ReferencedColumns[i] {
			return false
		}
	}
	if prev.OnDelete != next.OnDelete {
		return false
	}
	if prev.OnUpdate != next.OnUpdate {
		return false
	}
	return true
}

// ColumnTypeChange detects if a column type has changed
func (f *PostgresFlavour) ColumnTypeChange(prev, next *introspect.Column) *ColumnTypeChange {
	if prev.Type == next.Type {
		return nil
	}
	// Type changed - not safe by default
	return NewColumnTypeChange(prev.Type, next.Type, false)
}

// ShouldRedefineTable determines if a table needs to be recreated
func (f *PostgresFlavour) ShouldRedefineTable(tableName string, changes []string) bool {
	// PostgreSQL supports most ALTER TABLE operations
	// Only redefine for very complex changes
	return false
}

// CanRenameIndex returns whether index renames are supported
func (f *PostgresFlavour) CanRenameIndex() bool {
	return true
}

// CanRenameForeignKey returns whether foreign key renames are supported
func (f *PostgresFlavour) CanRenameForeignKey() bool {
	return true
}

// IndexShouldBeRenamed determines if an index should be renamed
func (f *PostgresFlavour) IndexShouldBeRenamed(prev, next *introspect.Index) bool {
	return f.IndexesMatch(prev, next) && prev.Name != next.Name
}

// LowerCasesTableNames returns true if table names should be lowercased
func (f *PostgresFlavour) LowerCasesTableNames() bool {
	return false
}

// TableShouldBeIgnored returns true if a table should be ignored
func (f *PostgresFlavour) TableShouldBeIgnored(tableName string) bool {
	return tableName == "_prisma_migrations"
}
