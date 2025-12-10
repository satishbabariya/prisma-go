// Package diff provides simplified schema comparison.
package diff

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// SimpleDiffer compares database schemas
type SimpleDiffer struct {
	provider string
}

// NewSimpleDiffer creates a new simple differ
func NewSimpleDiffer(provider string) *SimpleDiffer {
	return &SimpleDiffer{provider: provider}
}

// CompareSchemas compares two database schemas
func (d *SimpleDiffer) CompareSchemas(source, target *introspect.DatabaseSchema) *DiffResult {
	result := &DiffResult{
		TablesToCreate: []TableChange{},
		TablesToAlter:  []TableChange{},
		TablesToDrop:   []TableChange{},
		Changes:        []Change{},
	}

	// Build map of target tables
	targetTables := make(map[string]introspect.Table)
	for _, table := range target.Tables {
		targetTables[table.Name] = table
	}

	// Build map of source tables
	sourceTables := make(map[string]introspect.Table)
	for _, table := range source.Tables {
		sourceTables[table.Name] = table
	}

	// Tables to create (in target, not in source)
	for _, targetTable := range target.Tables {
		if _, exists := sourceTables[targetTable.Name]; !exists {
			result.TablesToCreate = append(result.TablesToCreate, TableChange{
				Name:   targetTable.Name,
				Action: "CREATE",
			})
			result.Changes = append(result.Changes, Change{
				Type:        "CreateTable",
				Table:       targetTable.Name,
				Description: fmt.Sprintf("Create table '%s'", targetTable.Name),
				IsSafe:      true,
			})
		}
	}

	// Tables to drop (in source, not in target)
	// Exclude system tables like _prisma_migrations
	systemTables := map[string]bool{
		"_prisma_migrations": true,
	}

	for _, sourceTable := range source.Tables {
		if _, exists := targetTables[sourceTable.Name]; !exists {
			// Skip system tables
			if systemTables[sourceTable.Name] {
				continue
			}
			result.TablesToDrop = append(result.TablesToDrop, TableChange{
				Name:   sourceTable.Name,
				Action: "DROP",
			})
			result.Changes = append(result.Changes, Change{
				Type:        "DropTable",
				Table:       sourceTable.Name,
				Description: fmt.Sprintf("Drop table '%s'", sourceTable.Name),
				IsSafe:      false,
				Warnings:    []string{"Dropping table will delete all data"},
			})
		}
	}

	// Tables to alter (in both, but different)
	for _, targetTable := range target.Tables {
		if sourceTable, exists := sourceTables[targetTable.Name]; exists {
			changes := d.compareTables(&sourceTable, &targetTable)
			if len(changes) > 0 {
				result.TablesToAlter = append(result.TablesToAlter, TableChange{
					Name:    targetTable.Name,
					Action:  "ALTER",
					Changes: changes,
				})
				result.Changes = append(result.Changes, changes...)
			}
		}
	}

	return result
}

// compareTables compares two tables
func (d *SimpleDiffer) compareTables(source, target *introspect.Table) []Change {
	var changes []Change

	// Build column maps
	sourceColumns := make(map[string]introspect.Column)
	for _, col := range source.Columns {
		sourceColumns[col.Name] = col
	}

	targetColumns := make(map[string]introspect.Column)
	for _, col := range target.Columns {
		targetColumns[col.Name] = col
	}

	// Columns to add
	for colName, targetCol := range targetColumns {
		if _, exists := sourceColumns[colName]; !exists {
			changes = append(changes, Change{
				Type:        "AddColumn",
				Table:       target.Name,
				Column:      colName,
				Description: fmt.Sprintf("Add column '%s.%s' %s", target.Name, colName, targetCol.Type),
				IsSafe:      true,
				ColumnMetadata: &ColumnMetadata{
					Type:          targetCol.Type,
					Nullable:      targetCol.Nullable,
					DefaultValue:  targetCol.DefaultValue,
					AutoIncrement: targetCol.AutoIncrement,
				},
			})
		}
	}

	// Columns to drop
	for colName := range sourceColumns {
		if _, exists := targetColumns[colName]; !exists {
			changes = append(changes, Change{
				Type:        "DropColumn",
				Table:       target.Name,
				Column:      colName,
				Description: fmt.Sprintf("Drop column '%s.%s'", target.Name, colName),
				IsSafe:      false,
				Warnings:    []string{"Dropping column will delete all data in that column"},
			})
		}
	}

	// Columns to alter
	for colName, targetCol := range targetColumns {
		if sourceCol, exists := sourceColumns[colName]; exists {
			colChanges := d.compareColumns(&sourceCol, &targetCol, target.Name, colName)
			changes = append(changes, colChanges...)
		}
	}

	// Index changes
	indexChanges := d.compareIndexes(source, target)
	changes = append(changes, indexChanges...)

	return changes
}

