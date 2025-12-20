package compiler

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNestedLogicalOperators(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("Simple nested AND", func(t *testing.T) {
		// (status = 'active' AND role = 'admin')
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.AND,
				Conditions: []domain.Condition{
					{Field: "status", Operator: domain.Equals, Value: "active"},
					{Field: "role", Operator: domain.Equals, Value: "admin"},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "WHERE")
		assert.Contains(t, compiled.SQL.Query, "status = $1")
		assert.Contains(t, compiled.SQL.Query, "AND")
		assert.Contains(t, compiled.SQL.Query, "role = $2")
		assert.Len(t, compiled.SQL.Args, 2)
	})

	t.Run("Nested OR inside AND", func(t *testing.T) {
		// status = 'active' AND (role = 'admin' OR role = 'moderator')
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.AND,
				Conditions: []domain.Condition{
					{Field: "status", Operator: domain.Equals, Value: "active"},
				},
				NestedFilters: []domain.Filter{
					{
						Operator: domain.OR,
						Conditions: []domain.Condition{
							{Field: "role", Operator: domain.Equals, Value: "admin"},
							{Field: "role", Operator: domain.Equals, Value: "moderator"},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "status = $1")
		assert.Contains(t, sql, "AND")
		// Should have parentheses around the OR clause
		assert.Contains(t, sql, "(")
		assert.Contains(t, sql, "role")
		assert.Contains(t, sql, "OR")
		assert.Len(t, compiled.SQL.Args, 3)
	})

	t.Run("Complex nested combination", func(t *testing.T) {
		// (status = 'active' AND role = 'admin') OR (verified = true AND premium = true)
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.OR,
				NestedFilters: []domain.Filter{
					{
						Operator: domain.AND,
						Conditions: []domain.Condition{
							{Field: "status", Operator: domain.Equals, Value: "active"},
							{Field: "role", Operator: domain.Equals, Value: "admin"},
						},
					},
					{
						Operator: domain.AND,
						Conditions: []domain.Condition{
							{Field: "verified", Operator: domain.Equals, Value: true},
							{Field: "premium", Operator: domain.Equals, Value: true},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "WHERE")
		// Should have two groups of AND conditions joined by OR
		assert.Contains(t, sql, "OR")
		assert.Contains(t, sql, "status")
		assert.Contains(t, sql, "role")
		assert.Contains(t, sql, "verified")
		assert.Contains(t, sql, "premium")
		// Should have parentheses around each AND group
		assert.Regexp(t, `\(.*status.*AND.*role.*\)`, sql)
		assert.Len(t, compiled.SQL.Args, 4)
	})

	t.Run("Triple nesting", func(t *testing.T) {
		// (a = 1 AND (b = 2 OR c = 3)) OR d = 4
		query := &domain.Query{
			Model:     "test",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.OR,
				NestedFilters: []domain.Filter{
					{
						Operator: domain.AND,
						Conditions: []domain.Condition{
							{Field: "a", Operator: domain.Equals, Value: 1},
						},
						NestedFilters: []domain.Filter{
							{
								Operator: domain.OR,
								Conditions: []domain.Condition{
									{Field: "b", Operator: domain.Equals, Value: 2},
									{Field: "c", Operator: domain.Equals, Value: 3},
								},
							},
						},
					},
					{
						Conditions: []domain.Condition{
							{Field: "d", Operator: domain.Equals, Value: 4},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "a")
		assert.Contains(t, sql, "b")
		assert.Contains(t, sql, "c")
		assert.Contains(t, sql, "d")
		// Should have proper nesting with parentheses
		assert.Contains(t, sql, "(")
		assert.Contains(t, sql, ")")
		assert.Len(t, compiled.SQL.Args, 4)
	})

	t.Run("NOT with nested filters", func(t *testing.T) {
		// NOT (status = 'inactive' OR deleted = true)
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.NOT,
				NestedFilters: []domain.Filter{
					{
						Operator: domain.OR,
						Conditions: []domain.Condition{
							{Field: "status", Operator: domain.Equals, Value: "inactive"},
							{Field: "deleted", Operator: domain.Equals, Value: true},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "NOT")
		assert.Contains(t, sql, "status")
		assert.Contains(t, sql, "deleted")
		assert.Len(t, compiled.SQL.Args, 2)
	})

	t.Run("Mixed conditions and nested filters", func(t *testing.T) {
		// email LIKE '%@example.com' AND (status = 'active' OR verified = true)
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.AND,
				Conditions: []domain.Condition{
					{Field: "email", Operator: domain.Contains, Value: "@example.com"},
				},
				NestedFilters: []domain.Filter{
					{
						Operator: domain.OR,
						Conditions: []domain.Condition{
							{Field: "status", Operator: domain.Equals, Value: "active"},
							{Field: "verified", Operator: domain.Equals, Value: true},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "email")
		assert.Contains(t, sql, "LIKE")
		assert.Contains(t, sql, "AND")
		assert.Contains(t, sql, "status")
		assert.Contains(t, sql, "verified")
		assert.Contains(t, sql, "OR")
		assert.Len(t, compiled.SQL.Args, 3)
	})
}

func TestNestedFiltersEdgeCases(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("Empty nested filter", func(t *testing.T) {
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.AND,
				Conditions: []domain.Condition{
					{Field: "id", Operator: domain.Equals, Value: 1},
				},
				NestedFilters: []domain.Filter{
					{}, // Empty filter
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		// Should still work, just ignoring empty nested filter
		assert.Contains(t, compiled.SQL.Query, "id = $1")
	})

	t.Run("Only nested filters, no direct conditions", func(t *testing.T) {
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Operator: domain.OR,
				NestedFilters: []domain.Filter{
					{
						Conditions: []domain.Condition{
							{Field: "a", Operator: domain.Equals, Value: 1},
						},
					},
					{
						Conditions: []domain.Condition{
							{Field: "b", Operator: domain.Equals, Value: 2},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "a")
		assert.Contains(t, sql, "b")
		assert.Contains(t, sql, "OR")
	})
}
