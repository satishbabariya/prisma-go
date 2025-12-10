// Package executor executes queries and maps results to structs.
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// Executor executes queries and maps results
type Executor struct {
	db        *sql.DB
	provider  string
	generator sqlgen.Generator
	stmtCache map[string]*sql.Stmt
	cacheMu   sync.RWMutex
}

// NewExecutor creates a new query executor
func NewExecutor(db *sql.DB, provider string) *Executor {
	return &Executor{
		db:        db,
		provider:  provider,
		generator: sqlgen.NewGenerator(provider),
		stmtCache: make(map[string]*sql.Stmt),
	}
}

// getCachedStmt gets a cached prepared statement or creates a new one
func (e *Executor) getCachedStmt(ctx context.Context, query string) (*sql.Stmt, error) {
	e.cacheMu.RLock()
	stmt, ok := e.stmtCache[query]
	e.cacheMu.RUnlock()

	if ok && stmt != nil {
		return stmt, nil
	}

	// Create new prepared statement
	stmt, err := e.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}

	// Cache it
	e.cacheMu.Lock()
	e.stmtCache[query] = stmt
	e.cacheMu.Unlock()

	return stmt, nil
}

// ClearStmtCache clears the prepared statement cache
func (e *Executor) ClearStmtCache() {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	for _, stmt := range e.stmtCache {
		stmt.Close()
	}
	e.stmtCache = make(map[string]*sql.Stmt)
}

// FindMany executes a SELECT query and maps results to a slice
func (e *Executor) FindMany(ctx context.Context, table string, selectFields map[string]bool, where *sqlgen.WhereClause, orderBy []sqlgen.OrderBy, limit, offset *int, include map[string]bool, dest interface{}) error {
	return e.FindManyWithRelations(ctx, table, selectFields, where, orderBy, limit, offset, include, nil, dest)
}

// FindManyWithRelations executes a SELECT query with relations and maps results to a slice
func (e *Executor) FindManyWithRelations(ctx context.Context, table string, selectFields map[string]bool, where *sqlgen.WhereClause, orderBy []sqlgen.OrderBy, limit, offset *int, include map[string]bool, relations map[string]RelationMetadata, dest interface{}) error {
	// Convert selectFields map to slice
	var columns []string
	if selectFields != nil && len(selectFields) > 0 {
		for field := range selectFields {
			columns = append(columns, field)
		}
	}

	var query *sqlgen.Query

	// Build JOINs if relations are included
	var joins []sqlgen.Join
	if include != nil && len(include) > 0 && relations != nil {
		joins = buildJoinsFromIncludes(table, include, relations)
	}

	if len(joins) > 0 {
		query = e.generator.GenerateSelectWithJoins(table, columns, joins, where, orderBy, limit, offset)
	} else {
		query = e.generator.GenerateSelect(table, columns, where, orderBy, limit, offset)
	}

	rows, err := e.db.QueryContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Use optimized JOIN mapping if we have JOINs
	if len(joins) > 0 && relations != nil {
		// Validate relations before scanning
		if err := validateRelations(relations); err != nil {
			return fmt.Errorf("invalid relations: %w", err)
		}
		err = e.scanJoinResults(rows, table, joins, relations, dest)
		if err != nil {
			return err
		}
	} else {
		err = e.scanRows(rows, dest)
		if err != nil {
			return err
		}

		// Load relations if include is specified (fallback to N+1)
		if include != nil && len(include) > 0 && relations != nil {
			if err := e.loadRelations(ctx, table, include, relations, dest); err != nil {
				return fmt.Errorf("failed to load relations: %w", err)
			}
		}
	}

	return nil
}

// FindFirst executes a SELECT query with LIMIT 1 and maps to a single struct
func (e *Executor) FindFirst(ctx context.Context, table string, selectFields map[string]bool, where *sqlgen.WhereClause, orderBy []sqlgen.OrderBy, include map[string]bool, dest interface{}) error {
	return e.FindFirstWithRelations(ctx, table, selectFields, where, orderBy, include, nil, dest)
}

