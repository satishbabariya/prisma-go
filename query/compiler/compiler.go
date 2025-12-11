// Package compiler compiles query AST into SQL.
package compiler

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/query/ast"
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// Compiler compiles query AST into SQL
type Compiler struct {
	generator sqlgen.Generator
}

// NewCompiler creates a new query compiler
func NewCompiler(provider string) *Compiler {
	return &Compiler{
		generator: sqlgen.NewGenerator(provider),
	}
}

// Compile compiles a query node into SQL
func (c *Compiler) Compile(node ast.QueryNode) (string, []interface{}, error) {
	switch node.Type() {
	case ast.NodeTypeFindMany:
		query, ok := node.(*ast.FindManyQuery)
		if !ok {
			return "", nil, ErrInvalidQuery
		}
		return c.compileFindMany(query)
	case ast.NodeTypeFindFirst:
		query, ok := node.(*ast.FindManyQuery) // FindFirst uses same structure as FindMany with Take=1
		if !ok {
			return "", nil, ErrInvalidQuery
		}
		take := 1
		query.Take = &take
		return c.compileFindMany(query)
	case ast.NodeTypeFindUnique:
		query, ok := node.(*ast.FindManyQuery)
		if !ok {
			return "", nil, ErrInvalidQuery
		}
		take := 1
		query.Take = &take
		return c.compileFindMany(query)
	case ast.NodeTypeCreate:
		query, ok := node.(*ast.CreateQuery)
		if !ok {
			return "", nil, ErrInvalidQuery
		}
		return c.compileCreate(query)
	case ast.NodeTypeUpdate:
		query, ok := node.(*ast.UpdateQuery)
		if !ok {
			return "", nil, ErrInvalidQuery
		}
		return c.compileUpdate(query)
	case ast.NodeTypeDelete:
		query, ok := node.(*ast.DeleteQuery)
		if !ok {
			return "", nil, ErrInvalidQuery
		}
		return c.compileDelete(query)
	default:
		return "", nil, ErrUnsupportedQuery
	}
}

func (c *Compiler) compileFindMany(query *ast.FindManyQuery) (string, []interface{}, error) {
	// Build WHERE clause
	var where *sqlgen.WhereClause
	if query.Where != nil {
		where = c.buildWhereClause(query.Where)
	}

	// Build ORDER BY clause
	var orderBy []sqlgen.OrderBy
	if len(query.OrderBy) > 0 {
		orderBy = make([]sqlgen.OrderBy, len(query.OrderBy))
		for i, ob := range query.OrderBy {
			direction := "ASC"
			if ob.Direction == ast.SortDesc {
				direction = "DESC"
			}
			orderBy[i] = sqlgen.OrderBy{
				Field:     ob.Field,
				Direction: direction,
			}
		}
	}

	// Build columns list
	var columns []string
	if query.Select != nil && len(query.Select.Fields) > 0 {
		columns = query.Select.Fields
	}

	// Generate SQL
	sqlQuery := c.generator.GenerateSelect(query.Model, columns, where, orderBy, query.Take, query.Skip)
	return sqlQuery.SQL, sqlQuery.Args, nil
}

// buildWhereClause converts AST WhereClause to sqlgen WhereClause
func (c *Compiler) buildWhereClause(where *ast.WhereClause) *sqlgen.WhereClause {
	if where == nil || len(where.Conditions) == 0 {
		return nil
	}

	conditions := make([]sqlgen.Condition, len(where.Conditions))
	for i, cond := range where.Conditions {
		op := c.mapOperator(cond.Operator)
		value := cond.Value

		// Transform value for LIKE operators
		if op == "LIKE" {
			value = c.transformLikeValue(cond.Operator, cond.Value)
		}

		conditions[i] = sqlgen.Condition{
			Field:    cond.Field,
			Operator: op,
			Value:    value,
		}
	}

	return &sqlgen.WhereClause{
		Conditions: conditions,
		Operator:   string(where.Operator),
	}
}

// transformLikeValue transforms a value for LIKE operators based on the comparison operator
func (c *Compiler) transformLikeValue(op ast.ComparisonOperator, value interface{}) interface{} {
	if value == nil {
		return value
	}

	valueStr, ok := value.(string)
	if !ok {
		// Try to convert to string
		valueStr = fmt.Sprintf("%v", value)
	}

	switch op {
	case ast.OpContains:
		// Contains: add wildcards on both sides
		return fmt.Sprintf("%%%s%%", valueStr)
	case ast.OpStartsWith:
		// StartsWith: add wildcard at the end
		return fmt.Sprintf("%s%%", valueStr)
	case ast.OpEndsWith:
		// EndsWith: add wildcard at the beginning
		return fmt.Sprintf("%%%s", valueStr)
	default:
		return value
	}
}

// compileCreate compiles a Create query
func (c *Compiler) compileCreate(query *ast.CreateQuery) (string, []interface{}, error) {
	columns := make([]string, 0, len(query.Data))
	values := make([]interface{}, 0, len(query.Data))

	for col, val := range query.Data {
		columns = append(columns, col)
		values = append(values, val)
	}

	sqlQuery := c.generator.GenerateInsert(query.Model, columns, values)
	return sqlQuery.SQL, sqlQuery.Args, nil
}

// compileUpdate compiles an Update query
func (c *Compiler) compileUpdate(query *ast.UpdateQuery) (string, []interface{}, error) {
	var where *sqlgen.WhereClause
	if query.Where != nil {
		where = c.buildWhereClause(query.Where)
	}

	sqlQuery := c.generator.GenerateUpdate(query.Model, query.Data, where)
	return sqlQuery.SQL, sqlQuery.Args, nil
}

// compileDelete compiles a Delete query
func (c *Compiler) compileDelete(query *ast.DeleteQuery) (string, []interface{}, error) {
	var where *sqlgen.WhereClause
	if query.Where != nil {
		where = c.buildWhereClause(query.Where)
	}

	sqlQuery := c.generator.GenerateDelete(query.Model, where)
	return sqlQuery.SQL, sqlQuery.Args, nil
}

// mapOperator maps AST comparison operators to SQL operators
func (c *Compiler) mapOperator(op ast.ComparisonOperator) string {
	switch op {
	case ast.OpEquals:
		return "="
	case ast.OpNotEquals:
		return "!="
	case ast.OpGreaterThan:
		return ">"
	case ast.OpLessThan:
		return "<"
	case ast.OpGreaterOrEqual:
		return ">="
	case ast.OpLessOrEqual:
		return "<="
	case ast.OpIn:
		return "IN"
	case ast.OpNotIn:
		return "NOT IN"
	case ast.OpContains:
		return "LIKE"
	case ast.OpStartsWith:
		return "LIKE"
	case ast.OpEndsWith:
		return "LIKE"
	default:
		return "="
	}
}
