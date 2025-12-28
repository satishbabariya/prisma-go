// Package mysql implements MySQL database introspection.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/domain"
)

// Introspector implements domain.Introspector for MySQL.
type Introspector struct{}

// NewIntrospector creates a new MySQL introspector.
func NewIntrospector() *Introspector {
	return &Introspector{}
}

// IntrospectDatabase introspects the entire MySQL database.
func (i *Introspector) IntrospectDatabase(ctx context.Context, db *sql.DB, schemas []string) (*domain.IntrospectedDatabase, error) {
	if len(schemas) == 0 {
		// Get current database
		var currentDB string
		if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&currentDB); err != nil {
			return nil, err
		}
		schemas = []string{currentDB}
	}

	result := &domain.IntrospectedDatabase{
		Tables: []domain.IntrospectedTable{},
		Enums:  []domain.IntrospectedEnum{}, // MySQL uses ENUM column type, not separate types
	}

	for _, schema := range schemas {
		tables, err := i.introspectTables(ctx, db, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect tables in schema %s: %w", schema, err)
		}
		result.Tables = append(result.Tables, tables...)
	}

	return result, nil
}

// introspectTables gets all tables in a schema.
func (i *Introspector) introspectTables(ctx context.Context, db *sql.DB, schema string) ([]domain.IntrospectedTable, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.QueryContext(ctx, query, schema)
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
		table.Schema = schema
		tables = append(tables, *table)
	}

	return tables, rows.Err()
}

// IntrospectTable introspects a single table.
func (i *Introspector) IntrospectTable(ctx context.Context, db *sql.DB, tableName string) (*domain.IntrospectedTable, error) {
	// Get current database
	var schema string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&schema); err != nil {
		return nil, err
	}

	table := &domain.IntrospectedTable{
		Name:        tableName,
		Columns:     []domain.IntrospectedColumn{},
		Indexes:     []domain.IntrospectedIndex{},
		ForeignKeys: []domain.IntrospectedForeignKey{},
	}

	// Get columns
	columns, err := i.introspectColumns(ctx, db, schema, tableName)
	if err != nil {
		return nil, err
	}
	table.Columns = columns

	// Get primary key
	pk, err := i.introspectPrimaryKey(ctx, db, schema, tableName)
	if err != nil {
		return nil, err
	}
	table.PrimaryKey = pk

	// Get indexes
	indexes, err := i.introspectIndexes(ctx, db, schema, tableName)
	if err != nil {
		return nil, err
	}
	table.Indexes = indexes

	// Get foreign keys
	fks, err := i.introspectForeignKeys(ctx, db, schema, tableName)
	if err != nil {
		return nil, err
	}
	table.ForeignKeys = fks

	return table, nil
}

// introspectColumns gets all columns for a table.
func (i *Introspector) introspectColumns(ctx context.Context, db *sql.DB, schema, tableName string) ([]domain.IntrospectedColumn, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			extra,
			ordinal_position
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []domain.IntrospectedColumn
	for rows.Next() {
		var col domain.IntrospectedColumn
		var isNullable string
		var defaultVal, extra sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &isNullable, &defaultVal, &extra, &col.OrdinalPosition); err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}

		// Check if auto-increment
		if extra.Valid && strings.Contains(extra.String, "auto_increment") {
			col.IsAutoIncrement = true
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// introspectPrimaryKey gets the primary key for a table.
func (i *Introspector) introspectPrimaryKey(ctx context.Context, db *sql.DB, schema, tableName string) (*domain.IntrospectedPrimaryKey, error) {
	query := `
		SELECT constraint_name, column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ? AND table_name = ?
		  AND constraint_name = 'PRIMARY'
		ORDER BY ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkName string
	var columns []string

	for rows.Next() {
		var colName string
		if err := rows.Scan(&pkName, &colName); err != nil {
			return nil, err
		}
		columns = append(columns, colName)
	}

	if len(columns) == 0 {
		return nil, nil
	}

	return &domain.IntrospectedPrimaryKey{
		Name:    pkName,
		Columns: columns,
	}, rows.Err()
}

// introspectIndexes gets all indexes for a table.
func (i *Introspector) introspectIndexes(ctx context.Context, db *sql.DB, schema, tableName string) ([]domain.IntrospectedIndex, error) {
	query := `
		SELECT
			index_name,
			column_name,
			non_unique,
			index_type
		FROM information_schema.statistics
		WHERE table_schema = ? AND table_name = ?
		  AND index_name != 'PRIMARY'
		ORDER BY index_name, seq_in_index
	`

	rows, err := db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*domain.IntrospectedIndex)

	for rows.Next() {
		var indexName, column, indexType string
		var nonUnique int

		if err := rows.Scan(&indexName, &column, &nonUnique, &indexType); err != nil {
			return nil, err
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, column)
		} else {
			indexMap[indexName] = &domain.IntrospectedIndex{
				Name:     indexName,
				Columns:  []string{column},
				IsUnique: nonUnique == 0,
				Type:     strings.ToLower(indexType),
			}
		}
	}

	var indexes []domain.IntrospectedIndex
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, rows.Err()
}

// introspectForeignKeys gets all foreign keys for a table.
func (i *Introspector) introspectForeignKeys(ctx context.Context, db *sql.DB, schema, tableName string) ([]domain.IntrospectedForeignKey, error) {
	query := `
		SELECT
			kcu.constraint_name,
			kcu.column_name,
			kcu.referenced_table_name,
			kcu.referenced_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc
		  ON kcu.constraint_name = rc.constraint_name
		  AND kcu.table_schema = rc.constraint_schema
		WHERE kcu.table_schema = ? AND kcu.table_name = ?
		  AND kcu.referenced_table_name IS NOT NULL
		ORDER BY kcu.constraint_name, kcu.ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]*domain.IntrospectedForeignKey)

	for rows.Next() {
		var constraintName, column, refTable, refColumn, deleteRule, updateRule string

		if err := rows.Scan(&constraintName, &column, &refTable, &refColumn, &deleteRule, &updateRule); err != nil {
			return nil, err
		}

		if fk, exists := fkMap[constraintName]; exists {
			fk.ColumnNames = append(fk.ColumnNames, column)
			fk.ReferencedColumns = append(fk.ReferencedColumns, refColumn)
		} else {
			fkMap[constraintName] = &domain.IntrospectedForeignKey{
				Name:              constraintName,
				ColumnNames:       []string{column},
				ReferencedTable:   refTable,
				ReferencedColumns: []string{refColumn},
				OnDelete:          mapReferentialAction(deleteRule),
				OnUpdate:          mapReferentialAction(updateRule),
			}
		}
	}

	var fks []domain.IntrospectedForeignKey
	for _, fk := range fkMap {
		fks = append(fks, *fk)
	}

	return fks, rows.Err()
}

// GetDatabaseVersion returns the MySQL version.
func (i *Introspector) GetDatabaseVersion(ctx context.Context, db *sql.DB) (string, error) {
	var version string
	err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	return version, err
}

// GetSchemas returns all available schemas/databases.
func (i *Introspector) GetSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY schema_name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}

	return schemas, rows.Err()
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
