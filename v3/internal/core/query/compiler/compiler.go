// Package compiler provides SQL query compilation.
package compiler

import (
	"context"
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema"
)

// SQLCompiler compiles domain queries to SQL.
type SQLCompiler struct {
	dialect  domain.SQLDialect
	registry *schema.MetadataRegistry // Schema metadata for relation resolution
}

// NewSQLCompiler creates a new SQL compiler.
func NewSQLCompiler(dialect domain.SQLDialect) *SQLCompiler {
	return &SQLCompiler{
		dialect:  dialect,
		registry: nil, // Will be set via SetRegistry when schema is available
	}
}

// SetRegistry sets the schema metadata registry for relation resolution.
func (c *SQLCompiler) SetRegistry(registry *schema.MetadataRegistry) {
	c.registry = registry
}

// Compile compiles a query to an executable form.
func (c *SQLCompiler) Compile(ctx context.Context, query *domain.Query) (*domain.CompiledQuery, error) {
	// Generate SQL based on operation
	var sql domain.SQL
	var err error
	var sqlStr string
	var args []interface{}

	switch query.Operation {
	case domain.FindMany, domain.FindFirst, domain.FindUnique:
		sql, err = c.compileSelect(query)
	case domain.Create:
		sqlStr, args, err = c.CompileCreate(query)
		if err == nil {
			sql = domain.SQL{Query: sqlStr, Args: args, Dialect: c.dialect}
		}
	case domain.CreateMany:
		sqlStr, args, err = c.CompileCreateMany(query)
		if err == nil {
			sql = domain.SQL{Query: sqlStr, Args: args, Dialect: c.dialect}
		}
	case domain.Update:
		sqlStr, args, err = c.CompileUpdate(query)
		if err == nil {
			sql = domain.SQL{Query: sqlStr, Args: args, Dialect: c.dialect}
		}
	case domain.UpdateMany:
		sqlStr, args, err = c.CompileUpdate(query) // UpdateMany uses same logic
		if err == nil {
			sql = domain.SQL{Query: sqlStr, Args: args, Dialect: c.dialect}
		}
	case domain.Delete, domain.DeleteMany:
		sql, err = c.compileDelete(query)
	case domain.Upsert:
		sqlStr, args, err = c.CompileUpsert(query)
		if err == nil {
			sql = domain.SQL{Query: sqlStr, Args: args, Dialect: c.dialect}
		}
	case domain.Aggregate:
		sql, err = c.compileAggregate(query)
	case domain.GroupBy:
		sql, err = c.compileGroupBy(query)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", query.Operation)
	}

	if err != nil {
		return nil, err
	}

	// Build result mapping
	mapping := c.buildResultMapping(query)

	compiled := &domain.CompiledQuery{
		SQL:           sql,
		Mapping:       mapping,
		OriginalQuery: query,
	}

	return compiled, nil
}

// Optimize optimizes a compiled query.
func (c *SQLCompiler) Optimize(ctx context.Context, compiled *domain.CompiledQuery) (*domain.CompiledQuery, error) {
	// For MVP, return as-is
	// Future: query optimization, index hints, etc.
	return compiled, nil
}

