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
	// Query SQL Server information_schema for tables
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
	tableMap := make(map[string]*Table)

	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			return nil, err
		}

		fullName := fmt.Sprintf("%s.%s", schemaName, tableName)
		table := &Table{
			Name:        tableName,
			Schema:      schemaName,
			Columns:     []Column{},
			Indexes:     []Index{},
			ForeignKeys: []ForeignKey{},
		}
		tables = append(tables, *table)
		tableMap[fullName] = table
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Introspect columns for each table
	for idx := range tables {
		table := &tables[idx]
		columns, err := i.introspectColumns(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect columns for %s.%s: %w", table.Schema, table.Name, err)
		}
		table.Columns = columns

		// Introspect primary keys
		pkColumns, err := i.introspectPrimaryKeys(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect primary keys for %s.%s: %w", table.Schema, table.Name, err)
		}
		if len(pkColumns) > 0 {
			table.PrimaryKey = &PrimaryKey{
				Name:    fmt.Sprintf("PK_%s", table.Name),
				Columns: pkColumns,
			}
		}

		// Introspect indexes
		indexes, err := i.introspectIndexes(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect indexes for %s.%s: %w", table.Schema, table.Name, err)
		}
		table.Indexes = indexes

		// Introspect foreign keys
		fks, err := i.introspectForeignKeys(ctx, table.Schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect foreign keys for %s.%s: %w", table.Schema, table.Name, err)
		}
		table.ForeignKeys = fks
	}

	return tables, nil
}

// introspectColumns introspects columns from SQL Server
func (i *SQLServerIntrospector) introspectColumns(ctx context.Context, schemaName, tableName string) ([]Column, error) {
	query := `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			ORDINAL_POSITION
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = @p1 AND TABLE_NAME = @p2
		ORDER BY ORDINAL_POSITION
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var colName, dataType, isNullable string
		var charMaxLen, numericPrecision, numericScale, ordinalPos sql.NullInt64
		var colDefault sql.NullString

		if err := rows.Scan(&colName, &dataType, &charMaxLen, &numericPrecision, &numericScale, &isNullable, &colDefault, &ordinalPos); err != nil {
			return nil, err
		}

		var defaultValue *string
		if colDefault.Valid {
			defaultValue = &colDefault.String
		}
		col := Column{
			Name:         colName,
			Type:         mapSQLServerType(dataType, charMaxLen, numericPrecision, numericScale),
			Nullable:     isNullable == "YES",
			DefaultValue: defaultValue,
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// introspectPrimaryKeys introspects primary keys from SQL Server
func (i *SQLServerIntrospector) introspectPrimaryKeys(ctx context.Context, schemaName, tableName string) ([]string, error) {
	query := `
		SELECT c.COLUMN_NAME
		FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
		INNER JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE c
			ON tc.CONSTRAINT_NAME = c.CONSTRAINT_NAME
			AND tc.TABLE_SCHEMA = c.TABLE_SCHEMA
			AND tc.TABLE_NAME = c.TABLE_NAME
		WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
			AND tc.TABLE_SCHEMA = @p1
			AND tc.TABLE_NAME = @p2
		ORDER BY c.ORDINAL_POSITION
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkColumns []string
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			return nil, err
		}
		pkColumns = append(pkColumns, colName)
	}

	return pkColumns, rows.Err()
}