// FindFirstWithRelations executes a SELECT query with relations and maps to a single struct
func (e *Executor) FindFirstWithRelations(ctx context.Context, table string, selectFields map[string]bool, where *sqlgen.WhereClause, orderBy []sqlgen.OrderBy, include map[string]bool, relations map[string]RelationMetadata, dest interface{}) error {
	// Convert selectFields map to slice
	var columns []string
	if selectFields != nil && len(selectFields) > 0 {
		for field := range selectFields {
			columns = append(columns, field)
		}
	}

	var query *sqlgen.Query
	limit := 1

	// Build JOINs if relations are included
	var joins []sqlgen.Join
	if include != nil && len(include) > 0 && relations != nil {
		joins = buildJoinsFromIncludes(table, include, relations)
	}

	if len(joins) > 0 {
		query = e.generator.GenerateSelectWithJoins(table, columns, joins, where, orderBy, &limit, nil)
	} else {
		query = e.generator.GenerateSelect(table, columns, where, orderBy, &limit, nil)
	}

	rows, err := e.db.QueryContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Use optimized JOIN mapping if we have JOINs
	if len(joins) > 0 && relations != nil {
		err = e.scanJoinResults(rows, table, joins, relations, dest)
		if err != nil {
			return err
		}
	} else {
		// Single row query
		if !rows.Next() {
			return fmt.Errorf("no rows found")
		}

		columns, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get columns: %w", err)
		}

		err = e.scanRowIntoStruct(rows, columns, dest)
		if err != nil {
			return err
		}

		// Load relations if include is specified (fallback to N+1)
		if include != nil && len(include) > 0 && relations != nil {
			if err := e.loadRelations(ctx, table, include, relations, dest); err != nil {
				return fmt.Errorf("failed to load relations: %w", err)
			}
		}
	}

	return nil
}

// Create executes an INSERT query and returns the created record
func (e *Executor) Create(ctx context.Context, table string, data interface{}) (interface{}, error) {
	columns, values, err := e.extractInsertData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract insert data: %w", err)
	}

	query := e.generator.GenerateInsert(table, columns, values)

	// For PostgreSQL, we can use RETURNING
	if e.provider == "postgresql" || e.provider == "postgres" {
		row := e.db.QueryRowContext(ctx, query.SQL, query.Args...)
		return e.scanRowToStruct(row, data)
	}

	// For other databases, execute insert then query back
	result, err := e.db.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}

	// Get the last insert ID if available
	id, err := result.LastInsertId()
	if err == nil {
		// Query back the record
		where := &sqlgen.WhereClause{
			Conditions: []sqlgen.Condition{
				{Field: "id", Operator: "=", Value: id},
			},
			Operator: "AND",
		}
		var found interface{} = data
		if err := e.FindFirst(ctx, table, nil, where, nil, nil, &found); err == nil {
			return found, nil
		}
	}

	return data, nil
}

// Upsert executes an INSERT ... ON CONFLICT ... DO UPDATE query
func (e *Executor) Upsert(ctx context.Context, table string, data interface{}, conflictTarget []string, updateColumns []string) (interface{}, error) {
	columns, values, err := e.extractInsertData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract insert data: %w", err)
	}

	// If conflictTarget is empty, try to infer from primary key or unique constraints
	if len(conflictTarget) == 0 {
		// Try to find id or primary key field
		for _, col := range columns {
			if col == "id" || col == "Id" {
				conflictTarget = []string{col}
				break
			}
		}
	}

	query := e.generator.GenerateUpsert(table, columns, values, updateColumns, conflictTarget)

	// For PostgreSQL, we can use RETURNING
	if e.provider == "postgresql" || e.provider == "postgres" {
		row := e.db.QueryRowContext(ctx, query.SQL, query.Args...)
		return e.scanRowToStruct(row, data)
	}

	// For other databases, execute upsert then query back
	result, err := e.db.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return nil, fmt.Errorf("upsert failed: %w", err)
	}

	// Get the last insert ID if available
	id, err := result.LastInsertId()
	if err == nil && len(conflictTarget) > 0 {
		// Query back the record
		where := &sqlgen.WhereClause{
			Conditions: []sqlgen.Condition{
				{Field: conflictTarget[0], Operator: "=", Value: id},
			},
			Operator: "AND",
		}
		var found interface{} = data
		if err := e.FindFirst(ctx, table, nil, where, nil, nil, &found); err == nil {
			return found, nil
		}
	}

	return data, nil
}

