// Package builder provides JOIN building functionality
package builder

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/query/columns"
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// JoinBuilder builds JOIN clauses
type JoinBuilder struct {
	joins []sqlgen.Join
}

// NewJoinBuilder creates a new JOIN builder
func NewJoinBuilder() *JoinBuilder {
	return &JoinBuilder{
		joins: []sqlgen.Join{},
	}
}

// InnerJoin adds an INNER JOIN
func (j *JoinBuilder) InnerJoin(table string, condition string) *JoinBuilder {
	j.joins = append(j.joins, sqlgen.Join{
		Type:      "INNER",
		Table:     table,
		Condition: condition,
	})
	return j
}

// LeftJoin adds a LEFT JOIN
func (j *JoinBuilder) LeftJoin(table string, condition string) *JoinBuilder {
	j.joins = append(j.joins, sqlgen.Join{
		Type:      "LEFT",
		Table:     table,
		Condition: condition,
	})
	return j
}

// RightJoin adds a RIGHT JOIN
func (j *JoinBuilder) RightJoin(table string, condition string) *JoinBuilder {
	j.joins = append(j.joins, sqlgen.Join{
		Type:      "RIGHT",
		Table:     table,
		Condition: condition,
	})
	return j
}

// FullJoin adds a FULL JOIN
func (j *JoinBuilder) FullJoin(table string, condition string) *JoinBuilder {
	j.joins = append(j.joins, sqlgen.Join{
		Type:      "FULL",
		Table:     table,
		Condition: condition,
	})
	return j
}

// CrossJoin adds a CROSS JOIN (no condition)
func (j *JoinBuilder) CrossJoin(table string) *JoinBuilder {
	j.joins = append(j.joins, sqlgen.Join{
		Type:      "CROSS",
		Table:     table,
		Condition: "",
	})
	return j
}

// Build returns the JOIN clauses
func (j *JoinBuilder) Build() []sqlgen.Join {
	return j.joins
}

// JoinCondition creates a join condition string from two columns
func JoinCondition(leftColumn columns.Column, operator string, rightColumn columns.Column) string {
	leftTable := leftColumn.Table()
	rightTable := rightColumn.Table()
	leftName := leftColumn.Name()
	rightName := rightColumn.Name()

	// Build condition: table1.column1 = table2.column2
	return fmt.Sprintf("%s.%s %s %s.%s",
		quoteIdentifier(leftTable),
		quoteIdentifier(leftName),
		operator,
		quoteIdentifier(rightTable),
		quoteIdentifier(rightName),
	)
}

// JoinConditionEQ creates an equality join condition
func JoinConditionEQ(leftColumn columns.Column, rightColumn columns.Column) string {
	return JoinCondition(leftColumn, "=", rightColumn)
}

// quoteIdentifier quotes an identifier (simplified - should use provider-specific quoting)
func quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

