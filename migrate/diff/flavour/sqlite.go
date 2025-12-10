// Package flavour provides SQLite-specific differ logic
package flavour

import (
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// SQLiteFlavour implements DifferFlavour for SQLite
type SQLiteFlavour struct{}

// NewSQLiteFlavour creates a new SQLite flavour
func NewSQLiteFlavour() DifferFlavour {
	return &SQLiteFlavour{}
}

// IndexesMatch checks if two indexes match by structure
func (f *SQLiteFlavour) IndexesMatch(prev, next *introspect.Index) bool {
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
func (f *SQLiteFlavour) ForeignKeysMatch(prev, next *introspect.ForeignKey) bool {
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
func (f *SQLiteFlavour) ColumnTypeChange(prev, next *introspect.Column) *ColumnTypeChange {
	if prev.Type == next.Type {
		return nil
	}
	return NewColumnTypeChange(prev.Type, next.Type, false)
}

// ShouldRedefineTable determines if a table needs to be recreated
func (f *SQLiteFlavour) ShouldRedefineTable(tableName string, changes []string) bool {
	// SQLite has very limited ALTER TABLE support
	// Most changes require table recreation
	return len(changes) > 0
}

// CanRenameIndex returns whether index renames are supported
func (f *SQLiteFlavour) CanRenameIndex() bool {
	return false // SQLite doesn't support index renames
}

// CanRenameForeignKey returns whether foreign key renames are supported
func (f *SQLiteFlavour) CanRenameForeignKey() bool {
	return false // SQLite doesn't support FK renames
}

// IndexShouldBeRenamed determines if an index should be renamed
func (f *SQLiteFlavour) IndexShouldBeRenamed(prev, next *introspect.Index) bool {
	return false // Not supported
}

// LowerCasesTableNames returns true if table names should be lowercased
func (f *SQLiteFlavour) LowerCasesTableNames() bool {
	return false
}

// TableShouldBeIgnored returns true if a table should be ignored
func (f *SQLiteFlavour) TableShouldBeIgnored(tableName string) bool {
	return tableName == "_prisma_migrations"
}
