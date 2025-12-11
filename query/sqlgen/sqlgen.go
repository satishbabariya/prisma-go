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

// CTE represents a Common Table Expression
type CTE struct {
	Name      string
	Query     *Query
	Columns   []string // Optional column names for the CTE
	Recursive bool     // Whether this is a RECURSIVE CTE
}

// WindowFunction represents a window function
type WindowFunction struct {
	Function string
	Field    string
	Alias    string
	Window   *WindowDefinition
}

// WindowDefinition defines the window frame
type WindowDefinition struct {
	PartitionByFields []string
	OrderByFields     []OrderBy
	FrameSpec         *WindowFrame
}

// WindowFrame defines the window frame
type WindowFrame struct {
	Type          string // "ROWS" or "RANGE"
	Start         *FrameBound
	End           *FrameBound
	ExclusionType string
}

// FrameBound defines a frame boundary
type FrameBound struct {
	Type   string
	Offset *int
}

// Generator generates SQL for a specific provider
type Generator interface {
	GenerateSelect(table string, columns []string, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query
	GenerateSelectWithJoins(table string, columns []string, joins []Join, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query
	GenerateSelectWithAggregates(table string, columns []string, aggregates []AggregateFunction, joins []Join, where *WhereClause, groupBy *GroupBy, having *Having, orderBy []OrderBy, limit, offset *int) *Query
	GenerateSelectWithCTE(table string, columns []string, ctes []CTE, joins []Join, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query
	GenerateSelectWithWindows(table string, columns []string, windowFuncs []WindowFunction, joins []Join, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query
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
	case "sqlserver", "mssql":
		return &SQLServerGenerator{}
	case "cockroachdb":
		return &PostgresGenerator{} // CockroachDB is PostgreSQL-compatible
	case "mongodb":
		return &MongoDBGenerator{}
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

// GenerateSelectWithCTE generates a SELECT query with CTEs
func (g *PostgresGenerator) GenerateSelectWithCTE(
	table string,
	columns []string,
	ctes []CTE,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// Build WITH clause if CTEs are present
	if len(ctes) > 0 {
		withParts := []string{}
		isRecursive := false
		for _, cte := range ctes {
			if cte.Recursive {
				isRecursive = true
				break
			}
		}

		if isRecursive {
			withParts = append(withParts, "WITH RECURSIVE")
		} else {
			withParts = append(withParts, "WITH")
		}

		cteParts := []string{}
		for _, cte := range ctes {
			cteSQL := quoteIdentifier(cte.Name)
			if len(cte.Columns) > 0 {
				colParts := make([]string, len(cte.Columns))
				for j, col := range cte.Columns {
					colParts[j] = quoteIdentifier(col)
				}
				cteSQL += " (" + strings.Join(colParts, ", ") + ")"
			}
			cteSQL += " AS (" + cte.Query.SQL + ")"
			cteParts = append(cteParts, cteSQL)
			args = append(args, cte.Query.Args...)
		}
		withParts = append(withParts, strings.Join(cteParts, ", "))
		parts = append(parts, strings.Join(withParts, " "))
	}

	// SELECT columns
	if len(columns) == 0 {
		parts = append(parts, "SELECT *")
	} else {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifier(col)
		}
		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(quotedCols, ", ")))
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
		argIndex++
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// GenerateSelectWithWindows generates a SELECT query with window functions
func (g *PostgresGenerator) GenerateSelectWithWindows(
	table string,
	columns []string,
	windowFuncs []WindowFunction,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// Build SELECT clause with columns and window functions
	var selectParts []string

	// Add regular columns
	if len(columns) > 0 {
		for _, col := range columns {
			selectParts = append(selectParts, quoteIdentifier(col))
		}
	}

	// Add window functions
	for _, wf := range windowFuncs {
		funcSQL := g.buildWindowFunction(wf, &argIndex)
		if wf.Alias != "" {
			funcSQL += " AS " + quoteIdentifier(wf.Alias)
		}
		selectParts = append(selectParts, funcSQL)
	}

	// If no columns or window functions, select all
	if len(selectParts) == 0 {
		selectParts = append(selectParts, "*")
	}

	parts = append(parts, "SELECT "+strings.Join(selectParts, ", "))

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
		argIndex++
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// buildWindowFunction builds a window function SQL
func (g *PostgresGenerator) buildWindowFunction(wf WindowFunction, argIndex *int) string {
	var funcSQL string

	// Build function call
	switch wf.Function {
	case "ROW_NUMBER", "RANK", "DENSE_RANK":
		funcSQL = fmt.Sprintf("%s()", wf.Function)
	case "LAG", "LEAD":
		// LAG/LEAD need field, offset, and optional default
		if wf.Field != "" {
			funcSQL = fmt.Sprintf("%s(%s", wf.Function, quoteIdentifier(wf.Field))
			// Note: offset and default would need to be stored in WindowFunction
			// For now, simplified version
			funcSQL += ")"
		}
	case "FIRST_VALUE", "LAST_VALUE":
		if wf.Field != "" {
			funcSQL = fmt.Sprintf("%s(%s)", wf.Function, quoteIdentifier(wf.Field))
		}
	default:
		// SUM, AVG, COUNT, MAX, MIN
		if wf.Field == "*" {
			funcSQL = fmt.Sprintf("%s(*)", wf.Function)
		} else if wf.Field != "" {
			funcSQL = fmt.Sprintf("%s(%s)", wf.Function, quoteIdentifier(wf.Field))
		}
	}

	// Build OVER clause
	if wf.Window != nil {
		funcSQL += " OVER ("
		windowParts := []string{}

		// PARTITION BY
		if len(wf.Window.PartitionByFields) > 0 {
			partParts := make([]string, len(wf.Window.PartitionByFields))
			for i, part := range wf.Window.PartitionByFields {
				partParts[i] = quoteIdentifier(part)
			}
			windowParts = append(windowParts, "PARTITION BY "+strings.Join(partParts, ", "))
		}

		// ORDER BY
		if len(wf.Window.OrderByFields) > 0 {
			orderParts := make([]string, len(wf.Window.OrderByFields))
			for i, ob := range wf.Window.OrderByFields {
				direction := "ASC"
				if ob.Direction == "DESC" || ob.Direction == "desc" {
					direction = "DESC"
				}
				orderParts[i] = fmt.Sprintf("%s %s", quoteIdentifier(ob.Field), direction)
			}
			windowParts = append(windowParts, "ORDER BY "+strings.Join(orderParts, ", "))
		}

		// Frame
		if wf.Window.FrameSpec != nil {
			frameSQL := wf.Window.FrameSpec.Type
			if wf.Window.FrameSpec.Start != nil && wf.Window.FrameSpec.End != nil {
				frameSQL += " BETWEEN " + g.buildFrameBound(wf.Window.FrameSpec.Start) + " AND " + g.buildFrameBound(wf.Window.FrameSpec.End)
			} else if wf.Window.FrameSpec.Start != nil {
				frameSQL += " " + g.buildFrameBound(wf.Window.FrameSpec.Start)
			}
			if wf.Window.FrameSpec.ExclusionType != "" {
				frameSQL += " " + wf.Window.FrameSpec.ExclusionType
			}
			windowParts = append(windowParts, frameSQL)
		}

		funcSQL += strings.Join(windowParts, " ")
		funcSQL += ")"
	}

	return funcSQL
}

// buildFrameBound builds a frame boundary SQL
func (g *PostgresGenerator) buildFrameBound(bound *FrameBound) string {
	switch bound.Type {
	case "UNBOUNDED PRECEDING":
		return "UNBOUNDED PRECEDING"
	case "PRECEDING":
		if bound.Offset != nil {
			return fmt.Sprintf("%d PRECEDING", *bound.Offset)
		}
		return "PRECEDING"
	case "CURRENT ROW":
		return "CURRENT ROW"
	case "FOLLOWING":
		if bound.Offset != nil {
			return fmt.Sprintf("%d FOLLOWING", *bound.Offset)
		}
		return "FOLLOWING"
	case "UNBOUNDED FOLLOWING":
		return "UNBOUNDED FOLLOWING"
	default:
		return bound.Type
	}
}

// GenerateSelectWithAggregates generates a SELECT query with aggregations
func (g *PostgresGenerator) GenerateSelectWithAggregates(
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

	// Build SELECT clause with columns and aggregates
	var selectParts []string

	// Add regular columns
	if len(columns) > 0 {
		for _, col := range columns {
			selectParts = append(selectParts, quoteIdentifier(col))
		}
	}

	// Add aggregates
	for _, agg := range aggregates {
		if agg.Field == "*" {
			if agg.Alias != "" {
				selectParts = append(selectParts, fmt.Sprintf("%s(*) AS %s", agg.Function, quoteIdentifier(agg.Alias)))
			} else {
				selectParts = append(selectParts, fmt.Sprintf("%s(*)", agg.Function))
			}
		} else {
			if agg.Alias != "" {
				selectParts = append(selectParts, fmt.Sprintf("%s(%s) AS %s", agg.Function, quoteIdentifier(agg.Field), quoteIdentifier(agg.Alias)))
			} else {
				selectParts = append(selectParts, fmt.Sprintf("%s(%s)", agg.Function, quoteIdentifier(agg.Field)))
			}
		}
	}

	// If no columns or aggregates, select all
	if len(selectParts) == 0 {
		selectParts = append(selectParts, "*")
	}

	parts = append(parts, "SELECT "+strings.Join(selectParts, ", "))

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
		argIndex++
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
		}, quoteIdentifier, "postgresql")
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
		}, quoteIdentifier, "postgresql")
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
	}, quoteIdentifier, "postgresql")
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

