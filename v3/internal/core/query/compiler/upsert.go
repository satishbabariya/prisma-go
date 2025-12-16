// Package compiler compiles UPSERT operations to SQL.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// CompileUpsert compiles an UPSERT query to INSERT ... ON CONFLICT DO UPDATE SQL.
func (c *SQLCompiler) CompileUpsert(query *domain.Query) (string, []interface{}, error) {
	if len(query.UpsertData) == 0 {
		return "", nil, fmt.Errorf("upsert data cannot be empty")
	}

	if len(query.UpsertKeys) == 0 {
		return "", nil, fmt.Errorf("upsert conflict keys cannot be empty")
	}

	var columns []string
	var placeholders []string
	var args []interface{}

	paramCount := 1
	for col, val := range query.UpsertData {
		columns = append(columns, col)
		placeholders = append(placeholders, c.placeholder(&paramCount))
		args = append(args, val)
	}

	// Build base INSERT
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		query.Model,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Add ON CONFLICT clause based on dialect
	switch c.dialect {
	case domain.PostgreSQL:
		// PostgreSQL: ON CONFLICT (key) DO UPDATE SET ...
		sql += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET ", strings.Join(query.UpsertKeys, ", "))

		var updateClauses []string
		for col, val := range query.UpsertUpdate {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = %s", col, c.placeholder(&paramCount)))
			args = append(args, val)
		}
		sql += strings.Join(updateClauses, ", ")
		sql += " RETURNING *"

	case domain.MySQL:
		// MySQL: ON DUPLICATE KEY UPDATE ...
		sql += " ON DUPLICATE KEY UPDATE "

		var updateClauses []string
		for col, val := range query.UpsertUpdate {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = %s", col, c.placeholder(&paramCount)))
			args = append(args, val)
		}
		sql += strings.Join(updateClauses, ", ")

	case domain.SQLite:
		// SQLite: ON CONFLICT (key) DO UPDATE SET ...
		sql += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET ", strings.Join(query.UpsertKeys, ", "))

		var updateClauses []string
		for col, val := range query.UpsertUpdate {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = %s", col, c.placeholder(&paramCount)))
			args = append(args, val)
		}
		sql += strings.Join(updateClauses, ", ")
		sql += " RETURNING *"

	default:
		return "", nil, fmt.Errorf("upsert not supported for dialect: %s", c.dialect)
	}

	return sql, args, nil
}
