// Package diff provides the central database tracking structure for schema comparison
package diff

import (
	"sort"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/diff/flavour"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// DifferDatabase is the central structure tracking all schema elements
type DifferDatabase struct {
	flavour flavour.DifferFlavour
	// The schemas being diffed
	prevSchema *introspect.DatabaseSchema
	nextSchema *introspect.DatabaseSchema
	// Table name -> table pair
	tables map[string]MigrationPair[*introspect.Table]
	// (table_name, column_name) -> column pair
	columns map[string]map[string]MigrationPair[*introspect.Column]
	// Column changes: (table_name, column_name) -> ColumnChanges
	columnChanges map[string]map[string]*ColumnChanges
	// Tables that need to be redefined (dropped and recreated)
	tablesToRedefine map[string]bool
}

// NewDifferDatabase creates a new DifferDatabase
func NewDifferDatabase(prevSchema, nextSchema *introspect.DatabaseSchema, f flavour.DifferFlavour) *DifferDatabase {
	db := &DifferDatabase{
		flavour:          f,
		prevSchema:       prevSchema,
		nextSchema:       nextSchema,
		tables:           make(map[string]MigrationPair[*introspect.Table]),
		columns:          make(map[string]map[string]MigrationPair[*introspect.Column]),
		columnChanges:    make(map[string]map[string]*ColumnChanges),
		tablesToRedefine: make(map[string]bool),
	}

	db.buildTables()
	db.buildColumns()

	return db
}

// buildTables builds the table mapping
func (db *DifferDatabase) buildTables() {
	// Build map of previous tables
	prevTables := make(map[string]*introspect.Table)
	if db.prevSchema != nil {
		for i := range db.prevSchema.Tables {
			table := &db.prevSchema.Tables[i]
			tableName := db.normalizeTableName(table.Name)
			if db.flavour.TableShouldBeIgnored(tableName) {
				continue
			}
			prevTables[tableName] = table
		}
	}

	// Build map of next tables and create pairs
	nextTables := make(map[string]*introspect.Table)
	if db.nextSchema != nil {
		for i := range db.nextSchema.Tables {
			table := &db.nextSchema.Tables[i]
			tableName := db.normalizeTableName(table.Name)
			if db.flavour.TableShouldBeIgnored(tableName) {
				continue
			}
			nextTables[tableName] = table
		}
	}

	// Create table pairs
	for tableName, prevTable := range prevTables {
		nextTable := nextTables[tableName]
		db.tables[tableName] = MigrationPair[*introspect.Table]{
			Previous: prevTable,
			Next:     nextTable,
		}
	}

	// Add tables that only exist in next schema
	for tableName, nextTable := range nextTables {
		if _, exists := prevTables[tableName]; !exists {
			db.tables[tableName] = MigrationPair[*introspect.Table]{
				Previous: nil,
				Next:     nextTable,
			}
		}
	}
}

// buildColumns builds the column mapping and detects changes
func (db *DifferDatabase) buildColumns() {
	for tableName, tablePair := range db.tables {
		if !HasBoth(tablePair) {
			continue
		}

		prevTable := tablePair.Previous
		nextTable := tablePair.Next

		// Build column maps
		prevColumns := make(map[string]*introspect.Column)
		for i := range prevTable.Columns {
			col := &prevTable.Columns[i]
			prevColumns[col.Name] = col
		}

		nextColumns := make(map[string]*introspect.Column)
		for i := range nextTable.Columns {
			col := &nextTable.Columns[i]
			nextColumns[col.Name] = col
		}

		// Initialize column maps for this table
		if db.columns[tableName] == nil {
			db.columns[tableName] = make(map[string]MigrationPair[*introspect.Column])
		}
		if db.columnChanges[tableName] == nil {
			db.columnChanges[tableName] = make(map[string]*ColumnChanges)
		}

		// Create column pairs and detect changes
		for colName, prevCol := range prevColumns {
			nextCol := nextColumns[colName]
			db.columns[tableName][colName] = MigrationPair[*introspect.Column]{
				Previous: prevCol,
				Next:     nextCol,
			}

			// Detect column changes if both exist
			if nextCol != nil {
				changes := allColumnChanges(prevCol, nextCol, db.flavour)
				db.columnChanges[tableName][colName] = changes
			}
		}

		// Add columns that only exist in next schema
		for colName, nextCol := range nextColumns {
			if _, exists := prevColumns[colName]; !exists {
				db.columns[tableName][colName] = MigrationPair[*introspect.Column]{
					Previous: nil,
					Next:     nextCol,
				}
			}
		}
	}
}

// normalizeTableName normalizes table name based on flavour
func (db *DifferDatabase) normalizeTableName(name string) string {
	if db.flavour.LowerCasesTableNames() {
		return strings.ToLower(name)
	}
	return name
}

// CreatedTables returns tables that exist only in the next schema
func (db *DifferDatabase) CreatedTables() []*introspect.Table {
	var result []*introspect.Table
	for _, pair := range db.tables {
		if pair.Next != nil && pair.Previous == nil {
			result = append(result, pair.Next)
		}
	}
	return result
}

// DroppedTables returns tables that exist only in the previous schema
func (db *DifferDatabase) DroppedTables() []*introspect.Table {
	var result []*introspect.Table
	for _, pair := range db.tables {
		if pair.Previous != nil && pair.Next == nil {
			result = append(result, pair.Previous)
		}
	}
	return result
}

// TablePairs returns tables that exist in both schemas
func (db *DifferDatabase) TablePairs() []TablePair {
	var result []TablePair
	for tableName, pair := range db.tables {
		if HasBoth(pair) {
			result = append(result, TablePair{
				Name:  tableName,
				Table: pair,
			})
		}
	}
	// Sort for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// TablePair represents a pair of tables
type TablePair struct {
	Name  string
	Table MigrationPair[*introspect.Table]
}

// ColumnPairs returns columns that exist in both schemas for a given table
func (db *DifferDatabase) ColumnPairs(tableName string) []ColumnPair {
	var result []ColumnPair
	tableCols, exists := db.columns[tableName]
	if !exists {
		return result
	}

	for colName, pair := range tableCols {
		if HasBoth(pair) {
			result = append(result, ColumnPair{
				TableName:  tableName,
				ColumnName: colName,
				Column:     pair,
			})
		}
	}
	return result
}

// ColumnPair represents a pair of columns
type ColumnPair struct {
	TableName  string
	ColumnName string
	Column     MigrationPair[*introspect.Column]
}

// CreatedColumns returns columns that exist only in the next schema for a given table
func (db *DifferDatabase) CreatedColumns(tableName string) []*introspect.Column {
	var result []*introspect.Column
	tableCols, exists := db.columns[tableName]
	if !exists {
		return result
	}

	for _, pair := range tableCols {
		if pair.Next != nil && pair.Previous == nil {
			result = append(result, pair.Next)
		}
	}
	return result
}

// DroppedColumns returns columns that exist only in the previous schema for a given table
func (db *DifferDatabase) DroppedColumns(tableName string) []*introspect.Column {
	var result []*introspect.Column
	tableCols, exists := db.columns[tableName]
	if !exists {
		return result
	}

	for _, pair := range tableCols {
		if pair.Previous != nil && pair.Next == nil {
			result = append(result, pair.Previous)
		}
	}
	return result
}

// ColumnChanges returns the changes for a column
func (db *DifferDatabase) ColumnChanges(tableName, columnName string) *ColumnChanges {
	if db.columnChanges[tableName] == nil {
		return nil
	}
	return db.columnChanges[tableName][columnName]
}

// SetTableToRedefine marks a table as needing redefinition
func (db *DifferDatabase) SetTableToRedefine(tableName string) {
	db.tablesToRedefine[tableName] = true
}

// ShouldRedefineTable returns true if a table should be redefined
func (db *DifferDatabase) ShouldRedefineTable(tableName string) bool {
	return db.tablesToRedefine[tableName]
}