// compareColumns compares two columns
func (d *SimpleDiffer) compareColumns(source, target *introspect.Column, tableName, columnName string) []Change {
	var changes []Change

	// Check if type, nullable, default, or auto-increment changed
	typeChanged := !d.typesEqual(source.Type, target.Type)
	nullableChanged := source.Nullable != target.Nullable
	defaultChanged := !d.defaultValuesEqual(source.DefaultValue, target.DefaultValue)
	autoIncrementChanged := source.AutoIncrement != target.AutoIncrement

	// If multiple properties changed, create a single AlterColumn change
	if typeChanged || nullableChanged || defaultChanged || autoIncrementChanged {
		oldNullable := source.Nullable
		change := Change{
			Type:        "AlterColumn",
			Table:       tableName,
			Column:      columnName,
			Description: d.buildAlterColumnDescription(source, target, tableName, columnName, typeChanged, nullableChanged, defaultChanged, autoIncrementChanged),
			IsSafe:      !typeChanged && target.Nullable, // Only safe if not changing type and making nullable
			ColumnMetadata: &ColumnMetadata{
				Type:          target.Type,
				Nullable:      target.Nullable,
				DefaultValue:  target.DefaultValue,
				AutoIncrement: target.AutoIncrement,
				OldType:       source.Type,
				OldNullable:   &oldNullable,
			},
		}

		if typeChanged {
			change.Warnings = append(change.Warnings, "Changing column type may cause data loss")
		}
		if nullableChanged && !target.Nullable {
			change.Warnings = append(change.Warnings, "Making column required may fail if NULL values exist")
		}

		changes = append(changes, change)
	}

	return changes
}

// buildAlterColumnDescription builds a description for AlterColumn changes
func (d *SimpleDiffer) buildAlterColumnDescription(source, target *introspect.Column, tableName, columnName string, typeChanged, nullableChanged, defaultChanged, autoIncrementChanged bool) string {
	var parts []string

	if typeChanged {
		parts = append(parts, fmt.Sprintf("change type from %s to %s", source.Type, target.Type))
	}
	if nullableChanged {
		if target.Nullable {
			parts = append(parts, "make nullable")
		} else {
			parts = append(parts, "make required")
		}
	}
	if defaultChanged {
		if target.DefaultValue != nil {
			parts = append(parts, fmt.Sprintf("set default to %s", *target.DefaultValue))
		} else {
			parts = append(parts, "remove default")
		}
	}
	if autoIncrementChanged {
		if target.AutoIncrement {
			parts = append(parts, "enable auto-increment")
		} else {
			parts = append(parts, "disable auto-increment")
		}
	}

	description := fmt.Sprintf("%s column '%s.%s'", strings.Join(parts, ", "), tableName, columnName)
	return strings.Title(description)
}

// defaultValuesEqual compares two default values
func (d *SimpleDiffer) defaultValuesEqual(val1, val2 *string) bool {
	if val1 == nil && val2 == nil {
		return true
	}
	if val1 == nil || val2 == nil {
		return false
	}
	return *val1 == *val2
}

// compareIndexes compares indexes between tables
func (d *SimpleDiffer) compareIndexes(source, target *introspect.Table) []Change {
	var changes []Change

	sourceIndexes := make(map[string]introspect.Index)
	for _, idx := range source.Indexes {
		sourceIndexes[idx.Name] = idx
	}

	targetIndexes := make(map[string]introspect.Index)
	for _, idx := range target.Indexes {
		targetIndexes[idx.Name] = idx
	}

	// Indexes to create
	for idxName, idx := range targetIndexes {
		if _, exists := sourceIndexes[idxName]; !exists {
			changes = append(changes, Change{
				Type:        "CreateIndex",
				Table:       target.Name,
				Index:       idxName,
				Description: fmt.Sprintf("Create index '%s' on %s(%s)", idxName, target.Name, strings.Join(idx.Columns, ", ")),
				IsSafe:      true,
			})
		}
	}

	// Indexes to drop
	for idxName := range sourceIndexes {
		if _, exists := targetIndexes[idxName]; !exists {
			changes = append(changes, Change{
				Type:        "DropIndex",
				Table:       target.Name,
				Index:       idxName,
				Description: fmt.Sprintf("Drop index '%s'", idxName),
				IsSafe:      true,
			})
		}
	}

	return changes
}

// typesEqual checks if two types are equivalent
func (d *SimpleDiffer) typesEqual(type1, type2 string) bool {
	// Normalize types
	type1 = strings.ToUpper(type1)
	type2 = strings.ToUpper(type2)

	// Direct match
	if type1 == type2 {
		return true
	}

	// Common equivalents
	equivalents := map[string][]string{
		"INTEGER":     {"INT", "INT4", "SERIAL"},
		"BIGINT":      {"INT8", "BIGSERIAL"},
		"VARCHAR":     {"CHARACTER VARYING", "TEXT"},
		"BOOLEAN":     {"BOOL"},
		"TIMESTAMP":   {"TIMESTAMP WITHOUT TIME ZONE"},
		"TIMESTAMPTZ": {"TIMESTAMP WITH TIME ZONE"},
		"FLOAT":       {"DOUBLE PRECISION", "FLOAT8", "REAL"},
		"DECIMAL":     {"NUMERIC"},
	}

	for key, variants := range equivalents {
		if type1 == key || contains(variants, type1) {
			if type2 == key || contains(variants, type2) {
				return true
			}
		}
	}

	// Check if they start with the same base type (for VARCHAR(255) vs VARCHAR(100))
	if strings.Contains(type1, "(") && strings.Contains(type2, "(") {
		base1 := strings.Split(type1, "(")[0]
		base2 := strings.Split(type2, "(")[0]
		return base1 == base2
	}

	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
