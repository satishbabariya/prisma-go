// Package executor executes queries and maps results to structs.
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/satishbabariya/prisma-go/query/builder"
	"github.com/satishbabariya/prisma-go/query/cache"
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// Executor executes queries and maps results
type Executor struct {
	db           *sql.DB
	provider     string
	generator    sqlgen.Generator
	stmtCache    map[string]*sql.Stmt
	cacheMu      sync.RWMutex
	queryCache   cache.Cache
	cacheEnabled bool
}

// NewExecutor creates a new query executor
func NewExecutor(db *sql.DB, provider string) *Executor {
	return &Executor{
		db:           db,
		provider:     provider,
		generator:    sqlgen.NewGenerator(provider),
		stmtCache:    make(map[string]*sql.Stmt),
		queryCache:   nil, // Cache disabled by default
		cacheEnabled: false,
	}
}

// SetCache enables query caching with the provided cache instance
func (e *Executor) SetCache(c cache.Cache) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	e.queryCache = c
	e.cacheEnabled = c != nil
}

// EnableCache enables query caching with default settings
func (e *Executor) EnableCache(maxSize int, defaultTTL time.Duration) {
	e.SetCache(cache.NewLRUCache(maxSize, defaultTTL))
}

// DisableCache disables query caching
func (e *Executor) DisableCache() {
	e.SetCache(nil)
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

// GetGenerator returns the SQL generator for this executor
func (e *Executor) GetGenerator() sqlgen.Generator {
	return e.generator
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
		joins = buildJoinsFromIncludes(table, include, relations, e.provider)
	}

	if len(joins) > 0 {
		query = e.generator.GenerateSelectWithJoins(table, columns, joins, where, orderBy, limit, offset)
	} else {
		query = e.generator.GenerateSelect(table, columns, where, orderBy, limit, offset)
	}

	// Check cache if enabled
	if e.cacheEnabled && e.queryCache != nil {
		cacheKey := cache.GenerateCacheKey(query.SQL, query.Args)
		if cached, ok := e.queryCache.Get(cacheKey); ok {
			// Copy cached result to destination
			return e.copyCachedResult(cached, dest)
		}
	}

	rows, err := e.db.QueryContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("query execution failed for table %q: SQL=%q, args=%v: %w", table, query.SQL, query.Args, err)
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
			return fmt.Errorf("failed to scan join results for table %q: %w", table, err)
		}
	} else {
		err = e.scanRows(rows, dest)
		if err != nil {
			return fmt.Errorf("failed to scan rows for table %q: %w", table, err)
		}

		// Relations are loaded via JOINs above, no need for N+1 fallback
	}

	// Cache result if enabled
	if e.cacheEnabled && e.queryCache != nil {
		cacheKey := cache.GenerateCacheKey(query.SQL, query.Args)
		// Create a deep copy for caching
		cachedValue := reflect.New(reflect.TypeOf(dest).Elem())
		if err := e.copyCachedResult(dest, cachedValue.Interface()); err == nil {
			e.queryCache.Set(cacheKey, cachedValue.Elem().Interface(), 0) // Use default TTL
		}
	}

	return nil
}

