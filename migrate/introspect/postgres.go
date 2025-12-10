// Package introspect provides PostgreSQL database introspection.
package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// PostgresIntrospector implements introspection for PostgreSQL
type PostgresIntrospector struct {
	db *sql.DB
}

// Introspect reads the PostgreSQL database schema
func (i *PostgresIntrospector) Introspect(ctx context.Context) (*DatabaseSchema, error) {
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

	// Introspect enums
	enums, err := i.introspectEnums(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect enums: %w", err)
	}
	schema.Enums = enums

	// Introspect sequences
	sequences, err := i.introspectSequences(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect sequences: %w", err)
	}
	schema.Sequences = sequences

	// Introspect views
	views, err := i.introspectViews(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect views: %w", err)
	}
	schema.Views = views

	return schema, nil
}

// introspectViews reads all views
func (i *PostgresIntrospector) introspectViews(ctx context.Context) ([]View, error) {
	query := `
		SELECT 
			table_schema,
			table_name,
			view_definition
		FROM information_schema.views
		WHERE table_schema = 'public'
		ORDER BY table_name
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	var views []View
	for rows.Next() {
		var view View
		var schema string
		err := rows.Scan(&schema, &view.Name, &view.Definition)
		if err != nil {
			return nil, fmt.Errorf("failed to scan view: %w", err)
		}
		views = append(views, view)
	}

	return views, rows.Err()
}

// introspectTables reads all tables and their columns
func (i *PostgresIntrospector) introspectTables(ctx context.Context) ([]Table, error) {
	// Query to get all tables in public schema
	query := `
		SELECT 
			table_schema,
			table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Schema, &table.Name); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		// Get columns for this table
		columns, err := i.introspectColumns(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect columns for %s: %w", table.Name, err)
		}
		table.Columns = columns

		// Get primary key
		pk, err := i.introspectPrimaryKey(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect primary key for %s: %w", table.Name, err)
		}
		table.PrimaryKey = pk

		// Get indexes
		indexes, err := i.introspectIndexes(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect indexes for %s: %w", table.Name, err)
		}
		table.Indexes = indexes

		// Get foreign keys
		fks, err := i.introspectForeignKeys(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect foreign keys for %s: %w", table.Name, err)
		}
		table.ForeignKeys = fks

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// introspectColumns reads all columns for a table
func (i *PostgresIntrospector) introspectColumns(ctx context.Context, schema, tableName string) ([]Column, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			udt_name,
			is_nullable,
			column_default,
			character_maximum_length,
			numeric_precision,
			numeric_scale
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := i.db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		var dataType, udtName string
		var isNullable string
		var defaultValue sql.NullString
		var maxLength, numPrecision, numScale sql.NullInt64

		err := rows.Scan(
			&col.Name,
			&dataType,
			&udtName,
			&isNullable,
			&defaultValue,
			&maxLength,
			&numPrecision,
			&numScale,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		// Map PostgreSQL type to generic type
		col.Type = mapPostgresType(dataType, udtName, maxLength.Int64, numPrecision.Int64, numScale.Int64)
		col.Nullable = (isNullable == "YES")

		// Parse default value
		if defaultValue.Valid && defaultValue.String != "" {
			col.DefaultValue = &defaultValue.String
		}

		// Check for auto-increment (SERIAL, BIGSERIAL, or sequences)
		col.AutoIncrement = isAutoIncrement(defaultValue.String, col.Type)

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// introspectPrimaryKey reads the primary key for a table
func (i *PostgresIntrospector) introspectPrimaryKey(ctx context.Context, schema, tableName string) (*PrimaryKey, error) {
	query := `
		SELECT 
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = $1
		  AND tc.table_name = $2
		GROUP BY tc.constraint_name
	`

	var pk PrimaryKey
	var columnsArray string

	err := i.db.QueryRowContext(ctx, query, schema, tableName).Scan(&pk.Name, &columnsArray)
	if err == sql.ErrNoRows {
		return nil, nil // No primary key
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query primary key: %w", err)
	}

	// Parse array of columns (PostgreSQL returns as {col1,col2})
	columnsArray = strings.Trim(columnsArray, "{}")
	pk.Columns = strings.Split(columnsArray, ",")

	return &pk, nil
}

// introspectIndexes reads all indexes for a table
func (i *PostgresIntrospector) introspectIndexes(ctx context.Context, schema, tableName string) ([]Index, error) {
	query := `
		SELECT 
			i.relname as index_name,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) as columns,
			ix.indisunique as is_unique
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1
		  AND t.relname = $2
		  AND NOT ix.indisprimary
		GROUP BY i.relname, ix.indisunique
		ORDER BY i.relname
	`

	rows, err := i.db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var idx Index
		var columnsArray string

		err := rows.Scan(&idx.Name, &columnsArray, &idx.IsUnique)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		// Parse array of columns
		columnsArray = strings.Trim(columnsArray, "{}")
		idx.Columns = strings.Split(columnsArray, ",")

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// introspectForeignKeys reads all foreign keys for a table
func (i *PostgresIntrospector) introspectForeignKeys(ctx context.Context, schema, tableName string) ([]ForeignKey, error) {
	query := `
		SELECT 
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position) as columns,
			ccu.table_name as referenced_table,
			array_agg(ccu.column_name ORDER BY kcu.ordinal_position) as referenced_columns,
			rc.update_rule as on_update,
			rc.delete_rule as on_delete
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = $1
		  AND tc.table_name = $2
		GROUP BY tc.constraint_name, ccu.table_name, rc.update_rule, rc.delete_rule
		ORDER BY tc.constraint_name
	`

	rows, err := i.db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		var columnsArray, refColumnsArray string

		err := rows.Scan(
			&fk.Name,
			&columnsArray,
			&fk.ReferencedTable,
			&refColumnsArray,
			&fk.OnUpdate,
			&fk.OnDelete,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		// Parse arrays
		columnsArray = strings.Trim(columnsArray, "{}")
		fk.Columns = strings.Split(columnsArray, ",")

		refColumnsArray = strings.Trim(refColumnsArray, "{}")
		fk.ReferencedColumns = strings.Split(refColumnsArray, ",")

		fks = append(fks, fk)
	}

	return fks, rows.Err()
}

// introspectEnums reads all enum types
func (i *PostgresIntrospector) introspectEnums(ctx context.Context) ([]Enum, error) {
	query := `
		SELECT 
			t.typname as enum_name,
			array_agg(e.enumlabel ORDER BY e.enumsortorder) as enum_values
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = 'public'
		GROUP BY t.typname
		ORDER BY t.typname
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enums: %w", err)
	}
	defer rows.Close()

	var enums []Enum
	for rows.Next() {
		var enum Enum
		var valuesArray string

		err := rows.Scan(&enum.Name, &valuesArray)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enum: %w", err)
		}

		// Parse array
		valuesArray = strings.Trim(valuesArray, "{}")
		enum.Values = strings.Split(valuesArray, ",")

		enums = append(enums, enum)
	}

	return enums, rows.Err()
}

// introspectSequences reads all sequences
func (i *PostgresIntrospector) introspectSequences(ctx context.Context) ([]Sequence, error) {
	query := `
		SELECT sequence_name
		FROM information_schema.sequences
		WHERE sequence_schema = 'public'
		ORDER BY sequence_name
	`

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sequences: %w", err)
	}
	defer rows.Close()

	var sequences []Sequence
	for rows.Next() {
		var seq Sequence
		if err := rows.Scan(&seq.Name); err != nil {
			return nil, fmt.Errorf("failed to scan sequence: %w", err)
		}
		sequences = append(sequences, seq)
	}

	return sequences, rows.Err()
}

// mapPostgresType maps PostgreSQL data types to generic types
func mapPostgresType(dataType, udtName string, maxLength, precision, scale int64) string {
	switch dataType {
	case "integer", "int", "int4":
		return "INTEGER"
	case "bigint", "int8":
		return "BIGINT"
	case "smallint", "int2":
		return "SMALLINT"
	case "serial":
		return "SERIAL"
	case "bigserial":
		return "BIGSERIAL"
	case "boolean", "bool":
		return "BOOLEAN"
	case "character varying", "varchar":
		if maxLength > 0 {
			return fmt.Sprintf("VARCHAR(%d)", maxLength)
		}
		return "VARCHAR"
	case "character", "char":
		if maxLength > 0 {
			return fmt.Sprintf("CHAR(%d)", maxLength)
		}
		return "CHAR"
	case "text":
		return "TEXT"
	case "numeric", "decimal":
		if precision > 0 && scale > 0 {
			return fmt.Sprintf("DECIMAL(%d,%d)", precision, scale)
		}
		return "DECIMAL"
	case "real", "float4":
		return "REAL"
	case "double precision", "float8":
		return "DOUBLE PRECISION"
	case "timestamp without time zone", "timestamp":
		return "TIMESTAMP"
	case "timestamp with time zone", "timestamptz":
		return "TIMESTAMPTZ"
	case "date":
		return "DATE"
	case "time without time zone", "time":
		return "TIME"
	case "json":
		return "JSON"
	case "jsonb":
		return "JSONB"
	case "uuid":
		return "UUID"
	case "bytea":
		return "BYTEA"
	case "USER-DEFINED":
		// This is likely an enum
		return udtName
	default:
		return dataType
	}
}

// isAutoIncrement checks if a column has auto-increment
func isAutoIncrement(defaultValue, dataType string) bool {
	if defaultValue == "" {
		return false
	}

	// Check for SERIAL types
	if strings.Contains(dataType, "SERIAL") {
		return true
	}

	// Check for nextval() function (sequence)
	if strings.Contains(strings.ToLower(defaultValue), "nextval") {
		return true
	}

	return false
}
