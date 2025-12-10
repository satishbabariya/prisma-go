// Package flavour provides MySQL-specific differ logic
package flavour

import (
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// MySQLFlavour implements DifferFlavour for MySQL
type MySQLFlavour struct{}

// NewMySQLFlavour creates a new MySQL flavour
func NewMySQLFlavour() DifferFlavour {
	return &MySQLFlavour{}
}

// IndexesMatch checks if two indexes match by structure
func (f *MySQLFlavour) IndexesMatch(prev, next *introspect.Index) bool {
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
func (f *MySQLFlavour) ForeignKeysMatch(prev, next *introspect.ForeignKey) bool {
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
func (f *MySQLFlavour) ColumnTypeChange(prev, next *introspect.Column) *ColumnTypeChange {
	if prev.Type == next.Type {
		return nil
	}
	return NewColumnTypeChange(prev.Type, next.Type, false)
}

// ShouldRedefineTable determines if a table needs to be recreated
func (f *MySQLFlavour) ShouldRedefineTable(tableName string, changes []string) bool {
	return false
}

// CanRenameIndex returns whether index renames are supported
func (f *MySQLFlavour) CanRenameIndex() bool {
	return true
}

// CanRenameForeignKey returns whether foreign key renames are supported
func (f *MySQLFlavour) CanRenameForeignKey() bool {
	return true
}

// IndexShouldBeRenamed determines if an index should be renamed
func (f *MySQLFlavour) IndexShouldBeRenamed(prev, next *introspect.Index) bool {
	return f.IndexesMatch(prev, next) && prev.Name != next.Name
}

// LowerCasesTableNames returns true if table names should be lowercased
func (f *MySQLFlavour) LowerCasesTableNames() bool {
	return true // MySQL lowercases table names
}

// TableShouldBeIgnored returns true if a table should be ignored
func (f *MySQLFlavour) TableShouldBeIgnored(tableName string) bool {
	return tableName == "_prisma_migrations"
}
