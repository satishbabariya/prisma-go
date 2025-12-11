// Package sqlgen provides JOIN query generation for relations.
package sqlgen

import (
	"fmt"
	"strings"
)

// Join represents a JOIN clause
type Join struct {
	Type      string   // "LEFT", "INNER", "RIGHT"
	Table     string   // Table to join
	Alias     string   // Table alias
	Condition string   // JOIN condition (e.g., "post.author_id = user.id")
	Columns   []string // Columns to select from this table (with prefix)
}

// GenerateSelectWithJoins generates a SELECT query with JOINs
func (g *PostgresGenerator) GenerateSelectWithJoins(
	table string,
	columns []string,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT columns with prefixes for JOINs
	if len(joins) > 0 {
		// Build column list with table prefixes
		var selectCols []string

		// Main table columns
		if len(columns) == 0 {
			// Select all columns from main table
			selectCols = append(selectCols, fmt.Sprintf("%s.*", quoteIdentifier(table)))
		} else {
			for _, col := range columns {
				selectCols = append(selectCols, fmt.Sprintf("%s.%s", quoteIdentifier(table), quoteIdentifier(col)))
			}
		}

		// Joined table columns
		for _, join := range joins {
			tableName := join.Table
			if join.Alias != "" {
				tableName = join.Alias
			}

			if len(join.Columns) > 0 {
				for _, col := range join.Columns {
					selectCols = append(selectCols, fmt.Sprintf("%s.%s", quoteIdentifier(tableName), quoteIdentifier(col)))
				}
			} else {
				// Select all columns from joined table
				selectCols = append(selectCols, fmt.Sprintf("%s.*", quoteIdentifier(tableName)))
			}
		}

		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(selectCols, ", ")))
	} else {
		// No JOINs, use simple SELECT
		if len(columns) == 0 {
			parts = append(parts, "SELECT *")
		} else {
			quotedCols := make([]string, len(columns))
			for i, col := range columns {
				quotedCols[i] = quoteIdentifier(col)
			}
			parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(quotedCols, ", ")))
		}
	}

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifier(table)))

	// JOIN clauses
	for _, join := range joins {
		joinType := strings.ToUpper(join.Type)
		if joinType == "" {
			joinType = "LEFT"
		}
		joinSQL := fmt.Sprintf("%s JOIN %s", joinType, quoteIdentifier(join.Table))
		if join.Alias != "" {
			joinSQL += " AS " + quoteIdentifier(join.Alias)
		}
		if join.Condition != "" {
			joinSQL += " ON " + join.Condition
		}
		parts = append(parts, joinSQL)
	}

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier, "postgresql")
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	// ORDER BY
	if len(orderBy) > 0 {
		orderParts := make([]string, len(orderBy))
		for i, ob := range orderBy {
			orderParts[i] = fmt.Sprintf("%s %s", quoteIdentifier(ob.Field), ob.Direction)
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	// LIMIT
	if limit != nil && *limit > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT $%d", argIndex))
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET $%d", argIndex))
		args = append(args, *offset)
		argIndex++
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// GenerateSelectWithJoins for MySQL
func (g *MySQLGenerator) GenerateSelectWithJoins(
	table string,
	columns []string,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT columns
	if len(columns) == 0 {
		parts = append(parts, "SELECT *")
	} else {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifierMySQL(col)
		}
		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(quotedCols, ", ")))
	}

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifierMySQL(table)))

	// JOIN clauses
	for _, join := range joins {
		joinType := strings.ToUpper(join.Type)
		if joinType == "" {
			joinType = "LEFT"
		}
		joinSQL := fmt.Sprintf("%s JOIN %s", joinType, quoteIdentifierMySQL(join.Table))
		if join.Alias != "" {
			joinSQL += " AS " + quoteIdentifierMySQL(join.Alias)
		}
		if join.Condition != "" {
			joinSQL += " ON " + join.Condition
		}
		parts = append(parts, joinSQL)
	}

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return "?"
		}, quoteIdentifierMySQL, "mysql")
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	// ORDER BY
	if len(orderBy) > 0 {
		orderParts := make([]string, len(orderBy))
		for i, ob := range orderBy {
			orderParts[i] = fmt.Sprintf("%s %s", quoteIdentifierMySQL(ob.Field), ob.Direction)
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	// LIMIT
	if limit != nil && *limit > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT ?", argIndex))
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET ?", argIndex))
		args = append(args, *offset)
		argIndex++
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// GenerateSelectWithJoins for SQLite
func (g *SQLiteGenerator) GenerateSelectWithJoins(
	table string,
	columns []string,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT columns with prefixes for JOINs
	if len(joins) > 0 {
		// Build column list with table prefixes
		var selectCols []string

		// Main table columns
		if len(columns) == 0 {
			// Select all columns from main table
			selectCols = append(selectCols, fmt.Sprintf("%s.*", quoteIdentifierSQLite(table)))
		} else {
			for _, col := range columns {
				selectCols = append(selectCols, fmt.Sprintf("%s.%s", quoteIdentifierSQLite(table), quoteIdentifierSQLite(col)))
			}
		}

		// Joined table columns
		for _, join := range joins {
			tableName := join.Table
			if join.Alias != "" {
				tableName = join.Alias
			}

			if len(join.Columns) > 0 {
				for _, col := range join.Columns {
					selectCols = append(selectCols, fmt.Sprintf("%s.%s", quoteIdentifierSQLite(tableName), quoteIdentifierSQLite(col)))
				}
			} else {
				// Select all columns from joined table
				selectCols = append(selectCols, fmt.Sprintf("%s.*", quoteIdentifierSQLite(tableName)))
			}
		}

		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(selectCols, ", ")))
	} else {
		// No JOINs, use simple SELECT
		if len(columns) == 0 {
			parts = append(parts, "SELECT *")
		} else {
			quotedCols := make([]string, len(columns))
			for i, col := range columns {
				quotedCols[i] = quoteIdentifierSQLite(col)
			}
			parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(quotedCols, ", ")))
		}
	}

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifierSQLite(table)))

	// JOIN clauses
	for _, join := range joins {
		joinType := strings.ToUpper(join.Type)
		if joinType == "" {
			joinType = "LEFT"
		}
		joinSQL := fmt.Sprintf("%s JOIN %s", joinType, quoteIdentifierSQLite(join.Table))
		if join.Alias != "" {
			joinSQL += " AS " + quoteIdentifierSQLite(join.Alias)
		}
		if join.Condition != "" {
			joinSQL += " ON " + join.Condition
		}
		parts = append(parts, joinSQL)
	}

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return "?"
		}, quoteIdentifierSQLite, "sqlite")
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	// ORDER BY
	if len(orderBy) > 0 {
		orderParts := make([]string, len(orderBy))
		for i, ob := range orderBy {
			orderParts[i] = fmt.Sprintf("%s %s", quoteIdentifierSQLite(ob.Field), ob.Direction)
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	// LIMIT
	if limit != nil && *limit > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT ?", argIndex))
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET ?", argIndex))
		args = append(args, *offset)
		argIndex++
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}
