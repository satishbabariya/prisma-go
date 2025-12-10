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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
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

func (g *PostgresGenerator) buildWhere(where *WhereClause, argIndex *int) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	op := "AND"
	if where.Operator == "OR" || where.Operator == "or" {
		op = "OR"
	}

	for _, cond := range where.Conditions {
		switch cond.Operator {
		case "=", "!=", ">", "<", ">=", "<=":
			conditions = append(conditions, fmt.Sprintf("%s %s $%d", quoteIdentifier(cond.Field), cond.Operator, *argIndex))
			args = append(args, cond.Value)
			(*argIndex)++
		case "IN":
			if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = fmt.Sprintf("$%d", *argIndex)
					args = append(args, values[i])
					(*argIndex)++
				}
				conditions = append(conditions, fmt.Sprintf("%s IN (%s)", quoteIdentifier(cond.Field), strings.Join(placeholders, ", ")))
			}
		case "NOT IN":
			if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = fmt.Sprintf("$%d", *argIndex)
					args = append(args, values[i])
					(*argIndex)++
				}
				conditions = append(conditions, fmt.Sprintf("%s NOT IN (%s)", quoteIdentifier(cond.Field), strings.Join(placeholders, ", ")))
			}
		case "LIKE":
			conditions = append(conditions, fmt.Sprintf("%s LIKE $%d", quoteIdentifier(cond.Field), *argIndex))
			args = append(args, cond.Value)
			(*argIndex)++
		case "IS NULL":
			conditions = append(conditions, fmt.Sprintf("%s IS NULL", quoteIdentifier(cond.Field)))
		case "IS NOT NULL":
			conditions = append(conditions, fmt.Sprintf("%s IS NOT NULL", quoteIdentifier(cond.Field)))
		}
	}

	return strings.Join(conditions, " "+op+" "), args
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
	} else {
		// Safety: require WHERE clause for DELETE
		parts = append(parts, "WHERE 1=0")
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *MySQLGenerator) buildWhere(where *WhereClause, argIndex *int) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	op := "AND"
	if where.Operator == "OR" || where.Operator == "or" {
		op = "OR"
	}

	for _, cond := range where.Conditions {
		switch cond.Operator {
		case "=", "!=", ">", "<", ">=", "<=":
			conditions = append(conditions, fmt.Sprintf("%s %s ?", quoteIdentifierMySQL(cond.Field), cond.Operator))
			args = append(args, cond.Value)
		case "IN":
			if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = "?"
					args = append(args, values[i])
				}
				conditions = append(conditions, fmt.Sprintf("%s IN (%s)", quoteIdentifierMySQL(cond.Field), strings.Join(placeholders, ", ")))
			}
		case "NOT IN":
			if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = "?"
					args = append(args, values[i])
				}
				conditions = append(conditions, fmt.Sprintf("%s NOT IN (%s)", quoteIdentifierMySQL(cond.Field), strings.Join(placeholders, ", ")))
			}
		case "LIKE":
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", quoteIdentifierMySQL(cond.Field)))
			args = append(args, cond.Value)
		case "IS NULL":
			conditions = append(conditions, fmt.Sprintf("%s IS NULL", quoteIdentifierMySQL(cond.Field)))
		case "IS NOT NULL":
			conditions = append(conditions, fmt.Sprintf("%s IS NOT NULL", quoteIdentifierMySQL(cond.Field)))
		}
	}

	return strings.Join(conditions, " "+op+" "), args
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
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
	if where != nil && len(where.Conditions) > 0 {
		whereSQL, whereArgs := g.buildWhere(where, &argIndex)
		parts = append(parts, "WHERE "+whereSQL)
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
	} else {
		// Safety: require WHERE clause for DELETE
		parts = append(parts, "WHERE 1=0")
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLiteGenerator) buildWhere(where *WhereClause, argIndex *int) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	op := "AND"
	if where.Operator == "OR" || where.Operator == "or" {
		op = "OR"
	}

	for _, cond := range where.Conditions {
		switch cond.Operator {
		case "=", "!=", ">", "<", ">=", "<=":
			conditions = append(conditions, fmt.Sprintf("%s %s ?", quoteIdentifierSQLite(cond.Field), cond.Operator))
			args = append(args, cond.Value)
		case "IN":
			if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = "?"
					args = append(args, values[i])
				}
				conditions = append(conditions, fmt.Sprintf("%s IN (%s)", quoteIdentifierSQLite(cond.Field), strings.Join(placeholders, ", ")))
			}
		case "NOT IN":
			if values, ok := cond.Value.([]interface{}); ok && len(values) > 0 {
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = "?"
					args = append(args, values[i])
				}
				conditions = append(conditions, fmt.Sprintf("%s NOT IN (%s)", quoteIdentifierSQLite(cond.Field), strings.Join(placeholders, ", ")))
			}
		case "LIKE":
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", quoteIdentifierSQLite(cond.Field)))
			args = append(args, cond.Value)
		case "IS NULL":
			conditions = append(conditions, fmt.Sprintf("%s IS NULL", quoteIdentifierSQLite(cond.Field)))
		case "IS NOT NULL":
			conditions = append(conditions, fmt.Sprintf("%s IS NOT NULL", quoteIdentifierSQLite(cond.Field)))
		}
	}

	return strings.Join(conditions, " "+op+" "), args
}

func quoteIdentifierSQLite(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

