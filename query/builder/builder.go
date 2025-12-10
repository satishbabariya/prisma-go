// Package builder provides a fluent query builder API.
package builder

import (
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// WhereBuilder builds WHERE clauses
type WhereBuilder struct {
	conditions  []sqlgen.Condition
	operator    string
	whereClause *sqlgen.WhereClause // For complex nested conditions
}

// NewWhereBuilder creates a new WHERE builder
func NewWhereBuilder() *WhereBuilder {
	return &WhereBuilder{
		conditions:  []sqlgen.Condition{},
		operator:    "AND",
		whereClause: nil,
	}
}

// Equals adds an equality condition
func (w *WhereBuilder) Equals(field string, value interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "=",
		Value:    value,
	})
	return w
}

// NotEquals adds a not-equals condition
func (w *WhereBuilder) NotEquals(field string, value interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "!=",
		Value:    value,
	})
	return w
}

// GreaterThan adds a greater-than condition
func (w *WhereBuilder) GreaterThan(field string, value interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: ">",
		Value:    value,
	})
	return w
}

// LessThan adds a less-than condition
func (w *WhereBuilder) LessThan(field string, value interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "<",
		Value:    value,
	})
	return w
}

// GreaterOrEqual adds a greater-or-equal condition
func (w *WhereBuilder) GreaterOrEqual(field string, value interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: ">=",
		Value:    value,
	})
	return w
}

// LessOrEqual adds a less-or-equal condition
func (w *WhereBuilder) LessOrEqual(field string, value interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "<=",
		Value:    value,
	})
	return w
}

// In adds an IN condition
func (w *WhereBuilder) In(field string, values []interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "IN",
		Value:    values,
	})
	return w
}

// NotIn adds a NOT IN condition
func (w *WhereBuilder) NotIn(field string, values []interface{}) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "NOT IN",
		Value:    values,
	})
	return w
}

// Like adds a LIKE condition
func (w *WhereBuilder) Like(field string, pattern string) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "LIKE",
		Value:    pattern,
	})
	return w
}

// IsNull adds an IS NULL condition
func (w *WhereBuilder) IsNull(field string) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "IS NULL",
		Value:    nil,
	})
	return w
}

// IsNotNull adds an IS NOT NULL condition
func (w *WhereBuilder) IsNotNull(field string) *WhereBuilder {
	w.conditions = append(w.conditions, sqlgen.Condition{
		Field:    field,
		Operator: "IS NOT NULL",
		Value:    nil,
	})
	return w
}

// SetOperator sets the logical operator (AND or OR)
func (w *WhereBuilder) SetOperator(op string) *WhereBuilder {
	w.operator = op
	return w
}

// Build builds the WHERE clause
func (w *WhereBuilder) Build() *sqlgen.WhereClause {
	// If we have a complex whereClause with groups, use it
	if w.whereClause != nil && !w.whereClause.IsEmpty() {
		// Add any direct conditions to the whereClause
		for _, cond := range w.conditions {
			w.whereClause.AddCondition(cond)
		}
		return w.whereClause
	}

	// Simple case: just direct conditions
	if len(w.conditions) == 0 {
		return nil
	}

	clause := sqlgen.NewWhereClause()
	clause.SetOperator(w.operator)
	for _, cond := range w.conditions {
		clause.AddCondition(cond)
	}
	return clause
}

// QueryBuilder builds complete queries
type QueryBuilder struct {
	table   string
	columns []string
	where   *sqlgen.WhereClause
	orderBy []sqlgen.OrderBy
	limit   *int
	offset  *int
}

// OrderByBuilder builds ORDER BY clauses
type OrderByBuilder struct {
	orderBy []sqlgen.OrderBy
}

// NewOrderByBuilder creates a new ORDER BY builder
func NewOrderByBuilder() *OrderByBuilder {
	return &OrderByBuilder{
		orderBy: []sqlgen.OrderBy{},
	}
}

// Asc adds an ascending ORDER BY clause
func (o *OrderByBuilder) Asc(field string) *OrderByBuilder {
	o.orderBy = append(o.orderBy, sqlgen.OrderBy{
		Field:     field,
		Direction: "ASC",
	})
	return o
}

// Desc adds a descending ORDER BY clause
func (o *OrderByBuilder) Desc(field string) *OrderByBuilder {
	o.orderBy = append(o.orderBy, sqlgen.OrderBy{
		Field:     field,
		Direction: "DESC",
	})
	return o
}

// Build returns the ORDER BY clauses
func (o *OrderByBuilder) Build() []sqlgen.OrderBy {
	return o.orderBy
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:   table,
		columns: nil, // nil means SELECT *
	}
}

// Select sets the columns to select
func (q *QueryBuilder) Select(columns ...string) *QueryBuilder {
	q.columns = columns
	return q
}

