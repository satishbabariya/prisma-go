// Package compiler compiles CREATE MANY (batch insert) operations to SQL.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// CompileCreateMany compiles a CREATE MANY query to batch INSERT SQL.
func (c *SQLCompiler) CompileCreateMany(query *domain.Query) (string, []interface{}, error) {
	if len(query.CreateManyData) == 0 {
		return "", nil, fmt.Errorf("create many data cannot be empty")
	}

	// Get columns from first row (all rows must have same columns)
	var columns []string
	for col := range query.CreateManyData[0] {
		columns = append(columns, col)
	}

	// Validate all rows have same columns
	for i, row := range query.CreateManyData {
		if len(row) != len(columns) {
			return "", nil, fmt.Errorf("row %d has different number of columns", i)
		}
		for _, col := range columns {
			if _, exists := row[col]; !exists {
				return "", nil, fmt.Errorf("row %d missing column %s", i, col)
			}
		}
	}

	var args []interface{}
	var valueClauses []string
	paramCount := 1

	// Build VALUES clauses for each row
	for _, row := range query.CreateManyData {
		var placeholders []string
		for _, col := range columns {
			placeholders = append(placeholders, c.placeholder(&paramCount))
			args = append(args, row[col])
		}
		valueClauses = append(valueClauses, "("+strings.Join(placeholders, ", ")+")")
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		query.Model,
		strings.Join(columns, ", "),
		strings.Join(valueClauses, ", "),
	)

	// Add RETURNING clause for PostgreSQL
	if c.dialect == domain.PostgreSQL {
		sql += " RETURNING *"
	}

	return sql, args, nil
}