// compileSelect compiles a SELECT query.
func (c *SQLCompiler) compileSelect(query *domain.Query) (domain.SQL, error) {
	var sqlBuilder strings.Builder
	var args []interface{}
	argIndex := 1

	// Build relation JOINs if requested
	var joins []RelationJoin
	var err error
	if len(query.Relations) > 0 {
		// Get registry from compiler context (would be set during initialization)
		// For now, relation loading requires explicit registry setup
		// This will be nil until integrated with full schema loading
		joins, err = c.buildRelationJoins(query.Model, query.Model, query.Relations, c.registry)
		if err != nil {
			return domain.SQL{}, fmt.Errorf("failed to build relation joins: %w", err)
		}
	}

	// SELECT clause
	sqlBuilder.WriteString("SELECT ")

	// DISTINCT clause
	if len(query.Distinct) > 0 {
		sqlBuilder.WriteString("DISTINCT ON (")
		for i, field := range query.Distinct {
			if i > 0 {
				sqlBuilder.WriteString(", ")
			}
			sqlBuilder.WriteString(field)
		}
		sqlBuilder.WriteString(") ")
	}

	// Build column list
	var columns []string
	if len(query.Selection.Fields) > 0 {
		// Select specific fields from base table
		for _, field := range query.Selection.Fields {
			columns = append(columns, fmt.Sprintf("%s.%s", query.Model, field))
		}
	} else {
		// Select all from base table
		columns = append(columns, fmt.Sprintf("%s.*", query.Model))
	}

	// Add columns from joined relations
	if len(joins) > 0 {
		joinColumns := getJoinColumns(joins)
		columns = append(columns, joinColumns...)
	}

	sqlBuilder.WriteString(strings.Join(columns, ", "))

	// FROM clause
	sqlBuilder.WriteString(" FROM ")
	sqlBuilder.WriteString(query.Model)

	// Add JOIN clauses
	if len(joins) > 0 {
		joinSQL := generateJoinSQL(joins)
		sqlBuilder.WriteString(joinSQL)
	}

	// WHERE clause
	if len(query.Filter.Conditions) > 0 || len(query.Filter.NestedFilters) > 0 || query.Cursor != nil {
		var whereClauses []string
		var whereArgs []interface{}

		// Add regular filter conditions (including nested)
		if len(query.Filter.Conditions) > 0 || len(query.Filter.NestedFilters) > 0 {
			whereClause, args, err := c.buildWhereClause(query.Filter, &argIndex)
			if err != nil {
				return domain.SQL{}, err
			}
			if whereClause != "" {
				whereClauses = append(whereClauses, whereClause)
				whereArgs = append(whereArgs, args...)
			}
		}

		// Add cursor condition for cursor-based pagination
		if query.Cursor != nil {
			// Cursor pagination: WHERE cursor_field > cursor_value
			cursorClause := fmt.Sprintf("%s > %s", query.Cursor.Field, c.placeholder(&argIndex))
			whereClauses = append(whereClauses, cursorClause)
			whereArgs = append(whereArgs, query.Cursor.Value)
		}

		if len(whereClauses) > 0 {
			sqlBuilder.WriteString(" WHERE ")
			sqlBuilder.WriteString(strings.Join(whereClauses, " AND "))
			args = append(args, whereArgs...)
		}
	}

	// ORDER BY clause
	if len(query.Ordering) > 0 {
		sqlBuilder.WriteString(" ORDER BY ")
		for i, order := range query.Ordering {
			if i > 0 {
				sqlBuilder.WriteString(", ")
			}
			sqlBuilder.WriteString(order.Field)
			sqlBuilder.WriteString(" ")
			sqlBuilder.WriteString(string(order.Direction))
		}
	}

	// LIMIT and OFFSET
	if query.Pagination.Take != nil {
		sqlBuilder.WriteString(fmt.Sprintf(" LIMIT %s", c.placeholder(&argIndex)))
		args = append(args, *query.Pagination.Take)
	}

	if query.Pagination.Skip != nil {
		sqlBuilder.WriteString(fmt.Sprintf(" OFFSET %s", c.placeholder(&argIndex)))
		args = append(args, *query.Pagination.Skip)
	}

	// For FindFirst, limit to 1
	if query.Operation == domain.FindFirst && query.Pagination.Take == nil {
		sqlBuilder.WriteString(" LIMIT 1")
	}

	return domain.SQL{
		Query:   sqlBuilder.String(),
		Args:    args,
		Dialect: c.dialect,
	}, nil
}

// buildWhereClause builds the WHERE clause with support for nested filters.
// Handles both direct conditions and nested filter groups recursively.
func (c *SQLCompiler) buildWhereClause(filter domain.Filter, argIndex *int) (string, []interface{}, error) {
	if len(filter.Conditions) == 0 && len(filter.NestedFilters) == 0 {
		return "", nil, nil
	}

	var parts []string
	var args []interface{}

	// Process direct conditions
	for _, condition := range filter.Conditions {
		clause, condArgs, err := c.buildCondition(condition, argIndex)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, clause)
		args = append(args, condArgs...)
	}

	// Process nested filters recursively
	for _, nestedFilter := range filter.NestedFilters {
		nestedClause, nestedArgs, err := c.buildWhereClause(nestedFilter, argIndex)
		if err != nil {
			return "", nil, err
		}
		// Wrap nested clause in parentheses for proper precedence
		if nestedClause != "" {
			parts = append(parts, "("+nestedClause+")")
			args = append(args, nestedArgs...)
		}
	}

	if len(parts) == 0 {
		return "", nil, nil
	}

	// Determine the operator to join parts
	operator := " AND "
	if filter.Operator == domain.OR {
		operator = " OR "
	} else if filter.Operator == domain.NOT {
		// For NOT, we wrap each part with NOT and join with AND
		for i, part := range parts {
			parts[i] = "NOT (" + part + ")"
		}
		operator = " AND "
	}

	return strings.Join(parts, operator), args, nil
}