// FindManyWithJoins executes a SELECT query with explicit JOINs and maps results to a slice
func (e *Executor) FindManyWithJoins(ctx context.Context, table string, selectFields map[string]bool, joins []sqlgen.Join, where *sqlgen.WhereClause, orderBy []sqlgen.OrderBy, limit, offset *int, include map[string]bool, relations map[string]RelationMetadata, dest interface{}) error {
	// Convert selectFields map to slice
	var columns []string
	if selectFields != nil && len(selectFields) > 0 {
		for field := range selectFields {
			columns = append(columns, field)
		}
	}

	var query *sqlgen.Query

	// Merge explicit joins with relation-based joins
	var allJoins []sqlgen.Join
	allJoins = append(allJoins, joins...)

	// Add relation-based joins if includes are specified
	if include != nil && len(include) > 0 && relations != nil {
		relationJoins := buildJoinsFromIncludes(table, include, relations, e.provider)
		allJoins = append(allJoins, relationJoins...)
	}

	if len(allJoins) > 0 {
		query = e.generator.GenerateSelectWithJoins(table, columns, allJoins, where, orderBy, limit, offset)
	} else {
		query = e.generator.GenerateSelect(table, columns, where, orderBy, limit, offset)
	}

	// Check cache if enabled
	if e.cacheEnabled && e.queryCache != nil {
		cacheKey := cache.GenerateCacheKey(query.SQL, query.Args)
		if cached, ok := e.queryCache.Get(cacheKey); ok {
			return e.copyCachedResult(cached, dest)
		}
	}

	rows, err := e.db.QueryContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Use optimized JOIN mapping if we have JOINs
	if len(allJoins) > 0 && relations != nil {
		if err := validateRelations(relations); err != nil {
			return fmt.Errorf("invalid relations: %w", err)
		}
		err = e.scanJoinResults(rows, table, allJoins, relations, dest)
		if err != nil {
			return err
		}
	} else {
		// Simple scan without JOINs - use existing scanJoinResults with empty joins
		err = e.scanJoinResults(rows, table, []sqlgen.Join{}, nil, dest)
		if err != nil {
			return fmt.Errorf("failed to scan results: %w", err)
		}
	}

	// Cache result if enabled
	if e.cacheEnabled && e.queryCache != nil {
		cacheKey := cache.GenerateCacheKey(query.SQL, query.Args)
		cachedValue := reflect.New(reflect.TypeOf(dest).Elem())
		if err := e.copyCachedResult(dest, cachedValue.Interface()); err == nil {
			e.queryCache.Set(cacheKey, cachedValue.Elem().Interface(), 0)
		}
	}

	return nil
}

// FindFirstWithJoins executes a SELECT query with explicit JOINs and returns the first result
func (e *Executor) FindFirstWithJoins(ctx context.Context, table string, selectFields map[string]bool, joins []sqlgen.Join, where *sqlgen.WhereClause, orderBy []sqlgen.OrderBy, include map[string]bool, relations map[string]RelationMetadata, dest interface{}) error {
	// Convert selectFields map to slice
	var columns []string
	if selectFields != nil && len(selectFields) > 0 {
		for field := range selectFields {
			columns = append(columns, field)
		}
	}

	var query *sqlgen.Query
	limit := 1

	// Merge explicit joins with relation-based joins
	var allJoins []sqlgen.Join
	allJoins = append(allJoins, joins...)

	// Add relation-based joins if includes are specified
	if include != nil && len(include) > 0 && relations != nil {
		relationJoins := buildJoinsFromIncludes(table, include, relations, e.provider)
		allJoins = append(allJoins, relationJoins...)
	}

	if len(allJoins) > 0 {
		query = e.generator.GenerateSelectWithJoins(table, columns, allJoins, where, orderBy, &limit, nil)
	} else {
		query = e.generator.GenerateSelect(table, columns, where, orderBy, &limit, nil)
	}

	rows, err := e.db.QueryContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}

	// Use optimized JOIN mapping if we have JOINs
	if len(allJoins) > 0 && relations != nil {
		if err := validateRelations(relations); err != nil {
			return fmt.Errorf("invalid relations: %w", err)
		}
		// For single row, we still use scanJoinResults but limit to first result
		err = e.scanJoinResults(rows, table, allJoins, relations, dest)
		if err != nil {
			return err
		}
	} else {
		// Simple scan without JOINs - scan first row only
		if rows.Next() {
			err = e.scanRows(rows, dest)
			if err != nil {
				return fmt.Errorf("failed to scan result: %w", err)
			}
		} else {
			return sql.ErrNoRows
		}
	}

	return nil
}

// copyCachedResult copies a cached result to the destination
func (e *Executor) copyCachedResult(cached interface{}, dest interface{}) error {
	cachedValue := reflect.ValueOf(cached)
	destValue := reflect.ValueOf(dest)

	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	destElem := destValue.Elem()
	cachedElem := cachedValue

	if cachedValue.Kind() == reflect.Ptr {
		cachedElem = cachedValue.Elem()
	}

	// Copy the cached value to destination
	if destElem.Type() == cachedElem.Type() {
		destElem.Set(cachedElem)
		return nil
	}

	// Handle slice copying
	if destElem.Kind() == reflect.Slice && cachedElem.Kind() == reflect.Slice {
		if destElem.Type().Elem() == cachedElem.Type().Elem() {
			newSlice := reflect.MakeSlice(destElem.Type(), cachedElem.Len(), cachedElem.Cap())
			reflect.Copy(newSlice, cachedElem)
			destElem.Set(newSlice)
			return nil
		}
	}

	return fmt.Errorf("cannot copy cached result: type mismatch")
}

