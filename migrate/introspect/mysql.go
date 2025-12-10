// Package introspect provides MySQL database introspection.
package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// MySQLIntrospector implements introspection for MySQL
type MySQLIntrospector struct {
	db *sql.DB
}

// Introspect reads the MySQL database schema
func (i *MySQLIntrospector) Introspect(ctx context.Context) (*DatabaseSchema, error) {
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
func (i *MySQLIntrospector) introspectTables(ctx context.Context) ([]Table, error) {
	// Get current database name
	var dbName string
	err := i.db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}

	// Query to get all tables
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := i.db.QueryContext(ctx, query, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		table.Schema = dbName
		if err := rows.Scan(&table.Name); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		// Get columns
		columns, err := i.introspectColumns(ctx, dbName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect columns for %s: %w", table.Name, err)
		}
		table.Columns = columns

		// Get primary key
		pk, err := i.introspectPrimaryKey(ctx, dbName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect primary key for %s: %w", table.Name, err)
		}
		table.PrimaryKey = pk

		// Get indexes
		indexes, err := i.introspectIndexes(ctx, dbName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect indexes for %s: %w", table.Name, err)
		}
		table.Indexes = indexes

		// Get foreign keys
		fks, err := i.introspectForeignKeys(ctx, dbName, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect foreign keys for %s: %w", table.Name, err)
		}
		table.ForeignKeys = fks

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// introspectColumns reads all columns for a table
func (i *MySQLIntrospector) introspectColumns(ctx context.Context, schema, tableName string) ([]Column, error) {
	query := `
		SELECT 
			column_name,
			column_type,
			is_nullable,
			column_default,
			extra
		FROM information_schema.columns
		WHERE table_schema = ?
		  AND table_name = ?
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
		var columnType string
		var isNullable string
		var defaultValue sql.NullString
		var extra string

		err := rows.Scan(
			&col.Name,
			&columnType,
			&isNullable,
			&defaultValue,
			&extra,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Type = i.mapMySQLType(columnType)
		col.Nullable = (isNullable == "YES")

		if defaultValue.Valid && defaultValue.String != "" {
			col.DefaultValue = &defaultValue.String
		}

		// Check for auto-increment
		col.AutoIncrement = strings.Contains(strings.ToLower(extra), "auto_increment")

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// introspectPrimaryKey reads the primary key for a table
func (i *MySQLIntrospector) introspectPrimaryKey(ctx context.Context, schema, tableName string) (*PrimaryKey, error) {
	query := `
		SELECT 
			constraint_name,
			GROUP_CONCAT(column_name ORDER BY ordinal_position) as columns
		FROM information_schema.key_column_usage
		WHERE table_schema = ?
		  AND table_name = ?
		  AND constraint_name = 'PRIMARY'
		GROUP BY constraint_name
	`

	var pk PrimaryKey
	var columnsStr string

	err := i.db.QueryRowContext(ctx, query, schema, tableName).Scan(&pk.Name, &columnsStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query primary key: %w", err)
	}

	pk.Columns = strings.Split(columnsStr, ",")
	return &pk, nil
}

// introspectIndexes reads all indexes for a table
func (i *MySQLIntrospector) introspectIndexes(ctx context.Context, schema, tableName string) ([]Index, error) {
	query := `
		SELECT 
			index_name,
			GROUP_CONCAT(column_name ORDER BY seq_in_index) as columns,
			MAX(non_unique) as is_non_unique
		FROM information_schema.statistics
		WHERE table_schema = ?
		  AND table_name = ?
		  AND index_name != 'PRIMARY'
		GROUP BY index_name
		ORDER BY index_name
	`

	rows, err := i.db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var idx Index
		var columnsStr string
		var isNonUnique int

		err := rows.Scan(&idx.Name, &columnsStr, &isNonUnique)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		idx.Columns = strings.Split(columnsStr, ",")
		idx.IsUnique = (isNonUnique == 0)

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// introspectForeignKeys reads all foreign keys for a table
func (i *MySQLIntrospector) introspectForeignKeys(ctx context.Context, schema, tableName string) ([]ForeignKey, error) {
	query := `
		SELECT 
			kcu.constraint_name,
			GROUP_CONCAT(kcu.column_name ORDER BY kcu.ordinal_position) as columns,
			kcu.referenced_table_name,
			GROUP_CONCAT(kcu.referenced_column_name ORDER BY kcu.ordinal_position) as referenced_columns,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc
			ON kcu.constraint_name = rc.constraint_name
			AND kcu.constraint_schema = rc.constraint_schema
		WHERE kcu.table_schema = ?
		  AND kcu.table_name = ?
		  AND kcu.referenced_table_name IS NOT NULL
		GROUP BY kcu.constraint_name, kcu.referenced_table_name, rc.update_rule, rc.delete_rule
		ORDER BY kcu.constraint_name
	`

	rows, err := i.db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		var columnsStr, refColumnsStr string

		err := rows.Scan(
			&fk.Name,
			&columnsStr,
			&fk.ReferencedTable,
			&refColumnsStr,
			&fk.OnUpdate,
			&fk.OnDelete,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		fk.Columns = strings.Split(columnsStr, ",")
		fk.ReferencedColumns = strings.Split(refColumnsStr, ",")

		fks = append(fks, fk)
	}

	return fks, rows.Err()
}

// mapMySQLType maps MySQL data types to generic types
func (i *MySQLIntrospector) mapMySQLType(mysqlType string) string {
	lowerType := strings.ToLower(mysqlType)

	switch {
	case strings.HasPrefix(lowerType, "int("):
		return "INT"
	case lowerType == "int":
		return "INT"
	case strings.HasPrefix(lowerType, "bigint"):
		return "BIGINT"
	case strings.HasPrefix(lowerType, "smallint"):
		return "SMALLINT"
	case strings.HasPrefix(lowerType, "tinyint(1)"):
		return "BOOLEAN"
	case strings.HasPrefix(lowerType, "tinyint"):
		return "TINYINT"
	case strings.HasPrefix(lowerType, "varchar"):
		return mysqlType
	case strings.HasPrefix(lowerType, "char"):
		return mysqlType
	case lowerType == "text":
		return "TEXT"
	case strings.HasPrefix(lowerType, "decimal"):
		return mysqlType
	case strings.HasPrefix(lowerType, "float"):
		return "FLOAT"
	case strings.HasPrefix(lowerType, "double"):
		return "DOUBLE"
	case strings.HasPrefix(lowerType, "timestamp"):
		return "TIMESTAMP"
	case lowerType == "datetime":
		return "DATETIME"
	case lowerType == "date":
		return "DATE"
	case lowerType == "time":
		return "TIME"
	case lowerType == "json":
		return "JSON"
	case strings.HasPrefix(lowerType, "blob"):
		return "BLOB"
	case strings.HasPrefix(lowerType, "enum"):
		return mysqlType
	default:
		return mysqlType
	}
}

