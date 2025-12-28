// Package postgresql implements PostgreSQL database introspection.
package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/domain"
)

// Introspector implements domain.Introspector for PostgreSQL.
type Introspector struct{}

// NewIntrospector creates a new PostgreSQL introspector.
func NewIntrospector() *Introspector {
	return &Introspector{}
}

// IntrospectDatabase introspects the entire PostgreSQL database.
func (i *Introspector) IntrospectDatabase(ctx context.Context, db *sql.DB, schemas []string) (*domain.IntrospectedDatabase, error) {
	if len(schemas) == 0 {
		schemas = []string{"public"} // Default schema
	}

	result := &domain.IntrospectedDatabase{
		Tables:    []domain.IntrospectedTable{},
		Enums:     []domain.IntrospectedEnum{},
		Sequences: []domain.IntrospectedSequence{},
	}

	// Introspect tables
	for _, schema := range schemas {
		tables, err := i.introspectTables(ctx, db, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect tables in schema %s: %w", schema, err)
		}
		result.Tables = append(result.Tables, tables...)

		// Introspect enums
		enums, err := i.introspectEnums(ctx, db, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect enums in schema %s: %w", schema, err)
		}
		result.Enums = append(result.Enums, enums...)
	}

	return result, nil
}

// introspectTables gets all tables in a schema.
func (i *Introspector) introspectTables(ctx context.Context, db *sql.DB, schema string) ([]domain.IntrospectedTable, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
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
	table := &domain.IntrospectedTable{
		Name:        tableName,
		Columns:     []domain.IntrospectedColumn{},
		Indexes:     []domain.IntrospectedIndex{},
		ForeignKeys: []domain.IntrospectedForeignKey{},
	}

	// Get columns
	columns, err := i.introspectColumns(ctx, db, tableName)
	if err != nil {
		return nil, err
	}
	table.Columns = columns

	// Get primary key
	pk, err := i.introspectPrimaryKey(ctx, db, tableName)
	if err != nil {
		return nil, err
	}
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

// introspectColumns gets all columns for a table.
func (i *Introspector) introspectColumns(ctx context.Context, db *sql.DB, tableName string) ([]domain.IntrospectedColumn, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			ordinal_position
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []domain.IntrospectedColumn
	for rows.Next() {
		var col domain.IntrospectedColumn
		var isNullable string
		var defaultVal sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &isNullable, &defaultVal, &col.OrdinalPosition); err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}

		// Check if auto-increment (SERIAL, BIGSERIAL)
		if defaultVal.Valid && (col.Type == "integer" || col.Type == "bigint") {
			if containsString(defaultVal.String, "nextval") {
				col.IsAutoIncrement = true
			}
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// introspectPrimaryKey gets the primary key for a table.
func (i *Introspector) introspectPrimaryKey(ctx context.Context, db *sql.DB, tableName string) (*domain.IntrospectedPrimaryKey, error) {
	query := `
		SELECT constraint_name, column_name
		FROM information_schema.key_column_usage
		WHERE table_name = $1
		  AND constraint_name IN (
			  SELECT constraint_name
			  FROM information_schema.table_constraints
			  WHERE table_name = $1
				AND constraint_type = 'PRIMARY KEY'
		  )
		ORDER BY ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, tableName)
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
		return nil, nil // No primary key
	}

	return &domain.IntrospectedPrimaryKey{
		Name:    pkName,
		Columns: columns,
	}, rows.Err()
}

// introspectIndexes gets all indexes for a table.
func (i *Introspector) introspectIndexes(ctx context.Context, db *sql.DB, tableName string) ([]domain.IntrospectedIndex, error) {
	query := `
		SELECT
			i.indexname,
			a.attname,
			ix.indisunique
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.tablename
		JOIN pg_index ix ON ix.indexrelid = (i.schemaname || '.' || i.indexname)::regclass
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(ix.indkey)
		WHERE i.tablename = $1
		  AND i.indexname NOT IN (
			  SELECT constraint_name
			  FROM information_schema.table_constraints
			  WHERE table_name = $1
				AND constraint_type = 'PRIMARY KEY'
		  )
		ORDER BY i.indexname, a.attnum
	`

	rows, err := db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*domain.IntrospectedIndex)

	for rows.Next() {
		var indexName, column string
		var isUnique bool

		if err := rows.Scan(&indexName, &column, &isUnique); err != nil {
			return nil, err
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, column)
		} else {
			indexMap[indexName] = &domain.IntrospectedIndex{
				Name:     indexName,
				Columns:  []string{column},
				IsUnique: isUnique,
				Type:     "btree", // Default for PostgreSQL
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
func (i *Introspector) introspectForeignKeys(ctx context.Context, db *sql.DB, tableName string) ([]domain.IntrospectedForeignKey, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
		  ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
		  ON ccu.constraint_name = tc.constraint_name
		JOIN information_schema.referential_constraints AS rc
		  ON rc.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_name = $1
		ORDER BY tc.constraint_name, kcu.ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, tableName)
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

// introspectEnums gets all enum types in a schema.
func (i *Introspector) introspectEnums(ctx context.Context, db *sql.DB, schema string) ([]domain.IntrospectedEnum, error) {
	query := `
		SELECT t.typname AS enum_name, e.enumlabel AS enum_value
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = $1
		ORDER BY t.typname, e.enumsortorder
	`

	rows, err := db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	enumMap := make(map[string]*domain.IntrospectedEnum)

	for rows.Next() {
		var enumName, enumValue string
		if err := rows.Scan(&enumName, &enumValue); err != nil {
			return nil, err
		}

		if enum, exists := enumMap[enumName]; exists {
			enum.Values = append(enum.Values, enumValue)
		} else {
			enumMap[enumName] = &domain.IntrospectedEnum{
				Name:   enumName,
				Values: []string{enumValue},
			}
		}
	}

	var enums []domain.IntrospectedEnum
	for _, enum := range enumMap {
		enums = append(enums, *enum)
	}

	return enums, rows.Err()
}

// GetDatabaseVersion returns the PostgreSQL version.
func (i *Introspector) GetDatabaseVersion(ctx context.Context, db *sql.DB) (string, error) {
	var version string
	err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	return version, err
}

// GetSchemas returns all available schemas.
func (i *Introspector) GetSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
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

// Helper functions

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}

func mapReferentialAction(action string) domain.ReferentialAction {
	switch action {
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
