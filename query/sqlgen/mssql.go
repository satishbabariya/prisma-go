// Package sqlgen provides SQL Server query generation.
package sqlgen

import (
	"fmt"
	"strings"
)

// SQLServerGenerator generates SQL Server (T-SQL) queries
type SQLServerGenerator struct{}

func (g *SQLServerGenerator) GenerateSelect(table string, columns []string, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

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

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("@p%d", i)
		}, quoteIdentifier, "sqlserver")
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

	// SQL Server uses TOP instead of LIMIT
	if limit != nil && *limit > 0 {
		// Wrap SELECT with TOP
		selectPart := parts[0]
		parts[0] = strings.Replace(selectPart, "SELECT", fmt.Sprintf("SELECT TOP %d", *limit), 1)
	}

	// OFFSET/FETCH (SQL Server 2012+)
	if offset != nil && *offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET %d ROWS", *offset))
		if limit != nil && *limit > 0 {
			parts = append(parts, fmt.Sprintf("FETCH NEXT %d ROWS ONLY", *limit))
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLServerGenerator) GenerateSelectWithJoins(table string, columns []string, joins []Join, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

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

	// JOINs
	for _, join := range joins {
		joinSQL := fmt.Sprintf("%s JOIN %s", join.Type, quoteIdentifier(join.Table))
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
			return fmt.Sprintf("@p%d", i)
		}, quoteIdentifier, "sqlserver")
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

	// SQL Server uses TOP instead of LIMIT
	if limit != nil && *limit > 0 {
		selectPart := parts[0]
		parts[0] = strings.Replace(selectPart, "SELECT", fmt.Sprintf("SELECT TOP %d", *limit), 1)
	}

	// OFFSET/FETCH
	if offset != nil && *offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET %d ROWS", *offset))
		if limit != nil && *limit > 0 {
			parts = append(parts, fmt.Sprintf("FETCH NEXT %d ROWS ONLY", *limit))
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// GenerateSelectWithCTE for SQL Server
func (g *SQLServerGenerator) GenerateSelectWithCTE(
	table string,
	columns []string,
	ctes []CTE,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	// SQL Server supports CTEs, use similar logic to PostgreSQL
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
			return fmt.Sprintf("@p%d", i)
		}, quoteIdentifier, "sqlserver")
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

	// SQL Server uses TOP instead of LIMIT
	if limit != nil && *limit > 0 {
		// Find SELECT part and modify it
		for i, part := range parts {
			if strings.HasPrefix(part, "SELECT") {
				parts[i] = strings.Replace(part, "SELECT", fmt.Sprintf("SELECT TOP %d", *limit), 1)
				break
			}
		}
	}

	// OFFSET/FETCH (SQL Server 2012+)
	if offset != nil && *offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET %d ROWS", *offset))
		if limit != nil && *limit > 0 {
			parts = append(parts, fmt.Sprintf("FETCH NEXT %d ROWS ONLY", *limit))
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// GenerateSelectWithWindows for SQL Server
func (g *SQLServerGenerator) GenerateSelectWithWindows(
	table string,
	columns []string,
	windowFuncs []WindowFunction,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	// SQL Server supports window functions, similar to PostgreSQL
	pgGen := &PostgresGenerator{}
	return pgGen.GenerateSelectWithWindows(table, columns, windowFuncs, joins, where, orderBy, limit, offset)
}

// GenerateSelectWithAggregates for SQL Server (stub - delegates to GenerateAggregate)
func (g *SQLServerGenerator) GenerateSelectWithAggregates(
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
	// For now, delegate to GenerateAggregate if we have aggregates
	// Full implementation would combine columns and aggregates
	if len(aggregates) > 0 {
		// Use PostgresGenerator's implementation as base (SQL Server is similar)
		pgGen := &PostgresGenerator{}
		return pgGen.GenerateSelectWithAggregates(table, columns, aggregates, joins, where, groupBy, having, orderBy, limit, offset)
	}
	// No aggregates, use regular select
	return g.GenerateSelect(table, columns, where, orderBy, limit, offset)
}

func (g *SQLServerGenerator) GenerateInsert(table string, columns []string, values []interface{}) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("INSERT INTO %s", quoteIdentifier(table)))

	// Columns
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = quoteIdentifier(col)
	}
	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))

	// VALUES
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("@p%d", argIndex)
		args = append(args, values[i])
		argIndex++
	}
	parts = append(parts, fmt.Sprintf("VALUES (%s)", strings.Join(placeholders, ", ")))

	// SQL Server OUTPUT clause (similar to PostgreSQL RETURNING)
	parts = append(parts, "OUTPUT INSERTED.*")

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLServerGenerator) GenerateUpdate(table string, set map[string]interface{}, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("UPDATE %s", quoteIdentifier(table)))

	// SET clause
	setParts := make([]string, 0, len(set))
	for col, val := range set {
		setParts = append(setParts, fmt.Sprintf("%s = @p%d", quoteIdentifier(col), argIndex))
		args = append(args, val)
		argIndex++
	}
	parts = append(parts, "SET "+strings.Join(setParts, ", "))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("@p%d", i)
		}, quoteIdentifier, "sqlserver")
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	// OUTPUT clause
	parts = append(parts, "OUTPUT INSERTED.*")

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLServerGenerator) GenerateDelete(table string, where *WhereClause) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	parts = append(parts, fmt.Sprintf("DELETE FROM %s", quoteIdentifier(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("@p%d", i)
		}, quoteIdentifier, "sqlserver")
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

func (g *SQLServerGenerator) GenerateUpsert(table string, columns []string, values []interface{}, updateColumns []string, conflictTarget []string) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SQL Server uses MERGE statement for upsert
	parts = append(parts, fmt.Sprintf("MERGE %s AS target", quoteIdentifier(table)))

	// USING clause (source data)
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = quoteIdentifier(col)
	}

	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("@p%d", argIndex)
		args = append(args, values[i])
		argIndex++
	}

	parts = append(parts, fmt.Sprintf("USING (SELECT %s) AS source (%s)",
		strings.Join(placeholders, ", "),
		strings.Join(quotedCols, ", ")))

	// ON clause (match condition)
	if len(conflictTarget) > 0 {
		onParts := make([]string, len(conflictTarget))
		for i, col := range conflictTarget {
			onParts[i] = fmt.Sprintf("target.%s = source.%s", quoteIdentifier(col), quoteIdentifier(col))
		}
		parts = append(parts, "ON "+strings.Join(onParts, " AND "))
	} else {
		// Default to primary key
		parts = append(parts, fmt.Sprintf("ON target.%s = source.%s", quoteIdentifier(columns[0]), quoteIdentifier(columns[0])))
	}

	// WHEN MATCHED (UPDATE)
	if len(updateColumns) > 0 {
		updateParts := make([]string, len(updateColumns))
		for i, col := range updateColumns {
			updateParts[i] = fmt.Sprintf("%s = source.%s", quoteIdentifier(col), quoteIdentifier(col))
		}
		parts = append(parts, fmt.Sprintf("WHEN MATCHED THEN UPDATE SET %s", strings.Join(updateParts, ", ")))
	}

	// WHEN NOT MATCHED (INSERT)
	insertCols := make([]string, len(columns))
	insertVals := make([]string, len(columns))
	for i, col := range columns {
		insertCols[i] = quoteIdentifier(col)
		insertVals[i] = fmt.Sprintf("source.%s", quoteIdentifier(col))
	}
	parts = append(parts, fmt.Sprintf("WHEN NOT MATCHED THEN INSERT (%s) VALUES (%s)",
		strings.Join(insertCols, ", "),
		strings.Join(insertVals, ", ")))

	// OUTPUT clause
	parts = append(parts, "OUTPUT INSERTED.*")

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

