// Package compiler implements SQL compilation for DELETE operations.
package compiler

import (
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// compileDelete compiles a DELETE query to domain.SQL format.
func (c *SQLCompiler) compileDelete(query *domain.Query) (domain.SQL, error) {
	sql, args, err := c.CompileDelete(query)
	if err != nil {
		return domain.SQL{}, err
	}

	return domain.SQL{
		Query:   sql,
		Args:    args,
		Dialect: c.dialect,
	}, nil
}
