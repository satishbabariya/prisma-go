// Package diff provides table comparison logic
package diff

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
)

// TableDiffer compares two tables and generates changes
type TableDiffer struct {
	prevTable *introspect.Table
	nextTable *introspect.Table
	db        *DifferDatabase
}

// NewTableDiffer creates a new TableDiffer
func NewTableDiffer(prevTable, nextTable *introspect.Table, db *DifferDatabase) *TableDiffer {
	return &TableDiffer{
		prevTable: prevTable,
		nextTable: nextTable,
		db:        db,
	}
}

// Compare compares the two tables and returns all changes
func (td *TableDiffer) Compare() []Change {
	var changes []Change

	// Compare columns
	changes = append(changes, td.compareColumns()...)

	// Compare indexes
	changes = append(changes, td.compareIndexes()...)

	// Compare foreign keys
	changes = append(changes, td.compareForeignKeys()...)

	// Compare primary key
	if td.primaryKeyChanged() {
		changes = append(changes, Change{
			Type:        ChangeTypeAlterColumn,
			Table:       td.nextTable.Name,
			Description: "Primary key changed",
			IsSafe:      false,
			Warnings:    []string{"Changing primary key may cause data loss"},
		})
	}

	return changes
}

// compareColumns compares columns between the two tables
func (td *TableDiffer) compareColumns() []Change {
	var changes []Change
	tableName := td.nextTable.Name

	// Get column pairs from database
	columnPairs := td.db.ColumnPairs(tableName)
	createdColumns := td.db.CreatedColumns(tableName)
	droppedColumns := td.db.DroppedColumns(tableName)

	// Added columns
	for _, col := range createdColumns {
		changes = append(changes, Change{
			Type:        ChangeTypeAddColumn,
			Table:       tableName,
			Column:      col.Name,
			Description: fmt.Sprintf("Add column '%s.%s' %s", tableName, col.Name, col.Type),
			IsSafe:      true,
			ColumnMetadata: &ColumnMetadata{
				Type:          col.Type,
				Nullable:      col.Nullable,
				DefaultValue:  col.DefaultValue,
				AutoIncrement: col.AutoIncrement,
			},
		})
	}

	// Dropped columns
	for _, col := range droppedColumns {
		changes = append(changes, Change{
			Type:        ChangeTypeDropColumn,
			Table:       tableName,
			Column:      col.Name,
			Description: fmt.Sprintf("Drop column '%s.%s'", tableName, col.Name),
			IsSafe:      false,
			Warnings:    []string{"Dropping column will delete all data in that column"},
		})
	}

	// Altered columns
	for _, pair := range columnPairs {
		colChanges := td.db.ColumnChanges(pair.TableName, pair.ColumnName)
		if colChanges != nil && colChanges.DiffersInSomething() {
			prevCol := pair.Column.Previous
			nextCol := pair.Column.Next

			change := Change{
				Type:        ChangeTypeAlterColumn,
				Table:       tableName,
				Column:      pair.ColumnName,
				Description: td.buildAlterColumnDescription(prevCol, nextCol, colChanges),
				IsSafe:      !colChanges.TypeChanged && nextCol.Nullable,
				ColumnMetadata: &ColumnMetadata{
					Type:          nextCol.Type,
					Nullable:      nextCol.Nullable,
					DefaultValue:  nextCol.DefaultValue,
					AutoIncrement: nextCol.AutoIncrement,
					OldType:       prevCol.Type,
					OldNullable:   &prevCol.Nullable,
				},
			}

			if colChanges.TypeChanged {
				change.Warnings = append(change.Warnings, "Changing column type may cause data loss")
			}
			if colChanges.NullableChanged && !nextCol.Nullable {
				change.Warnings = append(change.Warnings, "Making column required may fail if NULL values exist")
			}

			changes = append(changes, change)
		}
	}

	return changes
}

// buildAlterColumnDescription builds a description for AlterColumn changes
func (td *TableDiffer) buildAlterColumnDescription(prev, next *introspect.Column, changes *ColumnChanges) string {
	var parts []string

	if changes.TypeChanged {
		parts = append(parts, fmt.Sprintf("change type from %s to %s", prev.Type, next.Type))
	}
	if changes.NullableChanged {
		if next.Nullable {
			parts = append(parts, "make nullable")
		} else {
			parts = append(parts, "make required")
		}
	}
	if changes.DefaultChanged {
		if next.DefaultValue != nil {
			parts = append(parts, fmt.Sprintf("set default to %s", *next.DefaultValue))
		} else {
			parts = append(parts, "remove default")
		}
	}
	if changes.AutoIncrementChanged {
		if next.AutoIncrement {
			parts = append(parts, "enable auto-increment")
		} else {
			parts = append(parts, "disable auto-increment")
		}
	}

	description := fmt.Sprintf("%s column '%s.%s'", strings.Join(parts, ", "), td.nextTable.Name, next.Name)
	return strings.Title(description)
}

