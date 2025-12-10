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
