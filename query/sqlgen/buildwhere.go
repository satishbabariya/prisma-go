// Package sqlgen provides WHERE clause building logic.
package sqlgen

import (
	"fmt"
	"strings"
)

// buildWhereRecursive builds a WHERE clause with support for nested conditions
func buildWhereRecursive(where *WhereClause, argIndex *int, placeholder func(int) string, quoter func(string) string, provider string) (string, []interface{}) {
	if where == nil || where.IsEmpty() {
		return "", nil
	}

	var parts []string
	var args []interface{}

	// Process direct conditions
	for _, cond := range where.Conditions {
		condSQL, condArgs := buildCondition(cond, argIndex, placeholder, quoter, provider)
		if condSQL != "" {
			parts = append(parts, condSQL)
			args = append(args, condArgs...)
		}
	}

	// Process nested groups (recursive)
	for _, group := range where.Groups {
		groupSQL, groupArgs := buildWhereRecursive(group, argIndex, placeholder, quoter, provider)
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
func buildCondition(cond Condition, argIndex *int, placeholder func(int) string, quoter func(string) string, provider string) (string, []interface{}) {
	var args []interface{}
	var sql string

	// Handle JSON filters first
	if cond.JsonType != "" {
		return buildJsonCondition(cond, argIndex, placeholder, quoter, provider)
	}

	switch cond.Operator {
	case "=", "!=", ">", "<", ">=", "<=":
		sql = fmt.Sprintf("%s %s %s", quoter(cond.Field), cond.Operator, placeholder(*argIndex))
		args = append(args, cond.Value)
		(*argIndex)++

	case "IN":
		// Check if it's a subquery
		if cond.IsSubquery {
			// Import the Subquery type from builder package
			// For now, we'll use type assertion with interface{}
			if subquery, ok := cond.Value.(interface {
				GetSQL() string
				GetArgs() []interface{}
			}); ok {
				sql = fmt.Sprintf("%s IN (%s)", quoter(cond.Field), subquery.GetSQL())
				args = append(args, subquery.GetArgs()...)
				// Update argIndex based on number of args in subquery
				*argIndex += len(subquery.GetArgs())
			}
		} else {
			// Try to convert value to []interface{}
			var values []interface{}
			switch v := cond.Value.(type) {
			case []interface{}:
				values = v
			case []int:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			case []string:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			case []float64:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			case []int64:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			}

			if len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = placeholder(*argIndex)
					args = append(args, values[i])
					(*argIndex)++
				}
				sql = fmt.Sprintf("%s IN (%s)", quoter(cond.Field), strings.Join(placeholders, ", "))
			}
		}

	case "NOT IN":
		// Check if it's a subquery
		if cond.IsSubquery {
			if subquery, ok := cond.Value.(interface {
				GetSQL() string
				GetArgs() []interface{}
			}); ok {
				sql = fmt.Sprintf("%s NOT IN (%s)", quoter(cond.Field), subquery.GetSQL())
				args = append(args, subquery.GetArgs()...)
				*argIndex += len(subquery.GetArgs())
			}
		} else {
			// Try to convert value to []interface{}
			var values []interface{}
			switch v := cond.Value.(type) {
			case []interface{}:
				values = v
			case []int:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			case []string:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			case []float64:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			case []int64:
				values = make([]interface{}, len(v))
				for i, val := range v {
					values[i] = val
				}
			}

			if len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = placeholder(*argIndex)
					args = append(args, values[i])
					(*argIndex)++
				}
				sql = fmt.Sprintf("%s NOT IN (%s)", quoter(cond.Field), strings.Join(placeholders, ", "))
			}
		}

	case "EXISTS":
		if cond.IsSubquery {
			if subquery, ok := cond.Value.(interface {
				GetSQL() string
				GetArgs() []interface{}
			}); ok {
				sql = fmt.Sprintf("EXISTS (%s)", subquery.GetSQL())
				args = append(args, subquery.GetArgs()...)
				*argIndex += len(subquery.GetArgs())
			}
		}

	case "NOT EXISTS":
		if cond.IsSubquery {
			if subquery, ok := cond.Value.(interface {
				GetSQL() string
				GetArgs() []interface{}
			}); ok {
				sql = fmt.Sprintf("NOT EXISTS (%s)", subquery.GetSQL())
				args = append(args, subquery.GetArgs()...)
				*argIndex += len(subquery.GetArgs())
			}
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

// buildJsonCondition builds JSON-specific conditions based on provider
func buildJsonCondition(cond Condition, argIndex *int, placeholder func(int) string, quoter func(string) string, provider string) (string, []interface{}) {
	var args []interface{}
	var sql string

	field := quoter(cond.Field)
	path := cond.JsonPath
	if path == "" {
		path = "$" // Default to root
	}

	switch provider {
	case "postgresql", "postgres":
		switch cond.JsonType {
		case "path":
			// PostgreSQL: field->>'$.path' = value
			// Convert JSON path to PostgreSQL path format
			pgPath := convertJsonPathToPostgres(path)
			sql = fmt.Sprintf("%s->>%s = %s", field, pgPath, placeholder(*argIndex))
			args = append(args, cond.Value)
			(*argIndex)++

		case "contains":
			// PostgreSQL: field @> '{"key": "value"}'::jsonb
			sql = fmt.Sprintf("%s @> %s::jsonb", field, placeholder(*argIndex))
			args = append(args, cond.Value)
			(*argIndex)++

		case "array_contains":
			// PostgreSQL: field @> '[value]'::jsonb
			sql = fmt.Sprintf("%s @> %s::jsonb", field, placeholder(*argIndex))
			args = append(args, cond.Value)
			(*argIndex)++

		case "has_key":
			// PostgreSQL: field ? 'key'
			sql = fmt.Sprintf("%s ? %s", field, placeholder(*argIndex))
			args = append(args, cond.Value)
			(*argIndex)++
		}

	case "mysql":
		switch cond.JsonType {
		case "path":
			// MySQL: JSON_EXTRACT(field, '$.path') = value
			sql = fmt.Sprintf("JSON_EXTRACT(%s, %s) = %s", field, placeholder(*argIndex), placeholder(*argIndex+1))
			args = append(args, path, cond.Value)
			(*argIndex) += 2

		case "contains":
			// MySQL: JSON_CONTAINS(field, 'value')
			sql = fmt.Sprintf("JSON_CONTAINS(%s, %s)", field, placeholder(*argIndex))
			args = append(args, cond.Value)
			(*argIndex)++

		case "array_contains":
			// MySQL: JSON_CONTAINS(field, 'value', '$.path')
			sql = fmt.Sprintf("JSON_CONTAINS(%s, %s, %s)", field, placeholder(*argIndex), placeholder(*argIndex+1))
			args = append(args, cond.Value, path)
			(*argIndex) += 2

		case "has_key":
			// MySQL: JSON_CONTAINS_PATH(field, 'one', '$.key')
			sql = fmt.Sprintf("JSON_CONTAINS_PATH(%s, 'one', %s)", field, placeholder(*argIndex))
			args = append(args, fmt.Sprintf("$.%s", cond.Value))
			(*argIndex)++
		}

	case "sqlite":
		switch cond.JsonType {
		case "path":
			// SQLite: json_extract(field, '$.path') = value
			sql = fmt.Sprintf("json_extract(%s, %s) = %s", field, placeholder(*argIndex), placeholder(*argIndex+1))
			args = append(args, path, cond.Value)
			(*argIndex) += 2

		case "contains":
			// SQLite: json_extract(field, '$') contains value (approximation)
			sql = fmt.Sprintf("json_extract(%s, '$') LIKE %s", field, placeholder(*argIndex))
			args = append(args, fmt.Sprintf("%%\"%v\"%%", cond.Value))
			(*argIndex)++

		case "array_contains":
			// SQLite: json_each with subquery
			sql = fmt.Sprintf("EXISTS (SELECT 1 FROM json_each(%s) WHERE value = %s)", field, placeholder(*argIndex))
			args = append(args, cond.Value)
			(*argIndex)++

		case "has_key":
			// SQLite: json_extract(field, '$.key') IS NOT NULL
			sql = fmt.Sprintf("json_extract(%s, %s) IS NOT NULL", field, placeholder(*argIndex))
			args = append(args, fmt.Sprintf("$.%s", cond.Value))
			(*argIndex)++
		}
	}

	return sql, args
}

// convertJsonPathToPostgres converts JSON path to PostgreSQL format
// e.g., "$.name" -> "'$.name'" or "$[0]" -> "'$[0]'"
func convertJsonPathToPostgres(path string) string {
	// PostgreSQL expects the path as a string literal
	return fmt.Sprintf("'%s'", path)
}