// invalidateTableCache invalidates all cache entries for a specific table
func (e *Executor) invalidateTableCache(table string) {
	if e.cacheEnabled && e.queryCache != nil {
		// Invalidate all queries for this table
		pattern := fmt.Sprintf("query:*%s*", table)
		e.queryCache.InvalidatePattern(pattern)
		// Also invalidate by table name pattern
		e.queryCache.InvalidatePattern(fmt.Sprintf("*:%s:*", table))
	}
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
		joins = buildJoinsFromIncludes(table, include, relations, e.provider)
	}

	if len(joins) > 0 {
		query = e.generator.GenerateSelectWithJoins(table, columns, joins, where, orderBy, &limit, nil)
	} else {
		query = e.generator.GenerateSelect(table, columns, where, orderBy, &limit, nil)
	}

	// Check cache if enabled
	if e.cacheEnabled && e.queryCache != nil {
		cacheKey := cache.GenerateCacheKey(query.SQL, query.Args)
		if cached, ok := e.queryCache.Get(cacheKey); ok {
			// Copy cached result to destination
			return e.copyCachedResult(cached, dest)
		}
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

		// Relations are loaded via JOINs above, no need for N+1 fallback
	}

	return nil
}

// Create executes an INSERT query and returns the created record
func (e *Executor) Create(ctx context.Context, table string, data interface{}, nestedWrites ...*builder.NestedWriteOperation) (interface{}, error) {
	// Invalidate cache for this table
	e.invalidateTableCache(table)

	// Start transaction for nested writes
	var tx *sql.Tx
	var err error
	if len(nestedWrites) > 0 {
		tx, err = e.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to start transaction: %w", err)
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}()
	}

	columns, values, err := e.extractInsertData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract insert data: %w", err)
	}

	query := e.generator.GenerateInsert(table, columns, values)

	var result sql.Result
	var insertedID interface{}

	// Execute INSERT
	if tx != nil {
		result, err = tx.ExecContext(ctx, query.SQL, query.Args...)
	} else {
		result, err = e.db.ExecContext(ctx, query.SQL, query.Args...)
	}

	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}

	// Get the last insert ID if available
	id, err := result.LastInsertId()
	if err == nil {
		insertedID = id
	} else {
		// Try to extract ID from data
		insertedID = e.extractIDFromData(data)
	}

	// Execute nested writes if provided
	if len(nestedWrites) > 0 && insertedID != nil {
		// Extract relation metadata (simplified - in real implementation, this should come from schema)
		relations := make(map[string]RelationMetadata) // TODO: Get from schema AST
		if err := e.ExecuteNestedWrites(ctx, tx, table, insertedID, nestedWrites, relations); err != nil {
			return nil, fmt.Errorf("failed to execute nested writes: %w", err)
		}
	}

	// Query back the record
	if tx != nil {
		// Use transaction for query
		where := &sqlgen.WhereClause{
			Conditions: []sqlgen.Condition{
				{Field: "id", Operator: "=", Value: insertedID},
			},
			Operator: "AND",
		}
		var found interface{} = data
		// For now, query using regular connection (transaction query would need separate method)
		if err := e.FindFirst(ctx, table, nil, where, nil, nil, &found); err == nil {
			return found, nil
		}
	} else {
		// For PostgreSQL, we can use RETURNING
		if e.provider == "postgresql" || e.provider == "postgres" {
			row := e.db.QueryRowContext(ctx, query.SQL, query.Args...)
			return e.scanRowToStruct(row, data)
		}

		// Query back the record
		if insertedID != nil {
			where := &sqlgen.WhereClause{
				Conditions: []sqlgen.Condition{
					{Field: "id", Operator: "=", Value: insertedID},
				},
				Operator: "AND",
			}
			var found interface{} = data
			if err := e.FindFirst(ctx, table, nil, where, nil, nil, &found); err == nil {
				return found, nil
			}
		}
	}

	return data, nil
}

// extractIDFromData extracts ID from data struct
func (e *Executor) extractIDFromData(data interface{}) interface{} {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		idField := v.FieldByName("Id")
		if !idField.IsValid() {
			idField = v.FieldByName("ID")
		}
		if idField.IsValid() && idField.CanInterface() {
			return idField.Interface()
		}
	}

	return nil
}

