// Package compiler implements aggregation SQL generation.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// compileAggregate compiles an aggregation query.
func (c *SQLCompiler) compileAggregate(query *domain.Query) (domain.SQL, error) {
	var sqlBuilder strings.Builder
	var args []interface{}
	argIndex := 1

	// SELECT clause with aggregations
	sqlBuilder.WriteString("SELECT ")

	if len(query.Aggregations) == 0 {
		return domain.SQL{}, fmt.Errorf("aggregate query requires at least one aggregation")
	}

	// Build aggregation expressions
	for i, agg := range query.Aggregations {
		if i > 0 {
			sqlBuilder.WriteString(", ")
		}

		switch agg.Function {
		case domain.Count:
			if agg.Field == "" || agg.Field == "*" {
				sqlBuilder.WriteString("COUNT(*)")
			} else {
				sqlBuilder.WriteString(fmt.Sprintf("COUNT(%s)", agg.Field))
			}
		case domain.Sum:
			if agg.Field == "" {
				return domain.SQL{}, fmt.Errorf("SUM requires a field")
			}
			sqlBuilder.WriteString(fmt.Sprintf("SUM(%s)", agg.Field))
		case domain.Avg:
			if agg.Field == "" {
				return domain.SQL{}, fmt.Errorf("AVG requires a field")
			}
			sqlBuilder.WriteString(fmt.Sprintf("AVG(%s)", agg.Field))
		case domain.Min:
			if agg.Field == "" {
				return domain.SQL{}, fmt.Errorf("MIN requires a field")
			}
			sqlBuilder.WriteString(fmt.Sprintf("MIN(%s)", agg.Field))
		case domain.Max:
			if agg.Field == "" {
				return domain.SQL{}, fmt.Errorf("MAX requires a field")
			}
			sqlBuilder.WriteString(fmt.Sprintf("MAX(%s)", agg.Field))
		default:
			return domain.SQL{}, fmt.Errorf("unsupported aggregation function: %s", agg.Function)
		}

		// Add alias for the aggregation
		sqlBuilder.WriteString(fmt.Sprintf(" AS %s_%s", agg.Function, agg.Field))
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

	return domain.SQL{
		Query:   sqlBuilder.String(),
		Args:    args,
		Dialect: c.dialect,
	}, nil
}
