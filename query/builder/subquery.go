// Package builder provides subquery building functionality
package builder

import (
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// Subquery represents a subquery that can be used in WHERE or SELECT clauses
type Subquery struct {
	SQL  string
	Args []interface{}
	// Metadata for result mapping
	SelectFields []string
	Table        string
}

// GetSQL returns the SQL string for the subquery
func (s *Subquery) GetSQL() string {
	return s.SQL
}

// GetArgs returns the arguments for the subquery
func (s *Subquery) GetArgs() []interface{} {
	return s.Args
}

// SubqueryBuilder builds subqueries
type SubqueryBuilder struct {
	table     string
	columns   []string
	where     *sqlgen.WhereClause
	orderBy   []sqlgen.OrderBy
	groupBy   []string
	having    *sqlgen.Having
	limit     *int
	offset    *int
	generator sqlgen.Generator
}

// NewSubqueryBuilder creates a new subquery builder
func NewSubqueryBuilder(table string, generator sqlgen.Generator) *SubqueryBuilder {
	return &SubqueryBuilder{
		table:     table,
		columns:   []string{},
		generator: generator,
	}
}

// Select sets the columns to select in the subquery
func (s *SubqueryBuilder) Select(columns ...string) *SubqueryBuilder {
	s.columns = columns
	return s
}

// Where sets the WHERE clause for the subquery
func (s *SubqueryBuilder) Where(where *sqlgen.WhereClause) *SubqueryBuilder {
	s.where = where
	return s
}

// OrderBy adds an ORDER BY clause to the subquery
func (s *SubqueryBuilder) OrderBy(field string, direction string) *SubqueryBuilder {
	s.orderBy = append(s.orderBy, sqlgen.OrderBy{
		Field:     field,
		Direction: direction,
	})
	return s
}

// GroupBy adds a GROUP BY clause to the subquery
func (s *SubqueryBuilder) GroupBy(fields ...string) *SubqueryBuilder {
	s.groupBy = append(s.groupBy, fields...)
	return s
}

// Having sets the HAVING clause for the subquery
func (s *SubqueryBuilder) Having(having *sqlgen.Having) *SubqueryBuilder {
	s.having = having
	return s
}

// Limit sets the LIMIT for the subquery
func (s *SubqueryBuilder) Limit(limit int) *SubqueryBuilder {
	s.limit = &limit
	return s
}

// Offset sets the OFFSET for the subquery
func (s *SubqueryBuilder) Offset(offset int) *SubqueryBuilder {
	s.offset = &offset
	return s
}

// Build builds the subquery and returns SQL and arguments
func (s *SubqueryBuilder) Build() *Subquery {
	// Use aggregate query if we have GROUP BY or HAVING
	if len(s.groupBy) > 0 || s.having != nil {
		aggregates := []sqlgen.AggregateFunction{}
		if len(s.columns) > 0 {
			// Convert columns to aggregate functions if needed
			for _, col := range s.columns {
				aggregates = append(aggregates, sqlgen.AggregateFunction{
					Function: "COUNT",
					Field:    col,
				})
			}
		} else {
			// Default to COUNT(*)
			aggregates = append(aggregates, sqlgen.AggregateFunction{
				Function: "COUNT",
				Field:    "*",
			})
		}

		groupBy := &sqlgen.GroupBy{
			Fields: s.groupBy,
		}

		query := s.generator.GenerateAggregate(s.table, aggregates, s.where, groupBy, s.having)
		return &Subquery{
			SQL:          query.SQL,
			Args:         query.Args,
			SelectFields: s.columns,
			Table:        s.table,
		}
	}

	// Regular SELECT query
	query := s.generator.GenerateSelect(s.table, s.columns, s.where, s.orderBy, s.limit, s.offset)
	return &Subquery{
		SQL:          query.SQL,
		Args:         query.Args,
		SelectFields: s.columns,
		Table:        s.table,
	}
}

// IN creates a condition that checks if a column is IN the subquery
func IN(column string, subquery *Subquery) sqlgen.Condition {
	return sqlgen.Condition{
		Field:       column,
		Operator:    "IN",
		Value:       subquery,
		IsSubquery:  true,
	}
}

// NOT_IN creates a condition that checks if a column is NOT IN the subquery
func NOT_IN(column string, subquery *Subquery) sqlgen.Condition {
	return sqlgen.Condition{
		Field:       column,
		Operator:    "NOT IN",
		Value:       subquery,
		IsSubquery:  true,
	}
}

// EXISTS creates a condition that checks if the subquery returns any rows
func EXISTS(subquery *Subquery) sqlgen.Condition {
	return sqlgen.Condition{
		Field:       "",
		Operator:    "EXISTS",
		Value:       subquery,
		IsSubquery:  true,
	}
}

// NOT_EXISTS creates a condition that checks if the subquery returns no rows
func NOT_EXISTS(subquery *Subquery) sqlgen.Condition {
	return sqlgen.Condition{
		Field:       "",
		Operator:    "NOT EXISTS",
		Value:       subquery,
		IsSubquery:  true,
	}
}

// ColumnIN creates a type-safe IN condition using a column
func ColumnIN(column interface{ Name() string }, subquery *Subquery) sqlgen.Condition {
	return IN(column.Name(), subquery)
}

// ColumnNOT_IN creates a type-safe NOT IN condition using a column
func ColumnNOT_IN(column interface{ Name() string }, subquery *Subquery) sqlgen.Condition {
	return NOT_IN(column.Name(), subquery)
}