// compareIndexes compares indexes between the two tables
func (td *TableDiffer) compareIndexes() []Change {
	var changes []Change
	tableName := td.nextTable.Name

	// Build index maps
	prevIndexes := make(map[string]*introspect.Index)
	for i := range td.prevTable.Indexes {
		idx := &td.prevTable.Indexes[i]
		prevIndexes[idx.Name] = idx
	}

	nextIndexes := make(map[string]*introspect.Index)
	for i := range td.nextTable.Indexes {
		idx := &td.nextTable.Indexes[i]
		nextIndexes[idx.Name] = idx
	}

	// Find index pairs (potential renames)
	indexPairs := td.findIndexPairs(prevIndexes, nextIndexes)
	matchedPrev := make(map[string]bool)
	matchedNext := make(map[string]bool)

	// Process renames
	for prevName, nextName := range indexPairs {
		if td.db.flavour.CanRenameIndex() && td.db.flavour.IndexShouldBeRenamed(prevIndexes[prevName], nextIndexes[nextName]) {
			changes = append(changes, Change{
				Type:        ChangeTypeRenameIndex,
				Table:       tableName,
				Index:       nextName,
				OldName:     prevName,
				NewName:     nextName,
				Description: fmt.Sprintf("Rename index '%s' to '%s'", prevName, nextName),
				IsSafe:      true,
			})
			matchedPrev[prevName] = true
			matchedNext[nextName] = true
		}
	}

	// Created indexes (not matched)
	for name, idx := range nextIndexes {
		if !matchedNext[name] {
			changes = append(changes, Change{
				Type:        ChangeTypeCreateIndex,
				Table:       tableName,
				Index:       name,
				Description: fmt.Sprintf("Create index '%s' on %s(%s)", name, tableName, strings.Join(idx.Columns, ", ")),
				IsSafe:      true,
			})
		}
	}

	// Dropped indexes (not matched)
	for name := range prevIndexes {
		if !matchedPrev[name] {
			changes = append(changes, Change{
				Type:        ChangeTypeDropIndex,
				Table:       tableName,
				Index:       name,
				Description: fmt.Sprintf("Drop index '%s'", name),
				IsSafe:      true,
			})
		}
	}

	return changes
}

// findIndexPairs finds matching indexes by structure (for rename detection)
func (td *TableDiffer) findIndexPairs(prevIndexes, nextIndexes map[string]*introspect.Index) map[string]string {
	pairs := make(map[string]string)

	// Only match if there's exactly one candidate (avoid ambiguity)
	for prevName, prevIdx := range prevIndexes {
		// Count how many next indexes match this prev index
		var matches []string
		for nextName, nextIdx := range nextIndexes {
			if td.db.flavour.IndexesMatch(prevIdx, nextIdx) {
				matches = append(matches, nextName)
			}
		}

		// Only create pair if exactly one match
		if len(matches) == 1 {
			pairs[prevName] = matches[0]
		}
	}

	return pairs
}

// compareForeignKeys compares foreign keys between the two tables
func (td *TableDiffer) compareForeignKeys() []Change {
	var changes []Change
	tableName := td.nextTable.Name

	// Build FK maps
	prevFKs := make(map[string]*introspect.ForeignKey)
	for i := range td.prevTable.ForeignKeys {
		fk := &td.prevTable.ForeignKeys[i]
		prevFKs[fk.Name] = fk
	}

	nextFKs := make(map[string]*introspect.ForeignKey)
	for i := range td.nextTable.ForeignKeys {
		fk := &td.nextTable.ForeignKeys[i]
		nextFKs[fk.Name] = fk
	}

	// Find FK pairs (potential renames)
	fkPairs := td.findForeignKeyPairs(prevFKs, nextFKs)
	matchedPrev := make(map[string]bool)
	matchedNext := make(map[string]bool)

	// Process renames
	for prevName, nextName := range fkPairs {
		if td.db.flavour.CanRenameForeignKey() && td.db.flavour.ForeignKeysMatch(prevFKs[prevName], nextFKs[nextName]) {
			changes = append(changes, Change{
				Type:        ChangeTypeRenameForeignKey,
				Table:       tableName,
				OldName:     prevName,
				NewName:     nextName,
				Description: fmt.Sprintf("Rename foreign key '%s' to '%s'", prevName, nextName),
				IsSafe:      true,
			})
			matchedPrev[prevName] = true
			matchedNext[nextName] = true
		}
	}

	// Created foreign keys (not matched)
	for name := range nextFKs {
		if !matchedNext[name] {
			changes = append(changes, Change{
				Type:        ChangeTypeCreateForeignKey,
				Table:       tableName,
				Description: fmt.Sprintf("Create foreign key '%s'", name),
				IsSafe:      true,
			})
		}
	}

	// Dropped foreign keys (not matched)
	for name := range prevFKs {
		if !matchedPrev[name] {
			changes = append(changes, Change{
				Type:        ChangeTypeDropForeignKey,
				Table:       tableName,
				Description: fmt.Sprintf("Drop foreign key '%s'", name),
				IsSafe:      false,
				Warnings:    []string{"Dropping foreign key removes referential integrity"},
			})
		}
	}

	return changes
}

// findForeignKeyPairs finds matching foreign keys by structure (for rename detection)
func (td *TableDiffer) findForeignKeyPairs(prevFKs, nextFKs map[string]*introspect.ForeignKey) map[string]string {
	pairs := make(map[string]string)

	for prevName, prevFK := range prevFKs {
		// Count how many next FKs match this prev FK
		var matches []string
		for nextName, nextFK := range nextFKs {
			if td.db.flavour.ForeignKeysMatch(prevFK, nextFK) {
				matches = append(matches, nextName)
			}
		}

		// Only create pair if exactly one match
		if len(matches) == 1 {
			pairs[prevName] = matches[0]
		}
	}

	return pairs
}

// primaryKeyChanged checks if the primary key has changed
func (td *TableDiffer) primaryKeyChanged() bool {
	prevPK := td.prevTable.PrimaryKey
	nextPK := td.nextTable.PrimaryKey

	// Both nil
	if prevPK == nil && nextPK == nil {
		return false
	}

	// One nil, one not
	if prevPK == nil || nextPK == nil {
		return true
	}

	// Compare columns
	if len(prevPK.Columns) != len(nextPK.Columns) {
		return true
	}

	for i, col := range prevPK.Columns {
		if col != nextPK.Columns[i] {
			return true
		}
	}

	return false
}
