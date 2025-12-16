// Package compiler implements SQL compilation from queries.
package compiler

import (
	"context"
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// SQLCompiler implements the domain.QueryCompiler interface.
type SQLCompiler struct {
	dialect domain.SQLDialect
}

// NewSQLCompiler creates a new SQL compiler.
func NewSQLCompiler(dialect domain.SQLDialect) *SQLCompiler {
	return &SQLCompiler{
		dialect: dialect,
	}
}

// Compile compiles a query to an executable form.
func (c *SQLCompiler) Compile(ctx context.Context, query *domain.Query) (*domain.CompiledQuery, error) {
	// Generate SQL based on operation
	var sql domain.SQL
	var err error

	switch query.Operation {
	case domain.FindMany, domain.FindFirst, domain.FindUnique:
		sql, err = c.compileSelect(query)
	case domain.Delete, domain.DeleteMany:
		sql, err = c.compileDelete(query)
	case domain.Aggregate:
		sql, err = c.compileAggregate(query)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", query.Operation)
	}

	if err != nil {
		return nil, err
	}

	// Build result mapping
	mapping := c.buildResultMapping(query)

	compiled := &domain.CompiledQuery{
		SQL:     sql,
		Mapping: mapping,
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

	// SELECT clause
	sqlBuilder.WriteString("SELECT ")
	if len(query.Selection.Fields) > 0 {
		for i, field := range query.Selection.Fields {
			if i > 0 {
				sqlBuilder.WriteString(", ")
			}
			sqlBuilder.WriteString(field)
		}
	} else {
		sqlBuilder.WriteString("*")
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

// buildWhereClause builds the WHERE clause.
func (c *SQLCompiler) buildWhereClause(filter domain.Filter, argIndex *int) (string, []interface{}, error) {
	if len(filter.Conditions) == 0 {
		return "", nil, nil
	}

	var clauses []string
	var args []interface{}

	for _, condition := range filter.Conditions {
		clause, condArgs, err := c.buildCondition(condition, argIndex)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, clause)
		args = append(args, condArgs...)
	}

	operator := " AND "
	if filter.Operator == domain.OR {
		operator = " OR "
	}

	return strings.Join(clauses, operator), args, nil
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
		clause = fmt.Sprintf("%s LIKE %s", condition.Field, c.placeholder(argIndex))
		args = append(args, fmt.Sprintf("%%%v%%", condition.Value))

	case domain.StartsWith:
		clause = fmt.Sprintf("%s LIKE %s", condition.Field, c.placeholder(argIndex))
		args = append(args, fmt.Sprintf("%v%%", condition.Value))

	case domain.EndsWith:
		clause = fmt.Sprintf("%s LIKE %s", condition.Field, c.placeholder(argIndex))
		args = append(args, fmt.Sprintf("%%%v", condition.Value))

	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", condition.Operator)
	}

	return clause, args, nil
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