// Where sets the WHERE clause
func (q *QueryBuilder) Where(where *sqlgen.WhereClause) *QueryBuilder {
	q.where = where
	return q
}

// OrderBy adds an ORDER BY clause
func (q *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
	q.orderBy = append(q.orderBy, sqlgen.OrderBy{
		Field:     field,
		Direction: direction,
	})
	return q
}

// Limit sets the LIMIT
func (q *QueryBuilder) Limit(limit int) *QueryBuilder {
	q.limit = &limit
	return q
}

// Offset sets the OFFSET
func (q *QueryBuilder) Offset(offset int) *QueryBuilder {
	q.offset = &offset
	return q
}

// GetTable returns the table name
func (q *QueryBuilder) GetTable() string {
	return q.table
}

// GetColumns returns the columns
func (q *QueryBuilder) GetColumns() []string {
	return q.columns
}

// AggregateBuilder builds aggregation queries
type AggregateBuilder struct {
	table      string
	aggregates []sqlgen.AggregateFunction
	where      *sqlgen.WhereClause
	groupBy    *sqlgen.GroupBy
	having     *sqlgen.Having
}

// NewAggregateBuilder creates a new aggregation builder
func NewAggregateBuilder(table string) *AggregateBuilder {
	return &AggregateBuilder{
		table:      table,
		aggregates: []sqlgen.AggregateFunction{},
	}
}

// Count adds a COUNT(*) aggregation
func (a *AggregateBuilder) Count(alias string) *AggregateBuilder {
	a.aggregates = append(a.aggregates, sqlgen.AggregateFunction{
		Function: "COUNT",
		Field:    "*",
		Alias:    alias,
	})
	return a
}

// Sum adds a SUM(field) aggregation
func (a *AggregateBuilder) Sum(field string, alias string) *AggregateBuilder {
	a.aggregates = append(a.aggregates, sqlgen.AggregateFunction{
		Function: "SUM",
		Field:    field,
		Alias:    alias,
	})
	return a
}

// Avg adds an AVG(field) aggregation
func (a *AggregateBuilder) Avg(field string, alias string) *AggregateBuilder {
	a.aggregates = append(a.aggregates, sqlgen.AggregateFunction{
		Function: "AVG",
		Field:    field,
		Alias:    alias,
	})
	return a
}

// Min adds a MIN(field) aggregation
func (a *AggregateBuilder) Min(field string, alias string) *AggregateBuilder {
	a.aggregates = append(a.aggregates, sqlgen.AggregateFunction{
		Function: "MIN",
		Field:    field,
		Alias:    alias,
	})
	return a
}

// Max adds a MAX(field) aggregation
func (a *AggregateBuilder) Max(field string, alias string) *AggregateBuilder {
	a.aggregates = append(a.aggregates, sqlgen.AggregateFunction{
		Function: "MAX",
		Field:    field,
		Alias:    alias,
	})
	return a
}

// Where sets the WHERE clause
func (a *AggregateBuilder) Where(where *sqlgen.WhereClause) *AggregateBuilder {
	a.where = where
	return a
}

// GroupBy sets the GROUP BY clause
func (a *AggregateBuilder) GroupBy(fields ...string) *AggregateBuilder {
	a.groupBy = &sqlgen.GroupBy{
		Fields: fields,
	}
	return a
}

// Having sets the HAVING clause
func (a *AggregateBuilder) Having(conditions []sqlgen.Condition, operator string) *AggregateBuilder {
	if operator == "" {
		operator = "AND"
	}
	a.having = &sqlgen.Having{
		Conditions: conditions,
		Operator:   operator,
	}
	return a
}

// GetTable returns the table name
func (a *AggregateBuilder) GetTable() string {
	return a.table
}

// GetAggregates returns the aggregation functions
func (a *AggregateBuilder) GetAggregates() []sqlgen.AggregateFunction {
	return a.aggregates
}

// GetWhere returns the WHERE clause
func (a *AggregateBuilder) GetWhere() *sqlgen.WhereClause {
	return a.where
}

// GetGroupBy returns the GROUP BY clause
func (a *AggregateBuilder) GetGroupBy() *sqlgen.GroupBy {
	return a.groupBy
}

// GetHaving returns the HAVING clause
func (a *AggregateBuilder) GetHaving() *sqlgen.Having {
	return a.having
}

// GetWhere returns the WHERE clause
func (q *QueryBuilder) GetWhere() *sqlgen.WhereClause {
	return q.where
}

// GetOrderBy returns the ORDER BY clauses
func (q *QueryBuilder) GetOrderBy() []sqlgen.OrderBy {
	return q.orderBy
}

// GetLimit returns the LIMIT
func (q *QueryBuilder) GetLimit() *int {
	return q.limit
}

// GetOffset returns the OFFSET
func (q *QueryBuilder) GetOffset() *int {
	return q.offset
}