func (g *SQLServerGenerator) GenerateAggregate(table string, aggregates []AggregateFunction, where *WhereClause, groupBy *GroupBy, having *Having) *Query {
	var parts []string
	var args []interface{}
	argIndex := 1

	// SELECT aggregates
	aggParts := make([]string, len(aggregates))
	for i, agg := range aggregates {
		aggSQL := fmt.Sprintf("%s(%s) AS %s", agg.Function, quoteIdentifier(agg.Field), quoteIdentifier(agg.Alias))
		aggParts[i] = aggSQL
	}
	parts = append(parts, fmt.Sprintf("SELECT %s", strings.Join(aggParts, ", ")))

	// FROM table
	parts = append(parts, fmt.Sprintf("FROM %s", quoteIdentifier(table)))

	// WHERE clause
	if where != nil && !where.IsEmpty() {
		whereSQL, whereArgs := buildWhereRecursive(where, &argIndex, func(i int) string {
			return fmt.Sprintf("@p%d", i)
		}, quoteIdentifier, "sqlserver")
		if whereSQL != "" {
			parts = append(parts, "WHERE "+whereSQL)
			args = append(args, whereArgs...)
		}
	}

	// GROUP BY
	if groupBy != nil && len(groupBy.Fields) > 0 {
		groupParts := make([]string, len(groupBy.Fields))
		for i, field := range groupBy.Fields {
			groupParts[i] = quoteIdentifier(field)
		}
		parts = append(parts, "GROUP BY "+strings.Join(groupParts, ", "))
	}

	// HAVING
	if having != nil && len(having.Conditions) > 0 {
		havingWhere := &WhereClause{
			Conditions: having.Conditions,
			Operator:   having.Operator,
		}
		havingSQL, havingArgs := buildWhereRecursive(havingWhere, &argIndex, func(i int) string {
			return fmt.Sprintf("@p%d", i)
		}, quoteIdentifier, "sqlserver")
		if havingSQL != "" {
			parts = append(parts, "HAVING "+havingSQL)
			args = append(args, havingArgs...)
		}
	}

	return &Query{
		SQL:  strings.Join(parts, " "),
		Args: args,
	}
}

// quoteIdentifier quotes identifiers for SQL Server
func quoteIdentifierSQLServer(name string) string {
	return fmt.Sprintf("[%s]", name)
}

