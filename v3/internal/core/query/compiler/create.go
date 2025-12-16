// Package compiler compiles CREATE operations to SQL.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// CompileCreate compiles a CREATE query to INSERT SQL.
func (c *SQLCompiler) CompileCreate(query *domain.Query) (string, []interface{}, error) {
	if len(query.CreateData) == 0 {
		return "", nil, fmt.Errorf("create data cannot be empty")
	}

	var columns []string
	var placeholders []string
	var args []interface{}

	paramCount := 1
	for col, val := range query.CreateData {
		columns = append(columns, col)
		placeholders = append(placeholders, c.placeholder(&paramCount))
		args = append(args, val)
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		query.Model,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Add RETURNING clause for PostgreSQL
	if c.dialect == domain.PostgreSQL {
		sql += " RETURNING *"
	}

	return sql, args, nil
}
