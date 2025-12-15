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
		parts = append(parts, "LIMIT ?")
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		parts = append(parts, "OFFSET ?")
		args = append(args, *offset)
		argIndex++
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// GenerateSelectWithAggregates for MySQL
func (g *MySQLGenerator) GenerateSelectWithAggregates(
	table string,
	columns []string,
	aggregates []AggregateFunction,
	joins []Join,
	where *WhereClause,
	groupBy *GroupBy,
	having *Having,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// Build SELECT clause
	var selectParts []string
	if len(columns) > 0 {
		for _, col := range columns {
			selectParts = append(selectParts, quoteIdentifierMySQL(col))
		}
	}
	for _, agg := range aggregates {
		if agg.Field == "*" {
			if agg.Alias != "" {
				selectParts = append(selectParts, fmt.Sprintf("%s(*) AS %s", agg.Function, quoteIdentifierMySQL(agg.Alias)))
			} else {
				selectParts = append(selectParts, fmt.Sprintf("%s(*)", agg.Function))
			}
		} else {
			if agg.Alias != "" {
				selectParts = append(selectParts, fmt.Sprintf("%s(%s) AS %s", agg.Function, quoteIdentifierMySQL(agg.Field), quoteIdentifierMySQL(agg.Alias)))
			} else {
				selectParts = append(selectParts, fmt.Sprintf("%s(%s)", agg.Function, quoteIdentifierMySQL(agg.Field)))
			}
		}
	}
	if len(selectParts) == 0 {
		selectParts = append(selectParts, "*")
	}
	parts = append(parts, "SELECT "+strings.Join(selectParts, ", "))

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
		if havingSQL != "" {
			parts = append(parts, "HAVING "+havingSQL)
			args = append(args, havingArgs...)
		}
	}

	// ORDER BY
	if len(orderBy) > 0 {
		orderParts := make([]string, len(orderBy))
		for i, ob := range orderBy {
			direction := "ASC"
			if ob.Direction == "DESC" || ob.Direction == "desc" {
				direction = "DESC"
			}
			orderParts[i] = fmt.Sprintf("%s %s", quoteIdentifierMySQL(ob.Field), direction)
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	// LIMIT
	if limit != nil && *limit > 0 {
		parts = append(parts, "LIMIT ?")
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		if limit == nil || *limit == 0 {
			parts = append(parts, "LIMIT 18446744073709551615")
		}
		parts = append(parts, "OFFSET ?")
		args = append(args, *offset)
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
		parts = append(parts, "LIMIT ?")
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		parts = append(parts, "OFFSET ?")
		args = append(args, *offset)
		argIndex++
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// GenerateSelectWithAggregates for SQLite
func (g *SQLiteGenerator) GenerateSelectWithAggregates(
	table string,
	columns []string,
	aggregates []AggregateFunction,
	joins []Join,
	where *WhereClause,
	groupBy *GroupBy,
	having *Having,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// Build SELECT clause
	var selectParts []string
	if len(columns) > 0 {
		for _, col := range columns {
			selectParts = append(selectParts, quoteIdentifierSQLite(col))
		}
	}
	for _, agg := range aggregates {
		if agg.Field == "*" {
			if agg.Alias != "" {
				selectParts = append(selectParts, fmt.Sprintf("%s(*) AS %s", agg.Function, quoteIdentifierSQLite(agg.Alias)))
			} else {
				selectParts = append(selectParts, fmt.Sprintf("%s(*)", agg.Function))
			}
		} else {
			if agg.Alias != "" {
				selectParts = append(selectParts, fmt.Sprintf("%s(%s) AS %s", agg.Function, quoteIdentifierSQLite(agg.Field), quoteIdentifierSQLite(agg.Alias)))
			} else {
				selectParts = append(selectParts, fmt.Sprintf("%s(%s)", agg.Function, quoteIdentifierSQLite(agg.Field)))
			}
		}
	}
	if len(selectParts) == 0 {
		selectParts = append(selectParts, "*")
	}
	parts = append(parts, "SELECT "+strings.Join(selectParts, ", "))

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
		if havingSQL != "" {
			parts = append(parts, "HAVING "+havingSQL)
			args = append(args, havingArgs...)
		}
	}

	// ORDER BY
	if len(orderBy) > 0 {
		orderParts := make([]string, len(orderBy))
		for i, ob := range orderBy {
			direction := "ASC"
			if ob.Direction == "DESC" || ob.Direction == "desc" {
				direction = "DESC"
			}
			orderParts[i] = fmt.Sprintf("%s %s", quoteIdentifierSQLite(ob.Field), direction)
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	// LIMIT
	if limit != nil && *limit > 0 {
		parts = append(parts, "LIMIT ?")
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		parts = append(parts, "OFFSET ?")
		args = append(args, *offset)
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}
