// Package compiler provides SQL compilation for mutations.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// CompileUpdate compiles an UPDATE query to SQL.
func (c *SQLCompiler) CompileUpdate(query *domain.Query) (string, []interface{}, error) {
	if len(query.UpdateData) == 0 {
		return "", nil, fmt.Errorf("update data cannot be empty")
	}

	if len(query.Filter.Conditions) == 0 {
		return "", nil, fmt.Errorf("UPDATE requires WHERE clause for safety")
	}

	var setClauses []string
	var args []interface{}
	paramCount := 1

	// Build SET clauses
	for col, val := range query.UpdateData {
		setClauses = append(setClauses, fmt.Sprintf(
			"%s = %s",
			col,
			c.placeholder(&paramCount),
		))
		args = append(args, val)
	}

	sql := fmt.Sprintf(
		"UPDATE %s SET %s",
		query.Model,
		strings.Join(setClauses, ", "),
	)

	// Add WHERE clause
	whereSQL, whereArgs, err := c.buildWhereClause(query.Filter, &paramCount)
	if err != nil {
		return "", nil, err
	}
	if whereSQL != "" {
		sql += " WHERE " + whereSQL
		args = append(args, whereArgs...)
	}

	return sql, args, nil
}

// CompileDelete compiles a DELETE query to SQL.
func (c *SQLCompiler) CompileDelete(query *domain.Query) (string, []interface{}, error) {
	if len(query.Filter.Conditions) == 0 {
		return "", nil, fmt.Errorf("DELETE requires WHERE clause for safety")
	}

	sql := fmt.Sprintf("DELETE FROM %s", query.Model)

	paramCount := 1
	// Add WHERE clause
	whereSQL, args, err := c.buildWhereClause(query.Filter, &paramCount)
	if err != nil {
		return "", nil, err
	}
	if whereSQL != "" {
		sql += " WHERE " + whereSQL
	}

	return sql, args, nil
}
