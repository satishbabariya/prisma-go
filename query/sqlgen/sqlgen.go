// Package sqlgen generates SQL for different database providers.
package sqlgen

import (
	"fmt"
	"strings"
)

// Query represents a SQL query with arguments
type Query struct {
	SQL  string
	Args []interface{}
}

// Generator generates SQL for a specific provider
type Generator interface {
	GenerateSelect(table string, columns []string, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query
	GenerateSelectWithJoins(table string, columns []string, joins []Join, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query
	GenerateInsert(table string, columns []string, values []interface{}) *Query
	GenerateUpdate(table string, set map[string]interface{}, where *WhereClause) *Query
	GenerateDelete(table string, where *WhereClause) *Query
	GenerateAggregate(table string, aggregates []AggregateFunction, where *WhereClause, groupBy *GroupBy, having *Having) *Query
	GenerateUpsert(table string, columns []string, values []interface{}, updateColumns []string, conflictTarget []string) *Query
}

// OrderBy represents an ORDER BY clause
type OrderBy struct {
	Field     string
	Direction string // "ASC" or "DESC"
}

// NewGenerator creates a new SQL generator for the given provider
func NewGenerator(provider string) Generator {
	switch provider {
	case "postgresql", "postgres":
		return &PostgresGenerator{}
	case "mysql":
		return &MySQLGenerator{}
	case "sqlite":
		return &SQLiteGenerator{}
	default:
		return &PostgresGenerator{} // default to postgres
	}
}

// PostgresGenerator generates PostgreSQL SQL
type PostgresGenerator struct{}

func (g *PostgresGenerator) GenerateSelect(table string, columns []string, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT columns
	if len(columns) == 0 {
		parts = append(parts, "SELECT *")
	} else {
		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(columns, ", ")))
	}

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifier(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
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
			orderParts[i] = fmt.Sprintf("%s %s", quoteIdentifier(ob.Field), direction)
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
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *PostgresGenerator) GenerateInsert(table string, columns []string, values []interface{}) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("INSERT INTO %s", quoteIdentifier(table)))

	// Columns
	if len(columns) > 0 {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifier(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
	}

	// VALUES
	if len(values) > 0 {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, values[i])
			argIndex++
		}
		parts = append(parts, fmt.Sprintf("VALUES (%s)", strings.Join(placeholders, ", ")))
	}

	// RETURNING * for PostgreSQL
	parts = append(parts, "RETURNING *")

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *PostgresGenerator) GenerateUpsert(table string, columns []string, values []interface{}, updateColumns []string, conflictTarget []string) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// INSERT INTO
	parts = append(parts, fmt.Sprintf("INSERT INTO %s", quoteIdentifier(table)))

	// Columns
	if len(columns) > 0 {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifier(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
	}

	// VALUES
	if len(values) > 0 {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, values[i])
			argIndex++
		}
		parts = append(parts, fmt.Sprintf("VALUES (%s)", strings.Join(placeholders, ", ")))
	}

	// ON CONFLICT ... DO UPDATE
	if len(conflictTarget) > 0 {
		conflictCols := make([]string, len(conflictTarget))
		for i, col := range conflictTarget {
			conflictCols[i] = quoteIdentifier(col)
		}
		parts = append(parts, fmt.Sprintf("ON CONFLICT (%s)", strings.Join(conflictCols, ", ")))

		// DO UPDATE SET
		if len(updateColumns) > 0 {
			setParts := make([]string, 0, len(updateColumns))
			for _, col := range updateColumns {
				setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", quoteIdentifier(col), quoteIdentifier(col)))
			}
			parts = append(parts, "DO UPDATE SET "+strings.Join(setParts, ", "))
		} else {
			// Update all columns except conflict target
			setParts := make([]string, 0, len(columns))
			conflictSet := make(map[string]bool)
			for _, col := range conflictTarget {
				conflictSet[col] = true
			}
			for _, col := range columns {
				if !conflictSet[col] {
					setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", quoteIdentifier(col), quoteIdentifier(col)))
				}
			}
			if len(setParts) > 0 {
				parts = append(parts, "DO UPDATE SET "+strings.Join(setParts, ", "))
			}
		}
	}

	// RETURNING * for PostgreSQL
	parts = append(parts, "RETURNING *")

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *PostgresGenerator) GenerateUpdate(table string, set map[string]interface{}, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("UPDATE %s", quoteIdentifier(table)))

	// SET clause
	if len(set) > 0 {
		setParts := make([]string, 0, len(set))
		for col, val := range set {
			setParts = append(setParts, fmt.Sprintf("%s = $%d", quoteIdentifier(col), argIndex))
			args = append(args, val)
			argIndex++
		}
		parts = append(parts, "SET "+strings.Join(setParts, ", "))
	}

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	// RETURNING * for PostgreSQL
	parts = append(parts, "RETURNING *")

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *PostgresGenerator) GenerateDelete(table string, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("DELETE FROM %s", quoteIdentifier(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	} else {
		// Safety: require WHERE clause for DELETE
		parts = append(parts, "WHERE 1=0") // Prevent accidental deletion of all rows
	}

	// RETURNING * for PostgreSQL
	parts = append(parts, "RETURNING *")

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// buildWhere is deprecated - use buildWhereRecursive instead
// Kept for backward compatibility but delegates to buildWhereRecursive
func (g *PostgresGenerator) buildWhere(where *WhereClause, argIndex *int) (string, []interface{}) {
	return buildWhereRecursive(where, argIndex, func(i int) string {
		return fmt.Sprintf("$%d", i)
	}, quoteIdentifier)
}

// quoteIdentifier quotes an identifier for PostgreSQL
func quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// MySQLGenerator generates MySQL SQL
type MySQLGenerator struct{}

func (g *MySQLGenerator) GenerateSelect(table string, columns []string, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT columns
	if len(columns) == 0 {
		parts = append(parts, "SELECT *")
	} else {
		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(columns, ", ")))
	}

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifierMySQL(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
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
		parts = append(parts, fmt.Sprintf("LIMIT ?"))
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		if limit == nil || *limit == 0 {
			// MySQL requires LIMIT when using OFFSET
			parts = append(parts, "LIMIT 18446744073709551615") // Max uint64
		}
		parts = append(parts, fmt.Sprintf("OFFSET ?"))
		args = append(args, *offset)
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *MySQLGenerator) GenerateInsert(table string, columns []string, values []interface{}) *Query {
	var parts []string
	var args []interface{}

	parts = append(parts, fmt.Sprintf("INSERT INTO %s", quoteIdentifierMySQL(table)))

	// Columns
	if len(columns) > 0 {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifierMySQL(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
	}

	// VALUES
	if len(values) > 0 {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
			args = append(args, values[i])
		}
		parts = append(parts, fmt.Sprintf("VALUES (%s)", strings.Join(placeholders, ", ")))
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *MySQLGenerator) GenerateUpsert(table string, columns []string, values []interface{}, updateColumns []string, conflictTarget []string) *Query {
	var parts []string
	var args []interface{}

	// INSERT INTO
	parts = append(parts, fmt.Sprintf("INSERT INTO %s", quoteIdentifierMySQL(table)))

	// Columns
	if len(columns) > 0 {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifierMySQL(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
	}

	// VALUES
	if len(values) > 0 {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
			args = append(args, values[i])
		}
		parts = append(parts, fmt.Sprintf("VALUES (%s)", strings.Join(placeholders, ", ")))
	}

	// ON DUPLICATE KEY UPDATE
	if len(conflictTarget) > 0 || len(updateColumns) > 0 {
		setParts := make([]string, 0)
		if len(updateColumns) > 0 {
			// Update specified columns
			for _, col := range updateColumns {
				setParts = append(setParts, fmt.Sprintf("%s = VALUES(%s)", quoteIdentifierMySQL(col), quoteIdentifierMySQL(col)))
			}
		} else {
			// Update all columns except conflict target
			conflictSet := make(map[string]bool)
			for _, col := range conflictTarget {
				conflictSet[col] = true
			}
			for _, col := range columns {
				if !conflictSet[col] {
					setParts = append(setParts, fmt.Sprintf("%s = VALUES(%s)", quoteIdentifierMySQL(col), quoteIdentifierMySQL(col)))
				}
			}
		}
		if len(setParts) > 0 {
			parts = append(parts, "ON DUPLICATE KEY UPDATE "+strings.Join(setParts, ", "))
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *MySQLGenerator) GenerateUpdate(table string, set map[string]interface{}, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("UPDATE %s", quoteIdentifierMySQL(table)))

	// SET clause
	if len(set) > 0 {
		setParts := make([]string, 0, len(set))
		for col, val := range set {
			setParts = append(setParts, fmt.Sprintf("%s = ?", quoteIdentifierMySQL(col)))
			args = append(args, val)
		}
		parts = append(parts, "SET "+strings.Join(setParts, ", "))
	}

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *MySQLGenerator) GenerateDelete(table string, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("DELETE FROM %s", quoteIdentifierMySQL(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	} else {
		// Safety: require WHERE clause for DELETE
		parts = append(parts, "WHERE 1=0")
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// buildWhere is deprecated - use buildWhereRecursive instead
// Kept for backward compatibility but delegates to buildWhereRecursive
func (g *MySQLGenerator) buildWhere(where *WhereClause, argIndex *int) (string, []interface{}) {
	return buildWhereRecursive(where, argIndex, func(i int) string {
		return "?"
	}, quoteIdentifierMySQL)
}

func quoteIdentifierMySQL(name string) string {
	return fmt.Sprintf("`%s`", name)
}

// SQLiteGenerator generates SQLite SQL
type SQLiteGenerator struct{}

func (g *SQLiteGenerator) GenerateSelect(table string, columns []string, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT columns
	if len(columns) == 0 {
		parts = append(parts, "SELECT *")
	} else {
		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(columns, ", ")))
	}

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifierSQLite(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
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
		parts = append(parts, fmt.Sprintf("LIMIT ?"))
		args = append(args, *limit)
		argIndex++
	}

	// OFFSET
	if offset != nil && *offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET ?"))
		args = append(args, *offset)
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLiteGenerator) GenerateInsert(table string, columns []string, values []interface{}) *Query {
	var parts []string
	var args []interface{}

	parts = append(parts, fmt.Sprintf("INSERT INTO %s", quoteIdentifierSQLite(table)))

	// Columns
	if len(columns) > 0 {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifierSQLite(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
	}

	// VALUES
	if len(values) > 0 {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
			args = append(args, values[i])
		}
		parts = append(parts, fmt.Sprintf("VALUES (%s)", strings.Join(placeholders, ", ")))
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLiteGenerator) GenerateUpsert(table string, columns []string, values []interface{}, updateColumns []string, conflictTarget []string) *Query {
	var parts []string
	var args []interface{}

	// INSERT INTO
	parts = append(parts, fmt.Sprintf("INSERT INTO %s", quoteIdentifierSQLite(table)))

	// Columns
	if len(columns) > 0 {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifierSQLite(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
	}

	// VALUES
	if len(values) > 0 {
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
			args = append(args, values[i])
		}
		parts = append(parts, fmt.Sprintf("VALUES (%s)", strings.Join(placeholders, ", ")))
	}

	// ON CONFLICT ... DO UPDATE
	if len(conflictTarget) > 0 {
		conflictCols := make([]string, len(conflictTarget))
		for i, col := range conflictTarget {
			conflictCols[i] = quoteIdentifierSQLite(col)
		}
		parts = append(parts, fmt.Sprintf("ON CONFLICT (%s)", strings.Join(conflictCols, ", ")))

		// DO UPDATE SET
		if len(updateColumns) > 0 {
			setParts := make([]string, 0, len(updateColumns))
			for _, col := range updateColumns {
				setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", quoteIdentifierSQLite(col), quoteIdentifierSQLite(col)))
			}
			parts = append(parts, "DO UPDATE SET "+strings.Join(setParts, ", "))
		} else {
			// Update all columns except conflict target
			setParts := make([]string, 0, len(columns))
			conflictSet := make(map[string]bool)
			for _, col := range conflictTarget {
				conflictSet[col] = true
			}
			for _, col := range columns {
				if !conflictSet[col] {
					setParts = append(setParts, fmt.Sprintf("%s = EXCLUDED.%s", quoteIdentifierSQLite(col), quoteIdentifierSQLite(col)))
				}
			}
			if len(setParts) > 0 {
				parts = append(parts, "DO UPDATE SET "+strings.Join(setParts, ", "))
			}
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLiteGenerator) GenerateUpdate(table string, set map[string]interface{}, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("UPDATE %s", quoteIdentifierSQLite(table)))

	// SET clause
	if len(set) > 0 {
		setParts := make([]string, 0, len(set))
		for col, val := range set {
			setParts = append(setParts, fmt.Sprintf("%s = ?", quoteIdentifierSQLite(col)))
			args = append(args, val)
		}
		parts = append(parts, "SET "+strings.Join(setParts, ", "))
	}

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLiteGenerator) GenerateDelete(table string, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("DELETE FROM %s", quoteIdentifierSQLite(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("$%d", i)
		}, quoteIdentifier)
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	} else {
		// Safety: require WHERE clause for DELETE
		parts = append(parts, "WHERE 1=0")
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// buildWhere is deprecated - use buildWhereRecursive instead
// Kept for backward compatibility but delegates to buildWhereRecursive
func (g *SQLiteGenerator) buildWhere(where *WhereClause, argIndex *int) (string, []interface{}) {
	return buildWhereRecursive(where, argIndex, func(i int) string {
		return "?"
	}, quoteIdentifierSQLite)
}

func quoteIdentifierSQLite(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}