// buildCondition builds a single condition.
func (c *SQLCompiler) buildCondition(condition domain.Condition, argIndex *int) (string, []interface{}, error) {
	var clause string
	var args []interface{}

	switch condition.Operator {
	case domain.Equals:
		clause = fmt.Sprintf("%s = %s", condition.Field, c.placeholder(argIndex))
		args = append(args, condition.Value)

	case domain.NotEquals:
		clause = fmt.Sprintf("%s != %s", condition.Field, c.placeholder(argIndex))
		args = append(args, condition.Value)

	case domain.In:
		// Assumes condition.Value is a slice
		values, ok := condition.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("IN operator requires a slice of values")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = c.placeholder(argIndex)
			args = append(args, values[i])
		}
		clause = fmt.Sprintf("%s IN (%s)", condition.Field, strings.Join(placeholders, ", "))

	case domain.NotIn:
		values, ok := condition.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("NOT IN operator requires a slice of values")
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = c.placeholder(argIndex)
			args = append(args, values[i])
		}
		clause = fmt.Sprintf("%s NOT IN (%s)", condition.Field, strings.Join(placeholders, ", "))

	case domain.Lt:
		clause = fmt.Sprintf("%s < %s", condition.Field, c.placeholder(argIndex))
		args = append(args, condition.Value)

	case domain.Lte:
		clause = fmt.Sprintf("%s <= %s", condition.Field, c.placeholder(argIndex))
		args = append(args, condition.Value)

	case domain.Gt:
		clause = fmt.Sprintf("%s > %s", condition.Field, c.placeholder(argIndex))
		args = append(args, condition.Value)

	case domain.Gte:
		clause = fmt.Sprintf("%s >= %s", condition.Field, c.placeholder(argIndex))
		args = append(args, condition.Value)

	case domain.Contains:
		if condition.Mode == domain.ModeInsensitive {
			clause = fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", condition.Field, c.placeholder(argIndex))
		} else {
			clause = fmt.Sprintf("%s LIKE %s", condition.Field, c.placeholder(argIndex))
		}
		args = append(args, fmt.Sprintf("%%%v%%", condition.Value))

	case domain.StartsWith:
		if condition.Mode == domain.ModeInsensitive {
			clause = fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", condition.Field, c.placeholder(argIndex))
		} else {
			clause = fmt.Sprintf("%s LIKE %s", condition.Field, c.placeholder(argIndex))
		}
		args = append(args, fmt.Sprintf("%v%%", condition.Value))

	case domain.EndsWith:
		if condition.Mode == domain.ModeInsensitive {
			clause = fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", condition.Field, c.placeholder(argIndex))
		} else {
			clause = fmt.Sprintf("%s LIKE %s", condition.Field, c.placeholder(argIndex))
		}
		args = append(args, fmt.Sprintf("%%%v", condition.Value))

	case domain.IsEmpty:
		// IsEmpty checks if array is empty or null
		// Value should be a bool: true = empty, false = not empty
		if isEmpty, ok := condition.Value.(bool); ok {
			if isEmpty {
				clause = fmt.Sprintf("(COALESCE(array_length(%s, 1), 0) = 0)", condition.Field)
			} else {
				clause = fmt.Sprintf("(array_length(%s, 1) > 0)", condition.Field)
			}
		} else {
			return "", nil, fmt.Errorf("isEmpty operator requires a boolean value")
		}

	case domain.Has:
		// Has checks if array contains a single value (PostgreSQL @> operator)
		switch c.dialect {
		case domain.PostgreSQL:
			clause = fmt.Sprintf("%s @> ARRAY[%s]", condition.Field, c.placeholder(argIndex))
			args = append(args, condition.Value)
		default:
			// For MySQL/SQLite, use JSON_CONTAINS or similar
			clause = fmt.Sprintf("JSON_CONTAINS(%s, %s)", condition.Field, c.placeholder(argIndex))
			args = append(args, condition.Value)
		}

	case domain.HasEvery:
		// HasEvery checks if array contains ALL values (PostgreSQL @> operator with array)
		values, ok := condition.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("hasEvery operator requires a slice of values")
		}
		switch c.dialect {
		case domain.PostgreSQL:
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = c.placeholder(argIndex)
				args = append(args, values[i])
			}
			clause = fmt.Sprintf("%s @> ARRAY[%s]", condition.Field, strings.Join(placeholders, ", "))
		default:
			return "", nil, fmt.Errorf("hasEvery operator not supported for dialect: %s", c.dialect)
		}

	case domain.HasSome:
		// HasSome checks if array contains ANY of the values (PostgreSQL && operator)
		values, ok := condition.Value.([]interface{})
		if !ok {
			return "", nil, fmt.Errorf("hasSome operator requires a slice of values")
		}
		switch c.dialect {
		case domain.PostgreSQL:
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = c.placeholder(argIndex)
				args = append(args, values[i])
			}
			clause = fmt.Sprintf("%s && ARRAY[%s]", condition.Field, strings.Join(placeholders, ", "))
		default:
			return "", nil, fmt.Errorf("hasSome operator not supported for dialect: %s", c.dialect)
		}

	case domain.IsNull:
		// IsNull checks if field is null
		// Value should be a bool: true = IS NULL, false = IS NOT NULL
		if isNull, ok := condition.Value.(bool); ok {
			if isNull {
				clause = fmt.Sprintf("%s IS NULL", condition.Field)
			} else {
				clause = fmt.Sprintf("%s IS NOT NULL", condition.Field)
			}
		} else {
			return "", nil, fmt.Errorf("isNull operator requires a boolean value")
		}

	case domain.Search:
		// Search performs fulltext search (PostgreSQL to_tsvector/to_tsquery)
		switch c.dialect {
		case domain.PostgreSQL:
			clause = fmt.Sprintf("to_tsvector(%s) @@ to_tsquery(%s)", condition.Field, c.placeholder(argIndex))
			args = append(args, condition.Value)
		case domain.MySQL:
			clause = fmt.Sprintf("MATCH(%s) AGAINST(%s IN NATURAL LANGUAGE MODE)", condition.Field, c.placeholder(argIndex))
			args = append(args, condition.Value)
		default:
			// Fallback to LIKE for SQLite
			clause = fmt.Sprintf("%s LIKE %s", condition.Field, c.placeholder(argIndex))
			args = append(args, fmt.Sprintf("%%%v%%", condition.Value))
		}

	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", condition.Operator)
	}

	return clause, args, nil
}

