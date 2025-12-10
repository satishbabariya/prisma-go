// Package sqlgen provides WHERE clause building logic.
package sqlgen

import (
	"fmt"
	"strings"
)

// buildWhereRecursive builds a WHERE clause with support for nested conditions
func buildWhereRecursive(where *WhereClause, argIndex *int, placeholder func(int) string, quoter func(string) string) (string, []interface{}) {
	if where == nil || where.IsEmpty() {
		return "", nil
	}

	var parts []string
	var args []interface{}

	// Process direct conditions
	for _, cond := range where.Conditions {
		condSQL, condArgs := buildCondition(cond, argIndex, placeholder, quoter)
		if condSQL != "" {
			parts = append(parts, condSQL)
			args = append(args, condArgs...)
		}
	}

	// Process nested groups (recursive)
	for _, group := range where.Groups {
		groupSQL, groupArgs := buildWhereRecursive(group, argIndex, placeholder, quoter)
		if groupSQL != "" {
			// Wrap in parentheses for precedence (NOT is already handled inside buildWhereRecursive)
			parts = append(parts, fmt.Sprintf("(%s)", groupSQL))
			args = append(args, groupArgs...)
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	// Join with operator (AND/OR)
	op := "AND"
	if where.Operator == "OR" || where.Operator == "or" {
		op = "OR"
	}

	result := strings.Join(parts, " "+op+" ")
	
	// Apply NOT to the entire clause if needed
	if where.IsNot {
		result = "NOT (" + result + ")"
	}

	return result, args
}

// buildCondition builds a single condition
func buildCondition(cond Condition, argIndex *int, placeholder func(int) string, quoter func(string) string) (string, []interface{}) {
	var args []interface{}
	var sql string

	switch cond.Operator {
	case "=", "!=", ">", "<", ">=", "<=":
		sql = fmt.Sprintf("%s %s %s", quoter(cond.Field), cond.Operator, placeholder(*argIndex))
		args = append(args, cond.Value)
		(*argIndex)++

	case "IN":
		if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = placeholder(*argIndex)
				args = append(args, values[i])
				(*argIndex)++
			}
			sql = fmt.Sprintf("%s IN (%s)", quoter(cond.Field), strings.Join(placeholders, ", "))
		}

	case "NOT IN":
		if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = placeholder(*argIndex)
				args = append(args, values[i])
				(*argIndex)++
			}
			sql = fmt.Sprintf("%s NOT IN (%s)", quoter(cond.Field), strings.Join(placeholders, ", "))
		}

	case "LIKE":
		sql = fmt.Sprintf("%s LIKE %s", quoter(cond.Field), placeholder(*argIndex))
		args = append(args, cond.Value)
		(*argIndex)++

	case "IS NULL":
		sql = fmt.Sprintf("%s IS NULL", quoter(cond.Field))

	case "IS NOT NULL":
		sql = fmt.Sprintf("%s IS NOT NULL", quoter(cond.Field))
	}

	return sql, args
}

