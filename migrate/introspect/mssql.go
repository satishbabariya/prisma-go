// Package introspect provides SQL Server database introspection.
package introspect

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLServerIntrospector implements introspection for SQL Server
type SQLServerIntrospector struct {
	db *sql.DB
}

// NewSQLServerIntrospector creates a new SQL Server introspector
func NewSQLServerIntrospector(db *sql.DB) *SQLServerIntrospector {
	return &SQLServerIntrospector{db: db}
}

// Introspect introspects a SQL Server database
func (i *SQLServerIntrospector) Introspect(ctx context.Context) (*DatabaseSchema, error) {
	schema := &DatabaseSchema{
		Tables:    []Table{},
		Enums:     []Enum{},
		Views:     []View{},
		Sequences: []Sequence{},
	}

	// Query information_schema for tables
	tables, err := i.introspectTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect tables: %w", err)
	}
	schema.Tables = tables

	// Query for views
	views, err := i.introspectViews(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect views: %w", err)
	}
	schema.Views = views

	return schema, nil
}

// introspectTables introspects tables from SQL Server
func (i *SQLServerIntrospector) introspectTables(ctx context.Context) ([]Table, error) {
	// Foundation: Query SQL Server information_schema
	// Full implementation would query:
	// - INFORMATION_SCHEMA.TABLES for table list
	// - INFORMATION_SCHEMA.COLUMNS for columns
	// - sys.indexes for indexes
	// - sys.foreign_keys for foreign keys
	// - sys.key_constraints for primary keys

	query := `
		SELECT 
			TABLE_SCHEMA,
			TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_SCHEMA, TABLE_NAME
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			return nil, err
		}

		// Foundation: Full implementation would introspect columns, indexes, etc.
		tables = append(tables, Table{
			Name:   tableName,
			Schema: schemaName,
		})
	}

	return tables, rows.Err()
}

// introspectViews introspects views from SQL Server
func (i *SQLServerIntrospector) introspectViews(ctx context.Context) ([]View, error) {
	query := `
		SELECT 
			TABLE_SCHEMA,
			TABLE_NAME,
			VIEW_DEFINITION
		FROM INFORMATION_SCHEMA.VIEWS
		ORDER BY TABLE_SCHEMA, TABLE_NAME
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []View
	for rows.Next() {
		var schemaName, viewName, definition string
		if err := rows.Scan(&schemaName, &viewName, &definition); err != nil {
			return nil, err
		}

		views = append(views, View{
			Name:       viewName,
			Definition: definition,
		})
	}

	return views, rows.Err()
}