// compileGroupBy compiles a GROUP BY query.
func (c *SQLCompiler) compileGroupBy(query *domain.Query) (domain.SQL, error) {
	if len(query.GroupBy) == 0 {
		return domain.SQL{}, fmt.Errorf("GROUP BY requires at least one field")
	}

	var sqlBuilder strings.Builder
	var args []interface{}
	argIndex := 1

	// SELECT clause - include group by fields and aggregations
	sqlBuilder.WriteString("SELECT ")

	// Add group by fields
	for i, field := range query.GroupBy {
		if i > 0 {
			sqlBuilder.WriteString(", ")
		}
		sqlBuilder.WriteString(field)
	}

	// Add aggregations
	for _, agg := range query.Aggregations {
		sqlBuilder.WriteString(", ")
		sqlBuilder.WriteString(fmt.Sprintf("%s(%s) as %s_%s",
			strings.ToUpper(string(agg.Function)),
			agg.Field,
			strings.ToLower(string(agg.Function)),
			agg.Field))
	}

	// FROM clause
	sqlBuilder.WriteString(" FROM ")
	sqlBuilder.WriteString(query.Model)

	// WHERE clause
	if len(query.Filter.Conditions) > 0 {
		whereClause, whereArgs, err := c.buildWhereClause(query.Filter, &argIndex)
		if err != nil {
			return domain.SQL{}, err
		}
		sqlBuilder.WriteString(" WHERE ")
		sqlBuilder.WriteString(whereClause)
		args = append(args, whereArgs...)
	}

	// GROUP BY clause
	sqlBuilder.WriteString(" GROUP BY ")
	for i, field := range query.GroupBy {
		if i > 0 {
			sqlBuilder.WriteString(", ")
		}
		sqlBuilder.WriteString(field)
	}

	// HAVING clause
	if len(query.Having.Conditions) > 0 {
		havingClause, havingArgs, err := c.buildWhereClause(query.Having, &argIndex)
		if err != nil {
			return domain.SQL{}, err
		}
		sqlBuilder.WriteString(" HAVING ")
		sqlBuilder.WriteString(havingClause)
		args = append(args, havingArgs...)
	}

	// ORDER BY clause
	if len(query.Ordering) > 0 {
		sqlBuilder.WriteString(" ORDER BY ")
		for i, order := range query.Ordering {
			if i > 0 {
				sqlBuilder.WriteString(", ")
			}
			sqlBuilder.WriteString(order.Field)
			sqlBuilder.WriteString(" ")
			sqlBuilder.WriteString(string(order.Direction))
		}
	}

	// LIMIT and OFFSET
	if query.Pagination.Take != nil {
		sqlBuilder.WriteString(fmt.Sprintf(" LIMIT %s", c.placeholder(&argIndex)))
		args = append(args, *query.Pagination.Take)
	}

	if query.Pagination.Skip != nil {
		sqlBuilder.WriteString(fmt.Sprintf(" OFFSET %s", c.placeholder(&argIndex)))
		args = append(args, *query.Pagination.Skip)
	}

	return domain.SQL{
		Query:   sqlBuilder.String(),
		Args:    args,
		Dialect: c.dialect,
	}, nil
}

