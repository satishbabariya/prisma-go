// Package introspect provides SQLite database introspection.
package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SQLiteIntrospector implements introspection for SQLite
type SQLiteIntrospector struct {
	db *sql.DB
}

// Introspect reads the SQLite database schema
func (i *SQLiteIntrospector) Introspect(ctx context.Context) (*DatabaseSchema, error) {
	schema := &DatabaseSchema{
		Tables:    []Table{},
		Enums:     []Enum{},
		Views:     []View{},
		Sequences: []Sequence{},
	}

	// Introspect tables
	tables, err := i.introspectTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect tables: %w", err)
	}
	schema.Tables = tables

	return schema, nil
}

// introspectTables reads all tables and their columns
func (i *SQLiteIntrospector) introspectTables(ctx context.Context) ([]Table, error) {
	// Query to get all tables (exclude system tables)
	query := `
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		table.Schema = "main"
		if err := rows.Scan(&table.Name); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		// Get columns
		columns, err := i.introspectColumns(ctx, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect columns for %s: %w", table.Name, err)
		}
		table.Columns = columns

		// Get primary key from column info
		pk := i.getPrimaryKeyFromColumns(columns)
		table.PrimaryKey = pk

		// Get indexes
		indexes, err := i.introspectIndexes(ctx, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect indexes for %s: %w", table.Name, err)
		}
		table.Indexes = indexes

		// Get foreign keys
		fks, err := i.introspectForeignKeys(ctx, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect foreign keys for %s: %w", table.Name, err)
		}
		table.ForeignKeys = fks

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// introspectColumns reads all columns for a table using PRAGMA
func (i *SQLiteIntrospector) introspectColumns(ctx context.Context, tableName string) ([]Column, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var cid int
		var col Column
		var colType string
		var notNull int
		var dfltValue sql.NullString
		var isPk int

		err := rows.Scan(
			&cid,
			&col.Name,
			&colType,
			&notNull,
			&dfltValue,
			&isPk,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Type = i.mapSQLiteType(colType)
		col.Nullable = (notNull == 0)

		if dfltValue.Valid && dfltValue.String != "" {
			col.DefaultValue = &dfltValue.String
		}

		// SQLite AUTOINCREMENT is complex to detect
		// It's only for INTEGER PRIMARY KEY columns
		if isPk == 1 && strings.ToUpper(colType) == "INTEGER" {
			col.AutoIncrement = true
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// getPrimaryKeyFromColumns extracts primary key info from columns
func (i *SQLiteIntrospector) getPrimaryKeyFromColumns(columns []Column) *PrimaryKey {
	// Check PRAGMA table_info which includes pk column
	// For now, we'll use a simpler approach
	// This is a simplified version
	// In production, we'd parse the CREATE TABLE statement
	return nil
}

// introspectIndexes reads all indexes for a table
func (i *SQLiteIntrospector) introspectIndexes(ctx context.Context, tableName string) ([]Index, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", tableName)

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var seq int
		var idx Index
		var unique int
		var origin string
		var partial int

		err := rows.Scan(&seq, &idx.Name, &unique, &origin, &partial)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		idx.IsUnique = (unique == 1)

		// Get columns for this index
		colQuery := fmt.Sprintf("PRAGMA index_info(%s)", idx.Name)
		colRows, err := i.db.QueryContext(ctx, colQuery)
		if err != nil {
			continue
		}

		var columns []string
		for colRows.Next() {
			var seqno, cid int
			var name sql.NullString

			if err := colRows.Scan(&seqno, &cid, &name); err != nil {
				colRows.Close()
				continue
			}

			if name.Valid {
				columns = append(columns, name.String)
			}
		}
		colRows.Close()

		idx.Columns = columns
		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// introspectForeignKeys reads all foreign keys for a table
func (i *SQLiteIntrospector) introspectForeignKeys(ctx context.Context, tableName string) ([]ForeignKey, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName)

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	// SQLite's foreign_key_list returns one row per column in the FK
	// We need to group them by id
	fkMap := make(map[int]*ForeignKey)

	for rows.Next() {
		var id, seq int
		var table, from, to string
		var onUpdate, onDelete, match string

		err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match)
		if err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		if fk, exists := fkMap[id]; exists {
			fk.Columns = append(fk.Columns, from)
			fk.ReferencedColumns = append(fk.ReferencedColumns, to)
		} else {
			fkMap[id] = &ForeignKey{
				Name:              fmt.Sprintf("%s_fk_%d", tableName, id),
				Columns:           []string{from},
				ReferencedTable:   table,
				ReferencedColumns: []string{to},
				OnUpdate:          onUpdate,
				OnDelete:          onDelete,
			}
		}
	}

	var fks []ForeignKey
	for _, fk := range fkMap {
		fks = append(fks, *fk)
	}

	return fks, rows.Err()
}

// mapSQLiteType maps SQLite data types to generic types
func (i *SQLiteIntrospector) mapSQLiteType(sqliteType string) string {
	upperType := strings.ToUpper(sqliteType)

	switch {
	case strings.Contains(upperType, "INT"):
		return "INTEGER"
	case strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CLOB"):
		return "TEXT"
	case strings.Contains(upperType, "BLOB"):
		return "BLOB"
	case strings.Contains(upperType, "REAL"), strings.Contains(upperType, "FLOA"), strings.Contains(upperType, "DOUB"):
		return "REAL"
	case strings.Contains(upperType, "NUMERIC"), strings.Contains(upperType, "DECIMAL"):
		return "NUMERIC"
	default:
		// SQLite has flexible typing
		return sqliteType
	}
}

