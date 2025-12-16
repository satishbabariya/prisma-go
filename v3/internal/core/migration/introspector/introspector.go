// Package introspector implements database introspection.
package introspector

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// DatabaseIntrospector implements the Introspector interface.
type DatabaseIntrospector struct {
	db database.Adapter
}

// NewDatabaseIntrospector creates a new database introspector.
func NewDatabaseIntrospector(db database.Adapter) *DatabaseIntrospector {
	return &DatabaseIntrospector{
		db: db,
	}
}

// IntrospectDatabase introspects the entire database.
func (i *DatabaseIntrospector) IntrospectDatabase(ctx context.Context) (*domain.DatabaseState, error) {
	if i.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	// Get list of tables
	tables, err := i.ListTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	// Introspect each table
	var dbTables []domain.Table
	for _, tableName := range tables {
		table, err := i.IntrospectTable(ctx, tableName)
		if err != nil {
			// Log error but continue with other tables
			continue
		}
		dbTables = append(dbTables, *table)
	}

	return &domain.DatabaseState{
		Tables: dbTables,
	}, nil
}

// IntrospectTable introspects a specific table.
func (i *DatabaseIntrospector) IntrospectTable(ctx context.Context, tableName string) (*domain.Table, error) {
	if i.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	// Get columns
	columns, err := i.getTableColumns(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}

	// Get indexes
	indexes, err := i.getTableIndexes(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
	}

	// Get constraints
	constraints, err := i.getTableConstraints(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints for table %s: %w", tableName, err)
	}

	return &domain.Table{
		Name:        tableName,
		Columns:     columns,
		Indexes:     indexes,
		Constraints: constraints,
	}, nil
}

// ListTables lists all tables in the database.
func (i *DatabaseIntrospector) ListTables(ctx context.Context) ([]string, error) {
	if i.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	var query string
	switch i.db.GetDialect() {
	case database.PostgreSQL:
		query = `
			SELECT tablename 
			FROM pg_tables 
			WHERE schemaname = 'public'
			ORDER BY tablename
		`
	case database.MySQL:
		query = `
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = DATABASE()
			ORDER BY table_name
		`
	case database.SQLite:
		query = `
			SELECT name 
			FROM sqlite_master 
			WHERE type='table' AND name NOT LIKE 'sqlite_%'
			ORDER BY name
		`
	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", i.db.GetDialect())
	}

	rows, err := i.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// getTableColumns gets all columns for a table.
func (i *DatabaseIntrospector) getTableColumns(ctx context.Context, tableName string) ([]domain.Column, error) {
	var query string
	switch i.db.GetDialect() {
	case database.PostgreSQL:
		query = `
			SELECT 
				column_name,
				data_type,
				is_nullable,
				column_default
			FROM information_schema.columns
			WHERE table_name = $1 AND table_schema = 'public'
			ORDER BY ordinal_position
		`
	case database.MySQL:
		query = `
			SELECT 
				column_name,
				data_type,
				is_nullable,
				column_default
			FROM information_schema.columns
			WHERE table_name = ? AND table_schema = DATABASE()
			ORDER BY ordinal_position
		`
	case database.SQLite:
		query = fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", i.db.GetDialect())
	}

	rows, err := i.db.Query(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []domain.Column
	for rows.Next() {
		var col domain.Column
		var nullable string
		var defaultValue sql.NullString

		if i.db.GetDialect() == database.SQLite {
			// SQLite PRAGMA returns: cid, name, type, notnull, dflt_value, pk
			var cid, notnull, pk int
			if err := rows.Scan(&cid, &col.Name, &col.Type, &notnull, &defaultValue, &pk); err != nil {
				return nil, fmt.Errorf("failed to scan column: %w", err)
			}
			col.IsNullable = notnull == 0
		} else {
			if err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultValue); err != nil {
				return nil, fmt.Errorf("failed to scan column: %w", err)
			}
			col.IsNullable = nullable == "YES"
		}

		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// getTableIndexes gets all indexes for a table.
func (i *DatabaseIntrospector) getTableIndexes(ctx context.Context, tableName string) ([]domain.Index, error) {
	var query string
	switch i.db.GetDialect() {
	case database.PostgreSQL:
		query = `
			SELECT
				i.relname as index_name,
				a.attname as column_name,
				ix.indisunique as is_unique
			FROM pg_class t
			JOIN pg_index ix ON t.oid = ix.indrelid
			JOIN pg_class i ON i.oid = ix.indexrelid
			JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
			WHERE t.relname = $1
			ORDER BY i.relname, a.attnum
		`
	case database.MySQL:
		query = `
			SELECT
				index_name,
				column_name,
				non_unique = 0 as is_unique
			FROM information_schema.statistics
			WHERE table_name = ? AND table_schema = DATABASE()
			ORDER BY index_name, seq_in_index
		`
	case database.SQLite:
		query = fmt.Sprintf("PRAGMA index_list(%s)", tableName)
	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", i.db.GetDialect())
	}

	rows, err := i.db.Query(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*domain.Index)
	for rows.Next() {
		var indexName, columnName string
		var isUnique bool

		if err := rows.Scan(&indexName, &columnName, &isUnique); err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[indexName] = &domain.Index{
				Name:     indexName,
				Columns:  []string{columnName},
				IsUnique: isUnique,
			}
		}
	}

	var indexes []domain.Index
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, nil
}

// getTableConstraints gets all constraints for a table.
func (i *DatabaseIntrospector) getTableConstraints(ctx context.Context, tableName string) ([]domain.Constraint, error) {
	// For now, return empty constraints
	// Full implementation would query foreign keys, check constraints, etc.
	return []domain.Constraint{}, nil
}

// Ensure DatabaseIntrospector implements Introspector interface.
var _ domain.Introspector = (*DatabaseIntrospector)(nil)