// Update executes an UPDATE query
func (e *Executor) Update(ctx context.Context, table string, set map[string]interface{}, where *sqlgen.WhereClause, dest interface{}) error {
	query := e.generator.GenerateUpdate(table, set, where)

	// For PostgreSQL, we can use RETURNING
	if e.provider == "postgresql" || e.provider == "postgres" {
		row := e.db.QueryRowContext(ctx, query.SQL, query.Args...)
		return e.scanRow(row, dest)
	}

	// For other databases, execute update then query back
	_, err := e.db.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	// If we have a WHERE clause, try to query back the updated record
	if where != nil && len(where.Conditions) > 0 {
		// Try to find the record using the WHERE clause
		return e.FindFirst(ctx, table, nil, where, nil, nil, dest)
	}

	return nil
}

// Delete executes a DELETE query
func (e *Executor) Delete(ctx context.Context, table string, where *sqlgen.WhereClause) error {
	query := e.generator.GenerateDelete(table, where)

	_, err := e.db.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

// CreateMany executes batch INSERT queries
func (e *Executor) CreateMany(ctx context.Context, table string, data []interface{}) ([]interface{}, error) {
	if len(data) == 0 {
		return []interface{}{}, nil
	}

	var results []interface{}

	// For PostgreSQL, we can use multi-row INSERT with RETURNING
	if e.provider == "postgresql" || e.provider == "postgres" {
		// Extract columns from first record
		columns, _, err := e.extractInsertData(data[0])
		if err != nil {
			return nil, fmt.Errorf("failed to extract insert data: %w", err)
		}

		// Build multi-row INSERT
		var parts []string
		var args []interface{}
		argIndex := 1

		parts = append(parts, fmt.Sprintf("INSERT INTO %s", e.quoteIdentifier(table)))
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = e.quoteIdentifier(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
		parts = append(parts, "VALUES")

		// Build VALUES for each row
		valueParts := make([]string, len(data))
		for i, record := range data {
			_, values, err := e.extractInsertData(record)
			if err != nil {
				return nil, fmt.Errorf("failed to extract insert data for record %d: %w", i, err)
			}

			placeholders := make([]string, len(values))
			for j := range values {
				if e.provider == "postgresql" || e.provider == "postgres" {
					placeholders[j] = fmt.Sprintf("$%d", argIndex)
				} else {
					placeholders[j] = "?"
				}
				args = append(args, values[j])
				argIndex++
			}
			valueParts[i] = fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
		}

		parts = append(parts, strings.Join(valueParts, ", "))
		parts = append(parts, "RETURNING *")

		querySQL := strings.Join(parts, " ")
		rows, err := e.db.QueryContext(ctx, querySQL, args...)
		if err != nil {
			return nil, fmt.Errorf("batch insert failed: %w", err)
		}
		defer rows.Close()

		// Scan all results
		for rows.Next() {
			record := reflect.New(reflect.TypeOf(data[0]).Elem()).Interface()
			columns, err := rows.Columns()
			if err != nil {
				return nil, fmt.Errorf("failed to get columns: %w", err)
			}
			if err := e.scanRowIntoStruct(rows, columns, record); err != nil {
				return nil, err
			}
			results = append(results, record)
		}

		return results, rows.Err()
	}

	// For other databases, insert one by one (can be optimized with transactions)
	for _, record := range data {
		result, err := e.Create(ctx, table, record)
		if err != nil {
			return nil, fmt.Errorf("batch insert failed at record: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// UpdateMany executes batch UPDATE queries
func (e *Executor) UpdateMany(ctx context.Context, table string, set map[string]interface{}, where *sqlgen.WhereClause) (int64, error) {
	query := e.generator.GenerateUpdate(table, set, where)

	result, err := e.db.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return 0, fmt.Errorf("batch update failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// DeleteMany executes batch DELETE queries
func (e *Executor) DeleteMany(ctx context.Context, table string, where *sqlgen.WhereClause) (int64, error) {
	query := e.generator.GenerateDelete(table, where)

	result, err := e.db.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return 0, fmt.Errorf("batch delete failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// quoteIdentifier quotes an identifier based on provider
func (e *Executor) quoteIdentifier(name string) string {
	switch e.provider {
	case "postgresql", "postgres":
		return fmt.Sprintf(`"%s"`, name)
	case "mysql":
		return fmt.Sprintf("`%s`", name)
	case "sqlite":
		return fmt.Sprintf(`"%s"`, name)
	default:
		return name
	}
}

// scanRows scans multiple rows into a slice
func (e *Executor) scanRows(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	sliceValue := destValue.Elem()
	if sliceValue.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	elementType := sliceValue.Type().Elem()
	if elementType.Kind() == reflect.Ptr {
		elementType = elementType.Elem()
	}

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		element := reflect.New(elementType).Interface()
		if err := e.scanRowIntoStruct(rows, columns, element); err != nil {
			return err
		}
		sliceValue = reflect.Append(sliceValue, reflect.ValueOf(element))
	}

	destValue.Elem().Set(sliceValue)
	return rows.Err()
}

// scanRow scans a single row into a struct
func (e *Executor) scanRow(row *sql.Row, dest interface{}) error {
	// Get columns from the struct type
	columns := e.getStructColumns(dest)
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no rows found")
		}
		return fmt.Errorf("scan failed: %w", err)
	}

	return e.mapValuesToStruct(columns, values, dest)
}

// scanRowToStruct scans a row into a struct (for RETURNING)
func (e *Executor) scanRowToStruct(row *sql.Row, dest interface{}) (interface{}, error) {
	columns := e.getStructColumns(dest)
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	if err := e.mapValuesToStruct(columns, values, dest); err != nil {
		return nil, err
	}

	return dest, nil
}

// scanRowIntoStruct scans a row into a struct
func (e *Executor) scanRowIntoStruct(rows *sql.Rows, columns []string, dest interface{}) error {
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	return e.mapValuesToStruct(columns, values, dest)
}

// getStructColumns extracts column names from struct tags
func (e *Executor) getStructColumns(dest interface{}) []string {
	t := reflect.TypeOf(dest)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var columns []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			columns = append(columns, dbTag)
		} else {
			// Fallback to snake_case of field name
			columns = append(columns, e.toSnakeCase(field.Name))
		}
	}

	return columns
}

// mapValuesToStruct maps database values to struct fields
func (e *Executor) mapValuesToStruct(columns []string, values []interface{}, dest interface{}) error {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	columnMap := make(map[string]int)
	for i, col := range columns {
		columnMap[strings.ToLower(col)] = i
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get column name from tag or field name
		columnName := field.Tag.Get("db")
		if columnName == "" || columnName == "-" {
			columnName = e.toSnakeCase(field.Name)
		}

		colIndex, ok := columnMap[strings.ToLower(columnName)]
		if !ok {
			continue
		}

		if colIndex >= len(values) {
			continue
		}

		value := values[colIndex]
		if value == nil {
			if fieldValue.Kind() == reflect.Ptr {
				fieldValue.Set(reflect.Zero(fieldValue.Type()))
			}
			continue
		}

		if err := e.setFieldValue(fieldValue, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a struct field value from a database value
func (e *Executor) setFieldValue(fieldValue reflect.Value, value interface{}) error {
	fieldType := fieldValue.Type()

	// Handle pointer fields
	if fieldType.Kind() == reflect.Ptr {
		if value == nil {
			fieldValue.Set(reflect.Zero(fieldType))
			return nil
		}
		elemType := fieldType.Elem()
		elemValue := reflect.New(elemType).Elem()
		if err := e.setFieldValue(elemValue, value); err != nil {
			return err
		}
		fieldValue.Set(elemValue.Addr())
		return nil
	}

	// Handle slice fields
	if fieldType.Kind() == reflect.Slice {
		// For now, skip relation fields
		return nil
	}

	// Convert value to field type
	valueValue := reflect.ValueOf(value)
	if !valueValue.IsValid() {
		return nil
	}

	valueType := valueValue.Type()
	if valueType.AssignableTo(fieldType) {
		fieldValue.Set(valueValue)
		return nil
	}

	if valueType.ConvertibleTo(fieldType) {
		fieldValue.Set(valueValue.Convert(fieldType))
		return nil
	}

	return fmt.Errorf("cannot convert %s to %s", valueType, fieldType)
}

// extractInsertData extracts columns and values from a struct
func (e *Executor) extractInsertData(data interface{}) ([]string, []interface{}, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("data must be a struct")
	}

	t := v.Type()
	var columns []string
	var values []interface{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get column name from tag
		columnName := field.Tag.Get("db")
		if columnName == "" || columnName == "-" {
			columnName = e.toSnakeCase(field.Name)
		}

		// Skip zero values for optional fields (can be improved)
		if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
			continue
		}

		columns = append(columns, columnName)
		values = append(values, fieldValue.Interface())
	}

	return columns, values, nil
}

// toSnakeCase converts PascalCase to snake_case
func (e *Executor) toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