// GenerateSelectWithCTE for MySQL
func (g *MySQLGenerator) GenerateSelectWithCTE(
	table string,
	columns []string,
	ctes []CTE,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// Build WITH clause if CTEs are present
	if len(ctes) > 0 {
		withParts := []string{}
		isRecursive := false
		for _, cte := range ctes {
			if cte.Recursive {
				isRecursive = true
				break
			}
		}

		if isRecursive {
			withParts = append(withParts, "WITH RECURSIVE")
		} else {
			withParts = append(withParts, "WITH")
		}

		cteParts := []string{}
		for _, cte := range ctes {
			cteSQL := quoteIdentifierMySQL(cte.Name)
			if len(cte.Columns) > 0 {
				colParts := make([]string, len(cte.Columns))
				for j, col := range cte.Columns {
					colParts[j] = quoteIdentifierMySQL(col)
				}
				cteSQL += " (" + strings.Join(colParts, ", ") + ")"
			}
			cteSQL += " AS (" + cte.Query.SQL + ")"
			cteParts = append(cteParts, cteSQL)
			args = append(args, cte.Query.Args...)
		}
		withParts = append(withParts, strings.Join(cteParts, ", "))
		parts = append(parts, strings.Join(withParts, " "))
	}

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

// GenerateSelectWithAggregates for MySQL - see implementation added earlier in file

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

// GenerateSelectWithWindows for MySQL
func (g *MySQLGenerator) GenerateSelectWithWindows(
	table string,
	columns []string,
	windowFuncs []WindowFunction,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	// MySQL 8.0+ supports window functions, similar to PostgreSQL
	pgGen := &PostgresGenerator{}
	return pgGen.GenerateSelectWithWindows(table, columns, windowFuncs, joins, where, orderBy, limit, offset)
}

// GenerateSelectWithAggregates for MySQL (already added above, this is just a marker)
// See implementation above after GenerateSelectWithJoins

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
		}, quoteIdentifier, "postgresql")
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
		}, quoteIdentifier, "postgresql")
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
	}, quoteIdentifierMySQL, "mysql")
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

