// Package sqlgen provides aggregation query generation.
package sqlgen

import (
	"fmt"
	"strings"
)

// AggregateFunction represents an aggregation function
type AggregateFunction struct {
	Function string // "COUNT", "SUM", "AVG", "MIN", "MAX"
	Field    string // Field to aggregate on ("*" for COUNT(*))
	Alias    string // Alias for the result
}

// GroupBy represents a GROUP BY clause
type GroupBy struct {
	Fields []string
}

// Having represents a HAVING clause (similar to WHERE but for aggregates)
type Having struct {
	Conditions []Condition
	Operator   string // "AND" or "OR"
}

// GenerateAggregate generates an aggregation query
func (g *PostgresGenerator) GenerateAggregate(
	table string,
	aggregates []AggregateFunction,
	where *WhereClause,
	groupBy *GroupBy,
	having *Having,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT with aggregations
	selectParts := make([]string, len(aggregates))
	for i, agg := range aggregates {
		if agg.Field == "*" {
			selectParts[i] = fmt.Sprintf("%s(*) AS %s", agg.Function, quoteIdentifier(agg.Alias))
		} else {
			selectParts[i] = fmt.Sprintf("%s(%s) AS %s", agg.Function, quoteIdentifier(agg.Field), quoteIdentifier(agg.Alias))
		}
	}

	// Add GROUP BY fields to SELECT if present
	if groupBy != nil && len(groupBy.Fields) > 0 {
		for _, field := range groupBy.Fields {
			selectParts = append(selectParts, quoteIdentifier(field))
		}
	}

	parts = append(parts, "SELECT "+strings.Join(selectParts, ", "))

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifier(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
	}

	// GROUP BY
	if groupBy != nil && len(groupBy.Fields) > 0 {
		groupByParts := make([]string, len(groupBy.Fields))
		for i, field := range groupBy.Fields {
			groupByParts[i] = quoteIdentifier(field)
		}
		parts = append(parts, "GROUP BY "+strings.Join(groupByParts, ", "))
	}

	// HAVING
	if having != nil && len(having.Conditions) > 0 {
		havingSQL, havingArgs := g.buildHaving(having, &argIndex)
		parts = append(parts, "HAVING "+havingSQL)
		args = append(args, havingArgs...)
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// buildHaving builds a HAVING clause (similar to WHERE but for aggregates)
func (g *PostgresGenerator) buildHaving(having *Having, argIndex *int) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	op := "AND"
	if having.Operator == "OR" || having.Operator == "or" {
		op = "OR"
	}

	for _, cond := range having.Conditions {
		switch cond.Operator {
		case "=", "!=", ">", "<", ">=", "<=":
			conditions = append(conditions, fmt.Sprintf("%s %s $%d", cond.Field, cond.Operator, *argIndex))
			args = append(args, cond.Value)
			(*argIndex)++
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " "+op+" "), args
}

// MySQL version
func (g *MySQLGenerator) GenerateAggregate(
	table string,
	aggregates []AggregateFunction,
	where *WhereClause,
	groupBy *GroupBy,
	having *Having,
) *Query {
	var parts []string
	var args []interface{}

	// SELECT with aggregations
	selectParts := make([]string, len(aggregates))
	for i, agg := range aggregates {
		if agg.Field == "*" {
			selectParts[i] = fmt.Sprintf("%s(*) AS %s", agg.Function, quoteIdentifierMySQL(agg.Alias))
		} else {
			selectParts[i] = fmt.Sprintf("%s(%s) AS %s", agg.Function, quoteIdentifierMySQL(agg.Field), quoteIdentifierMySQL(agg.Alias))
		}
	}

	// Add GROUP BY fields to SELECT if present
	if groupBy != nil && len(groupBy.Fields) > 0 {
		for _, field := range groupBy.Fields {
			selectParts = append(selectParts, quoteIdentifierMySQL(field))
		}
	}

	parts = append(parts, "SELECT "+strings.Join(selectParts, ", "))

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifierMySQL(table)))

	// WHERE clause
	argIndex := 1
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
	}

	// GROUP BY
	if groupBy != nil && len(groupBy.Fields) > 0 {
		groupByParts := make([]string, len(groupBy.Fields))
		for i, field := range groupBy.Fields {
			groupByParts[i] = quoteIdentifierMySQL(field)
		}
		parts = append(parts, "GROUP BY "+strings.Join(groupByParts, ", "))
	}

	// HAVING
	if having != nil && len(having.Conditions) > 0 {
		havingSQL, havingArgs := g.buildHavingMySQL(having)
		parts = append(parts, "HAVING "+havingSQL)
		args = append(args, havingArgs...)
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *MySQLGenerator) buildHavingMySQL(having *Having) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	op := "AND"
	if having.Operator == "OR" || having.Operator == "or" {
		op = "OR"
	}

	for _, cond := range having.Conditions {
		switch cond.Operator {
		case "=", "!=", ">", "<", ">=", "<=":
			conditions = append(conditions, fmt.Sprintf("%s %s ?", cond.Field, cond.Operator))
			args = append(args, cond.Value)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " "+op+" "), args
}

// SQLite version
func (g *SQLiteGenerator) GenerateAggregate(
	table string,
	aggregates []AggregateFunction,
	where *WhereClause,
	groupBy *GroupBy,
	having *Having,
) *Query {
	var parts []string
	var args []interface{}

	// SELECT with aggregations
	selectParts := make([]string, len(aggregates))
	for i, agg := range aggregates {
		if agg.Field == "*" {
			selectParts[i] = fmt.Sprintf("%s(*) AS %s", agg.Function, quoteIdentifierSQLite(agg.Alias))
		} else {
			selectParts[i] = fmt.Sprintf("%s(%s) AS %s", agg.Function, quoteIdentifierSQLite(agg.Field), quoteIdentifierSQLite(agg.Alias))
		}
	}

	// Add GROUP BY fields to SELECT if present
	if groupBy != nil && len(groupBy.Fields) > 0 {
		for _, field := range groupBy.Fields {
			selectParts = append(selectParts, quoteIdentifierSQLite(field))
		}
	}

	parts = append(parts, "SELECT "+strings.Join(selectParts, ", "))

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifierSQLite(table)))

	// WHERE clause
	argIndex := 1
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
	}

	// GROUP BY
	if groupBy != nil && len(groupBy.Fields) > 0 {
		groupByParts := make([]string, len(groupBy.Fields))
		for i, field := range groupBy.Fields {
			groupByParts[i] = quoteIdentifierSQLite(field)
		}
		parts = append(parts, "GROUP BY "+strings.Join(groupByParts, ", "))
	}

	// HAVING
	if having != nil && len(having.Conditions) > 0 {
		havingSQL, havingArgs := g.buildHavingSQLite(having)
		parts = append(parts, "HAVING "+havingSQL)
		args = append(args, havingArgs...)
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLiteGenerator) buildHavingSQLite(having *Having) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	op := "AND"
	if having.Operator == "OR" || having.Operator == "or" {
		op = "OR"
	}

	for _, cond := range having.Conditions {
		switch cond.Operator {
		case "=", "!=", ">", "<", ">=", "<=":
			conditions = append(conditions, fmt.Sprintf("%s %s ?", cond.Field, cond.Operator))
			args = append(args, cond.Value)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " "+op+" "), args
}