// CompileNestedWrites compiles nested write operations into SQL statements.
// Returns multiple SQL statements that should be executed in a transaction.
func (c *SQLCompiler) CompileNestedWrites(parentTable string, parentID interface{}, nestedWrites []domain.NestedWrite) ([]domain.SQL, error) {
	var statements []domain.SQL

	for _, nw := range nestedWrites {
		argIndex := 1
		var sql string
		var args []interface{}

		switch nw.Operation {
		case domain.NestedCreate:
			// INSERT INTO related_table (parent_id, ...fields) VALUES ($1, ...)
			if len(nw.Data) == 0 {
				return nil, fmt.Errorf("nested create requires data")
			}

			var fields []string
			var placeholders []string

			// Add parent reference (assumes foreign key follows convention: parentTable_id)
			fkField := strings.ToLower(parentTable) + "_id"
			fields = append(fields, fkField)
			placeholders = append(placeholders, c.placeholder(&argIndex))
			args = append(args, parentID)

			for field, value := range nw.Data {
				fields = append(fields, field)
				placeholders = append(placeholders, c.placeholder(&argIndex))
				args = append(args, value)
			}

			sql = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
				nw.Relation,
				strings.Join(fields, ", "),
				strings.Join(placeholders, ", "))

			if c.dialect == domain.PostgreSQL {
				sql += " RETURNING *"
			}

		case domain.NestedConnect:
			// UPDATE related_table SET parent_id = $1 WHERE ...conditions
			if len(nw.Where) == 0 {
				return nil, fmt.Errorf("nested connect requires where conditions")
			}

			fkField := strings.ToLower(parentTable) + "_id"
			sql = fmt.Sprintf("UPDATE %s SET %s = %s",
				nw.Relation, fkField, c.placeholder(&argIndex))
			args = append(args, parentID)

			whereClause, whereArgs, err := c.buildWhereClause(domain.Filter{Conditions: nw.Where}, &argIndex)
			if err != nil {
				return nil, err
			}
			sql += " WHERE " + whereClause
			args = append(args, whereArgs...)

		case domain.NestedDisconnect:
			// UPDATE related_table SET parent_id = NULL WHERE ...conditions
			if len(nw.Where) == 0 {
				return nil, fmt.Errorf("nested disconnect requires where conditions")
			}

			fkField := strings.ToLower(parentTable) + "_id"
			sql = fmt.Sprintf("UPDATE %s SET %s = NULL", nw.Relation, fkField)

			whereClause, whereArgs, err := c.buildWhereClause(domain.Filter{Conditions: nw.Where}, &argIndex)
			if err != nil {
				return nil, err
			}
			sql += " WHERE " + whereClause
			args = append(args, whereArgs...)

		case domain.NestedUpdate:
			// UPDATE related_table SET ...data WHERE parent_id = $1 AND ...conditions
			if len(nw.Data) == 0 {
				return nil, fmt.Errorf("nested update requires data")
			}

			var setClauses []string
			for field, value := range nw.Data {
				setClauses = append(setClauses, fmt.Sprintf("%s = %s", field, c.placeholder(&argIndex)))
				args = append(args, value)
			}

			sql = fmt.Sprintf("UPDATE %s SET %s", nw.Relation, strings.Join(setClauses, ", "))

			// Add parent filter
			fkField := strings.ToLower(parentTable) + "_id"
			sql += fmt.Sprintf(" WHERE %s = %s", fkField, c.placeholder(&argIndex))
			args = append(args, parentID)

			// Add additional conditions
			if len(nw.Where) > 0 {
				whereClause, whereArgs, err := c.buildWhereClause(domain.Filter{Conditions: nw.Where}, &argIndex)
				if err != nil {
					return nil, err
				}
				sql += " AND " + whereClause
				args = append(args, whereArgs...)
			}

		case domain.NestedDelete:
			// DELETE FROM related_table WHERE parent_id = $1 AND ...conditions
			fkField := strings.ToLower(parentTable) + "_id"
			sql = fmt.Sprintf("DELETE FROM %s WHERE %s = %s", nw.Relation, fkField, c.placeholder(&argIndex))
			args = append(args, parentID)

			if len(nw.Where) > 0 {
				whereClause, whereArgs, err := c.buildWhereClause(domain.Filter{Conditions: nw.Where}, &argIndex)
				if err != nil {
					return nil, err
				}
				sql += " AND " + whereClause
				args = append(args, whereArgs...)
			}

		case domain.NestedSet:
			// First disconnect all, then connect specified ones
			fkField := strings.ToLower(parentTable) + "_id"

			// Disconnect all
			disconnectSQL := fmt.Sprintf("UPDATE %s SET %s = NULL WHERE %s = %s",
				nw.Relation, fkField, fkField, c.placeholder(&argIndex))
			disconnectArgs := []interface{}{parentID}

			statements = append(statements, domain.SQL{
				Query:   disconnectSQL,
				Args:    disconnectArgs,
				Dialect: c.dialect,
			})

			// Connect specified ones
			if len(nw.Where) > 0 {
				argIndex = 1
				connectSQL := fmt.Sprintf("UPDATE %s SET %s = %s",
					nw.Relation, fkField, c.placeholder(&argIndex))
				connectArgs := []interface{}{parentID}

				whereClause, whereArgs, err := c.buildWhereClause(domain.Filter{Conditions: nw.Where}, &argIndex)
				if err != nil {
					return nil, err
				}
				connectSQL += " WHERE " + whereClause
				connectArgs = append(connectArgs, whereArgs...)

				statements = append(statements, domain.SQL{
					Query:   connectSQL,
					Args:    connectArgs,
					Dialect: c.dialect,
				})
			}
			continue // Already added statements, skip the append below

		default:
			return nil, fmt.Errorf("unsupported nested operation: %s", nw.Operation)
		}

		statements = append(statements, domain.SQL{
			Query:   sql,
			Args:    args,
			Dialect: c.dialect,
		})
	}

	return statements, nil
}

// placeholder returns the appropriate placeholder for the dialect.
func (c *SQLCompiler) placeholder(argIndex *int) string {
	defer func() { *argIndex++ }()

	switch c.dialect {
	case domain.PostgreSQL:
		return fmt.Sprintf("$%d", *argIndex)
	case domain.MySQL, domain.SQLite:
		return "?"
	default:
		return "?"
	}
}

// buildResultMapping builds the result mapping for a query.
func (c *SQLCompiler) buildResultMapping(query *domain.Query) domain.ResultMapping {
	mapping := domain.ResultMapping{
		Model:  query.Model,
		Fields: []domain.FieldMapping{},
	}

	// Map selected fields or all fields
	if len(query.Selection.Fields) > 0 {
		for _, field := range query.Selection.Fields {
			mapping.Fields = append(mapping.Fields, domain.FieldMapping{
				Field:  field,
				Column: field,
				Type:   "unknown", // Will be determined by schema later
			})
		}
	}

	return mapping
}

// Ensure SQLCompiler implements QueryCompiler interface.
var _ domain.QueryCompiler = (*SQLCompiler)(nil)