// introspectIndexes introspects indexes from SQL Server
func (i *SQLServerIntrospector) introspectIndexes(ctx context.Context, schemaName, tableName string) ([]Index, error) {
	query := `
		SELECT 
			i.name AS INDEX_NAME,
			i.is_unique AS IS_UNIQUE,
			i.is_primary_key AS IS_PRIMARY,
			c.name AS COLUMN_NAME,
			ic.key_ordinal AS ORDINAL_POSITION
		FROM sys.indexes i
		INNER JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		INNER JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
		INNER JOIN sys.tables t ON i.object_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE s.name = @p1 AND t.name = @p2
			AND i.type > 0 -- Exclude heap
		ORDER BY i.name, ic.key_ordinal
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*Index)
	for rows.Next() {
		var idxName, colName string
		var isUnique, isPrimary bool
		var ordinalPos int

		if err := rows.Scan(&idxName, &isUnique, &isPrimary, &colName, &ordinalPos); err != nil {
			return nil, err
		}

		if _, ok := indexMap[idxName]; !ok {
			indexMap[idxName] = &Index{
				Name:     idxName,
				IsUnique: isUnique,
				Columns:  []string{},
			}
		}
		indexMap[idxName].Columns = append(indexMap[idxName].Columns, colName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var indexes []Index
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, nil
}

// introspectForeignKeys introspects foreign keys from SQL Server
func (i *SQLServerIntrospector) introspectForeignKeys(ctx context.Context, schemaName, tableName string) ([]ForeignKey, error) {
	query := `
		SELECT 
			fk.name AS FK_NAME,
			OBJECT_SCHEMA_NAME(fk.parent_object_id) AS PARENT_SCHEMA,
			OBJECT_NAME(fk.parent_object_id) AS PARENT_TABLE,
			cp.name AS PARENT_COLUMN,
			OBJECT_SCHEMA_NAME(fk.referenced_object_id) AS REFERENCED_SCHEMA,
			OBJECT_NAME(fk.referenced_object_id) AS REFERENCED_TABLE,
			cr.name AS REFERENCED_COLUMN
		FROM sys.foreign_keys fk
		INNER JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
		INNER JOIN sys.columns cp ON fkc.parent_object_id = cp.object_id AND fkc.parent_column_id = cp.column_id
		INNER JOIN sys.columns cr ON fkc.referenced_object_id = cr.object_id AND fkc.referenced_column_id = cr.column_id
		WHERE OBJECT_SCHEMA_NAME(fk.parent_object_id) = @p1
			AND OBJECT_NAME(fk.parent_object_id) = @p2
		ORDER BY fk.name, fkc.constraint_column_id
	`

	rows, err := i.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]*ForeignKey)
	for rows.Next() {
		var fkName, parentSchema, parentTable, parentCol, refSchema, refTable, refCol string
		if err := rows.Scan(&fkName, &parentSchema, &parentTable, &parentCol, &refSchema, &refTable, &refCol); err != nil {
			return nil, err
		}

		if _, ok := fkMap[fkName]; !ok {
			fkMap[fkName] = &ForeignKey{
				Name:              fkName,
				ReferencedTable:   refTable,
				Columns:           []string{},
				ReferencedColumns: []string{},
			}
		}
		fkMap[fkName].Columns = append(fkMap[fkName].Columns, parentCol)
		fkMap[fkName].ReferencedColumns = append(fkMap[fkName].ReferencedColumns, refCol)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var fks []ForeignKey
	for _, fk := range fkMap {
		fks = append(fks, *fk)
	}

	return fks, nil
}

// mapSQLServerType maps SQL Server data types to Prisma types
func mapSQLServerType(dataType string, charMaxLen, numericPrecision, numericScale sql.NullInt64) string {
	switch dataType {
	case "int", "bigint", "smallint", "tinyint":
		return "Int"
	case "bit":
		return "Boolean"
	case "decimal", "numeric", "money", "smallmoney":
		return "Decimal"
	case "float", "real":
		return "Float"
	case "char", "varchar", "nchar", "nvarchar", "text", "ntext":
		return "String"
	case "date", "time", "datetime", "datetime2", "datetimeoffset", "smalldatetime":
		return "DateTime"
	case "binary", "varbinary", "image":
		return "Bytes"
	case "uniqueidentifier":
		return "String" // UUID as string
	case "xml":
		return "String" // XML as string
	case "json":
		return "Json"
	default:
		return "String" // Default fallback
	}
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
