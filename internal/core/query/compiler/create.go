// Package compiler compiles CREATE operations to SQL.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/pkg/domain"
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
		// Check if this is a relation field
		field, err := c.registry.GetField(query.Model, col)
		if err == nil && isRelationField(*field, c.registry) {
			// Process nested write
			nestedMap, ok := val.(map[string]interface{})
			if !ok {
				// Maybe error or skip?
				// For now, if we can't parse it, ignore or error.
				// Prisma client ensures it's a map (Fluent API does this).
				continue
			}

			// Parse operations (create, connect, etc.)
			for op, opData := range nestedMap {
				// Basic implementation for "create"
				if op == "create" {
					// Handle single create
					if data, ok := opData.(map[string]interface{}); ok {
						query.NestedWrites = append(query.NestedWrites, domain.NestedWrite{
							Relation:  col,
							Operation: domain.NestedCreate,
							Data:      data,
						})
					} else if list, ok := opData.([]map[string]interface{}); ok {
						// Handle create many (as list of single creates for now)
						/*
						   NOTE: This should probably be NestedCreateMany, but the domain
						   structure assumes 'Many []NestedWrite'.
						   For simplicity in this step, let's treat generic 'create'
						   that takes a list as multiple NestedCreate ops?
						   Actually domain has NestedCreateMany.
						*/
						// Actually, standard Prisma "create" on list relation takes a list or single.
						// We will map list to generic 'Create' ops or use 'Many' field in NestedWrite?
						// Let's use generic 'NestedCreate' but populating 'Many' if we were strictly following domain.
						// Or just append multiple NestedWrites? No, they belong to one relation.
						// Creating a specific NestedWrite for this relation.
						// A NestedWrite wraps the operation.
						// If opData is list, it means multiple inputs for this operation.
						// For now, let's stick to single create for MVP iteration or simple list loop.
						for _, item := range list {
							query.NestedWrites = append(query.NestedWrites, domain.NestedWrite{
								Relation:  col,
								Operation: domain.NestedCreate,
								Data:      item,
							})
						}
					}
				}
				// TODO: generic "createMany", "connect", etc.
			}
			continue
		}

		// Scalar field
		columns = append(columns, c.QuoteIdentifier(col))
		placeholders = append(placeholders, c.placeholder(&paramCount))
		args = append(args, val)
	}

	if len(columns) == 0 && len(query.NestedWrites) > 0 {
		// Only nested writes (e.g. creating a parent with only defaults + relations)
		// We need to insert the parent to get the ID.
		// INSERT INTO "Table" DEFAULT VALUES RETURNING *
		// But this depends on dialect.
		// For now assume at least one scalar or defaults.
		// If columns empty, use "DEFAULT VALUES" syntax?
		// Postgres: INSERT INTO ... DEFAULT VALUES ...
	}

	var sql string
	if len(columns) > 0 {
		sql = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			c.QuoteIdentifier(query.Model),
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)
	} else {
		// Handle empty columns (only defaults)
		if c.dialect == domain.PostgreSQL {
			sql = fmt.Sprintf("INSERT INTO %s DEFAULT VALUES", c.QuoteIdentifier(query.Model))
		} else {
			// Basic fallback
			return "", nil, fmt.Errorf("empty insert data not supported for this dialect")
		}
	}

	// Add RETURNING clause for PostgreSQL
	if c.dialect == domain.PostgreSQL {
		sql += " RETURNING *"
	}

	return sql, args, nil
}
