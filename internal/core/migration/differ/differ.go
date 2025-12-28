// Package differ implements database state comparison.
package differ

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// DatabaseDiffer implements the Differ interface.
type DatabaseDiffer struct{}

// NewDatabaseDiffer creates a new database differ.
func NewDatabaseDiffer() *DatabaseDiffer {
	return &DatabaseDiffer{}
}

// Compare compares two database states and returns the changes needed.
func (d *DatabaseDiffer) Compare(ctx context.Context, from, to *domain.DatabaseState) ([]domain.Change, error) {
	var changes []domain.Change

	// Compare tables
	tableChanges, err := d.compareTables(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to compare tables: %w", err)
	}
	changes = append(changes, tableChanges...)

	return changes, nil
}

// CompareTables compares two tables and returns the changes.
func (d *DatabaseDiffer) CompareTables(ctx context.Context, from, to *domain.Table) ([]domain.Change, error) {
	var changes []domain.Change

	// Compare columns
	columnChanges, err := d.compareTableColumns(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to compare columns: %w", err)
	}
	changes = append(changes, columnChanges...)

	// Compare indexes
	indexChanges := d.compareTableIndexes(from, to)
	changes = append(changes, indexChanges...)

	return changes, nil
}

// CompareColumns compares two columns and returns the changes.
func (d *DatabaseDiffer) CompareColumns(ctx context.Context, from, to *domain.Column) ([]domain.Change, error) {
	var changes []domain.Change

	// Check if column type changed
	if from.Type != to.Type {
		changes = append(changes, &AlterColumnChange{
			TableName:  "", // Will be set by caller
			ColumnName: from.Name,
			OldType:    from.Type,
			NewType:    to.Type,
		})
	}

	// Check if nullable changed
	if from.IsNullable != to.IsNullable {
		changes = append(changes, &AlterColumnChange{
			TableName:   "", // Will be set by caller
			ColumnName:  from.Name,
			OldNullable: from.IsNullable,
			NewNullable: to.IsNullable,
		})
	}

	return changes, nil
}

// compareTables compares tables between two database states.
func (d *DatabaseDiffer) compareTables(ctx context.Context, from, to *domain.DatabaseState) ([]domain.Change, error) {
	var changes []domain.Change

	// Create maps for efficient lookup
	fromTables := make(map[string]*domain.Table)
	toTables := make(map[string]*domain.Table)

	for i := range from.Tables {
		fromTables[from.Tables[i].Name] = &from.Tables[i]
	}
	for i := range to.Tables {
		toTables[to.Tables[i].Name] = &to.Tables[i]
	}

	// Find new tables
	for name, table := range toTables {
		if _, exists := fromTables[name]; !exists {
			changes = append(changes, &CreateTableChange{
				Table: *table,
			})
		}
	}

	// Find removed tables
	for name := range fromTables {
		if _, exists := toTables[name]; !exists {
			changes = append(changes, &DropTableChange{
				TableName: name,
			})
		}
	}

	// Find modified tables
	for name, fromTable := range fromTables {
		if toTable, exists := toTables[name]; exists {
			tableChanges, err := d.CompareTables(ctx, fromTable, toTable)
			if err != nil {
				return nil, fmt.Errorf("failed to compare table %s: %w", name, err)
			}
			changes = append(changes, tableChanges...)
		}
	}

	return changes, nil
}

// compareTableColumns compares columns between two tables.
func (d *DatabaseDiffer) compareTableColumns(ctx context.Context, from, to *domain.Table) ([]domain.Change, error) {
	var changes []domain.Change

	// Create maps for efficient lookup
	fromColumns := make(map[string]*domain.Column)
	toColumns := make(map[string]*domain.Column)

	for i := range from.Columns {
		fromColumns[from.Columns[i].Name] = &from.Columns[i]
	}
	for i := range to.Columns {
		toColumns[to.Columns[i].Name] = &to.Columns[i]
	}

	// Find new columns
	for name, column := range toColumns {
		if _, exists := fromColumns[name]; !exists {
			changes = append(changes, &AddColumnChange{
				TableName: to.Name,
				Column:    *column,
			})
		}
	}

	// Find removed columns
	for name := range fromColumns {
		if _, exists := toColumns[name]; !exists {
			changes = append(changes, &DropColumnChange{
				TableName:  from.Name,
				ColumnName: name,
			})
		}
	}

	// Find modified columns
	for name, fromCol := range fromColumns {
		if toCol, exists := toColumns[name]; exists {
			colChanges, err := d.CompareColumns(ctx, fromCol, toCol)
			if err != nil {
				return nil, fmt.Errorf("failed to compare column %s: %w", name, err)
			}
			// Set table name for column changes
			for _, change := range colChanges {
				if altChange, ok := change.(*AlterColumnChange); ok {
					altChange.TableName = from.Name
				}
			}
			changes = append(changes, colChanges...)
		}
	}

	return changes, nil
}

// compareTableIndexes compares indexes between two tables.
func (d *DatabaseDiffer) compareTableIndexes(from, to *domain.Table) []domain.Change {
	var changes []domain.Change

	// Create maps for efficient lookup
	fromIndexes := make(map[string]*domain.Index)
	toIndexes := make(map[string]*domain.Index)

	for i := range from.Indexes {
		fromIndexes[from.Indexes[i].Name] = &from.Indexes[i]
	}
	for i := range to.Indexes {
		toIndexes[to.Indexes[i].Name] = &to.Indexes[i]
	}

	// Find new indexes
	for name, index := range toIndexes {
		if _, exists := fromIndexes[name]; !exists {
			changes = append(changes, &CreateIndexChange{
				TableName: to.Name,
				Index:     *index,
			})
		}
	}

	// Find removed indexes
	for name := range fromIndexes {
		if _, exists := toIndexes[name]; !exists {
			changes = append(changes, &DropIndexChange{
				TableName: from.Name,
				IndexName: name,
			})
		}
	}

	return changes
}

// Ensure DatabaseDiffer implements Differ interface.
var _ domain.Differ = (*DatabaseDiffer)(nil)
