// Package sqlite implements SQLite database introspection.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/domain"
)

// Introspector implements domain.Introspector for SQLite.
type Introspector struct{}

// NewIntrospector creates a new SQLite introspector.
func NewIntrospector() *Introspector {
	return &Introspector{}
}

// IntrospectDatabase introspects the entire SQLite database.
func (i *Introspector) IntrospectDatabase(ctx context.Context, db *sql.DB, schemas []string) (*domain.IntrospectedDatabase, error) {
	// SQLite doesn't have schemas in the traditional sense
	result := &domain.IntrospectedDatabase{
		Tables: []domain.IntrospectedTable{},
		Enums:  []domain.IntrospectedEnum{}, // SQLite doesn't have native enums
	}

	tables, err := i.introspectTables(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect tables: %w", err)
	}
	result.Tables = tables

	return result, nil
}

// introspectTables gets all tables.
func (i *Introspector) introspectTables(ctx context.Context, db *sql.DB) ([]domain.IntrospectedTable, error) {
	query := `
		SELECT name 
		FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []domain.IntrospectedTable
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		table, err := i.IntrospectTable(ctx, db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect table %s: %w", tableName, err)
		}
		tables = append(tables, *table)
	}

	return tables, rows.Err()
}

// IntrospectTable introspects a single table.
func (i *Introspector) IntrospectTable(ctx context.Context, db *sql.DB, tableName string) (*domain.IntrospectedTable, error) {
	table := &domain.IntrospectedTable{
		Name:        tableName,
		Columns:     []domain.IntrospectedColumn{},
		Indexes:     []domain.IntrospectedIndex{},
		ForeignKeys: []domain.IntrospectedForeignKey{},
	}

	// Get columns using PRAGMA
	columns, err := i.introspectColumns(ctx, db, tableName)
	if err != nil {
		return nil, err
	}
	table.Columns = columns

	// Get primary key
	pk := i.extractPrimaryKey(columns)
	table.PrimaryKey = pk

	// Get indexes
	indexes, err := i.introspectIndexes(ctx, db, tableName)
	if err != nil {
		return nil, err
	}
	table.Indexes = indexes

	// Get foreign keys
	fks, err := i.introspectForeignKeys(ctx, db, tableName)
	if err != nil {
		return nil, err
	}
	table.ForeignKeys = fks

	return table, nil
}

// introspectColumns gets all columns using PRAGMA table_info.
func (i *Introspector) introspectColumns(ctx context.Context, db *sql.DB, tableName string) ([]domain.IntrospectedColumn, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []domain.IntrospectedColumn
	for rows.Next() {
		var cid int
		var col domain.IntrospectedColumn
		var notNull int
		var pk int
		var defaultVal sql.NullString

		if err := rows.Scan(&cid, &col.Name, &col.Type, &notNull, &defaultVal, &pk); err != nil {
			return nil, err
		}

		col.IsNullable = notNull == 0
		col.IsPrimaryKey = pk > 0
		col.OrdinalPosition = cid + 1

		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// extractPrimaryKey extracts primary key from columns.
func (i *Introspector) extractPrimaryKey(columns []domain.IntrospectedColumn) *domain.IntrospectedPrimaryKey {
	var pkColumns []string
	for _, col := range columns {
		if col.IsPrimaryKey {
			pkColumns = append(pkColumns, col.Name)
		}
	}

	if len(pkColumns) == 0 {
		return nil
	}

	return &domain.IntrospectedPrimaryKey{
		Name:    "PRIMARY",
		Columns: pkColumns,
	}
}

// introspectIndexes gets all indexes using PRAGMA index_list.
func (i *Introspector) introspectIndexes(ctx context.Context, db *sql.DB, tableName string) ([]domain.IntrospectedIndex, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []domain.IntrospectedIndex
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		// Skip auto-created indexes
		if strings.HasPrefix(name, "sqlite_autoindex") {
			continue
		}

		// Get index columns
		columns, err := i.getIndexColumns(ctx, db, name)
		if err != nil {
			return nil, err
		}

		indexes = append(indexes, domain.IntrospectedIndex{
			Name:     name,
			Columns:  columns,
			IsUnique: unique == 1,
			Type:     "btree", // SQLite uses B-tree for all indexes
		})
	}

	return indexes, rows.Err()
}

// getIndexColumns gets columns for a specific index.
func (i *Introspector) getIndexColumns(ctx context.Context, db *sql.DB, indexName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA index_info(%s)", indexName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var seqno, cid int
		var name sql.NullString

		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, err
		}

		if name.Valid {
			columns = append(columns, name.String)
		}
	}

	return columns, rows.Err()
}

// introspectForeignKeys gets all foreign keys using PRAGMA foreign_key_list.
func (i *Introspector) introspectForeignKeys(ctx context.Context, db *sql.DB, tableName string) ([]domain.IntrospectedForeignKey, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[int]*domain.IntrospectedForeignKey)

	for rows.Next() {
		var id, seq int
		var table, from, to string
		var onUpdate, onDelete, match sql.NullString

		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}

		if fk, exists := fkMap[id]; exists {
			fk.ColumnNames = append(fk.ColumnNames, from)
			fk.ReferencedColumns = append(fk.ReferencedColumns, to)
		} else {
			var updateAction, deleteAction domain.ReferentialAction
			if onUpdate.Valid {
				updateAction = mapReferentialAction(onUpdate.String)
			}
			if onDelete.Valid {
				deleteAction = mapReferentialAction(onDelete.String)
			}

			fkMap[id] = &domain.IntrospectedForeignKey{
				Name:              fmt.Sprintf("fk_%s_%d", tableName, id),
				ColumnNames:       []string{from},
				ReferencedTable:   table,
				ReferencedColumns: []string{to},
				OnDelete:          deleteAction,
				OnUpdate:          updateAction,
			}
		}
	}

	var fks []domain.IntrospectedForeignKey
	for _, fk := range fkMap {
		fks = append(fks, *fk)
	}

	return fks, rows.Err()
}

// GetDatabaseVersion returns the SQLite version.
func (i *Introspector) GetDatabaseVersion(ctx context.Context, db *sql.DB) (string, error) {
	var version string
	err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	return version, err
}

// GetSchemas returns available schemas (always returns ["main"] for SQLite).
func (i *Introspector) GetSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	return []string{"main"}, nil
}

func mapReferentialAction(action string) domain.ReferentialAction {
	switch strings.ToUpper(action) {
	case "CASCADE":
		return domain.Cascade
	case "RESTRICT":
		return domain.Restrict
	case "SET NULL":
		return domain.SetNull
	case "SET DEFAULT":
		return domain.SetDefault
	default:
		return domain.NoAction
	}
}