// GenerateSelectWithCTE for SQLite
func (g *SQLiteGenerator) GenerateSelectWithCTE(
	table string,
	columns []string,
	ctes []CTE,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// Build WITH clause if CTEs are present
	if len(ctes) > 0 {
		withParts := []string{}
		isRecursive := false
		for _, cte := range ctes {
			if cte.Recursive {
				isRecursive = true
				break
			}
		}

		if isRecursive {
			withParts = append(withParts, "WITH RECURSIVE")
		} else {
			withParts = append(withParts, "WITH")
		}

		cteParts := []string{}
		for _, cte := range ctes {
			cteSQL := quoteIdentifierSQLite(cte.Name)
			if len(cte.Columns) > 0 {
				colParts := make([]string, len(cte.Columns))
				for j, col := range cte.Columns {
					colParts[j] = quoteIdentifierSQLite(col)
				}
				cteSQL += " (" + strings.Join(colParts, ", ") + ")"
			}
			cteSQL += " AS (" + cte.Query.SQL + ")"
			cteParts = append(cteParts, cteSQL)
			args = append(args, cte.Query.Args...)
		}
		withParts = append(withParts, strings.Join(cteParts, ", "))
		parts = append(parts, strings.Join(withParts, " "))
	}

	// SELECT columns
	if len(columns) == 0 {
		parts = append(parts, "SELECT *")
	} else {
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = quoteIdentifierSQLite(col)
		}
		parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(quotedCols, ", ")))
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

// GenerateSelectWithWindows for SQLite
func (g *SQLiteGenerator) GenerateSelectWithWindows(
	table string,
	columns []string,
	windowFuncs []WindowFunction,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	// SQLite 3.25.0+ supports window functions, similar to PostgreSQL
	pgGen := &PostgresGenerator{}
	return pgGen.GenerateSelectWithWindows(table, columns, windowFuncs, joins, where, orderBy, limit, offset)
}

// GenerateSelectWithAggregates for SQLite - see implementation added earlier in file

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
		}, quoteIdentifier, "postgresql")
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
		}, quoteIdentifier, "postgresql")
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
	}, quoteIdentifierSQLite, "sqlite")
}

func quoteIdentifierSQLite(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}