// Upsert executes an INSERT ... ON CONFLICT ... DO UPDATE query
func (e *Executor) Upsert(ctx context.Context, table string, data interface{}, conflictTarget []string, updateColumns []string) (interface{}, error) {
	// Invalidate cache for this table
	e.invalidateTableCache(table)

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
	// Invalidate cache for this table
	e.invalidateTableCache(table)

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
	// Invalidate cache for this table
	e.invalidateTableCache(table)

	query := e.generator.GenerateDelete(table, where)

	_, err := e.db.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

// CreateMany executes batch INSERT queries
func (e *Executor) CreateMany(ctx context.Context, table string, data []interface{}) ([]interface{}, error) {
	// Invalidate cache for this table
	e.invalidateTableCache(table)

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
	// Invalidate cache for this table
	e.invalidateTableCache(table)

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
	// Invalidate cache for this table
	e.invalidateTableCache(table)

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
			return fmt.Errorf("failed to scan row into struct (columns: %v): %w", columns, err)
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

	// Special handling for SQLite boolean conversion (int64 -> bool)
	// SQLite stores booleans as INTEGER (0 or 1), but Go expects bool
	if fieldType.Kind() == reflect.Bool {
		switch v := value.(type) {
		case int64:
			fieldValue.SetBool(v != 0)
			return nil
		case int32:
			fieldValue.SetBool(v != 0)
			return nil
		case int:
			fieldValue.SetBool(v != 0)
			return nil
		case int16:
			fieldValue.SetBool(v != 0)
			return nil
		case int8:
			fieldValue.SetBool(v != 0)
			return nil
		case uint64:
			fieldValue.SetBool(v != 0)
			return nil
		case uint32:
			fieldValue.SetBool(v != 0)
			return nil
		case uint:
			fieldValue.SetBool(v != 0)
			return nil
		case uint16:
			fieldValue.SetBool(v != 0)
			return nil
		case uint8:
			fieldValue.SetBool(v != 0)
			return nil
		}
	}

	// Special handling for SQLite DateTime conversion (string -> time.Time)
	// SQLite stores DateTime as TEXT, but Go expects time.Time
	if fieldType == reflect.TypeOf(time.Time{}) {
		var timeStr string
		var ok bool

		// Handle string type
		if timeStr, ok = value.(string); !ok {
			// Handle []byte (some drivers return bytes)
			if bytes, ok := value.([]byte); ok {
				timeStr = string(bytes)
			} else {
				return fmt.Errorf("cannot convert %s to time.Time: value is not string or []byte", valueType)
			}
		}

		// Try common SQLite datetime formats (including space-separated RFC3339-like)
		// First, try converting space to T for RFC3339 parsing
		if strings.Contains(timeStr, " ") && !strings.Contains(timeStr, "T") {
			// Replace space with T to make it RFC3339-compatible
			rfc3339Str := strings.Replace(timeStr, " ", "T", 1)
			if t, err := time.Parse(time.RFC3339Nano, rfc3339Str); err == nil {
				fieldValue.Set(reflect.ValueOf(t))
				return nil
			}
			if t, err := time.Parse(time.RFC3339, rfc3339Str); err == nil {
				fieldValue.Set(reflect.ValueOf(t))
				return nil
			}
		}

		formats := []string{
			time.RFC3339Nano,                      // T-separated RFC3339Nano
			time.RFC3339,                          // T-separated RFC3339
			"2006-01-02 15:04:05.999999999-07:00", // Space-separated with timezone
			"2006-01-02 15:04:05.999999999+07:00", // Space-separated with +timezone
			"2006-01-02 15:04:05.999999999Z07:00", // Space-separated with Z timezone
			"2006-01-02T15:04:05.999999999Z07:00", // T-separated with timezone
			"2006-01-02T15:04:05Z07:00",           // T-separated without nanoseconds
			"2006-01-02T15:04:05",                 // T-separated without timezone
			"2006-01-02 15:04:05.999999999",       // Space-separated without timezone
			"2006-01-02 15:04:05",                 // Space-separated simple
		}

		for _, format := range formats {
			if t, err := time.Parse(format, timeStr); err == nil {
				fieldValue.Set(reflect.ValueOf(t))
				return nil
			}
		}

		// Last resort: try parsing with time.ParseInLocation for local time
		if t, err := time.ParseInLocation("2006-01-02 15:04:05.999999999", timeStr, time.Local); err == nil {
			fieldValue.Set(reflect.ValueOf(t))
			return nil
		}
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local); err == nil {
			fieldValue.Set(reflect.ValueOf(t))
			return nil
		}

		return fmt.Errorf("cannot parse time string %q to time.Time", timeStr)
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
