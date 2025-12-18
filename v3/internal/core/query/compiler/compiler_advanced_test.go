package compiler

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileDistinct(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("DISTINCT ON for PostgreSQL", func(t *testing.T) {
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Distinct:  []string{"email"},
			Selection: domain.Selection{
				Fields: []string{"email", "name"},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "DISTINCT ON (email)")
		assert.Contains(t, compiled.SQL.Query, "email, name")
	})

	t.Run("Multiple DISTINCT fields", func(t *testing.T) {
		query := &domain.Query{
			Model:     "posts",
			Operation: domain.FindMany,
			Distinct:  []string{"category", "author_id"},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "DISTINCT ON (category, author_id)")
	})
}

func TestCompileCursorPagination(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("Cursor with WHERE clause", func(t *testing.T) {
		query := &domain.Query{
			Model:     "posts",
			Operation: domain.FindMany,
			Cursor: &domain.Cursor{
				Field: "id",
				Value: 100,
			},
			Pagination: domain.Pagination{
				Take: 20,
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "WHERE")
		assert.Contains(t, compiled.SQL.Query, "id >")
		assert.Contains(t, compiled.SQL.Query, "LIMIT")
		assert.Equal(t, 100, compiled.SQL.Args[0])
	})

	t.Run("Cursor with existing WHERE conditions", func(t *testing.T) {
		query := &domain.Query{
			Model:     "posts",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Conditions: []domain.Condition{
					{Field: "published", Operator: domain.Equals, Value: true},
				},
			},
			Cursor: &domain.Cursor{
				Field: "created_at",
				Value: "2024-01-01",
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "WHERE")
		assert.Contains(t, compiled.SQL.Query, "published")
		assert.Contains(t, compiled.SQL.Query, "created_at >")
	})
}

func TestCompileGroupByHaving(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("GROUP BY with aggregation", func(t *testing.T) {
		query := &domain.Query{
			Model:     "orders",
			Operation: domain.Aggregate,
			GroupBy:   []string{"customer_id"},
			Aggregations: []domain.Aggregation{
				{Function: domain.Count, Field: "*"},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "GROUP BY customer_id")
		assert.Contains(t, compiled.SQL.Query, "COUNT(*)")
	})

	t.Run("GROUP BY with HAVING clause", func(t *testing.T) {
		query := &domain.Query{
			Model:     "orders",
			Operation: domain.Aggregate,
			GroupBy:   []string{"customer_id"},
			Aggregations: []domain.Aggregation{
				{Function: domain.Sum, Field: "amount"},
			},
			Having: domain.Filter{
				Conditions: []domain.Condition{
					{Field: "SUM(amount)", Operator: domain.Gt, Value: 1000},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "GROUP BY")
		assert.Contains(t, compiled.SQL.Query, "HAVING")
		assert.Contains(t, compiled.SQL.Query, "SUM(amount)")
	})
}

func TestCompileFilterOperators(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("Case insensitive Contains", func(t *testing.T) {
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Conditions: []domain.Condition{
					{
						Field:    "email",
						Operator: domain.Contains,
						Value:    "example",
						Mode:     domain.ModeInsensitive,
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "LOWER(email) LIKE LOWER")
		assert.Contains(t, compiled.SQL.Args[0], "%example%")
	})

	t.Run("Array Has operator", func(t *testing.T) {
		query := &domain.Query{
			Model:     "posts",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Conditions: []domain.Condition{
					{
						Field:    "tags",
						Operator: domain.Has,
						Value:    "golang",
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "tags @> ARRAY")
	})

	t.Run("IsNull operator", func(t *testing.T) {
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Conditions: []domain.Condition{
					{
						Field:    "deleted_at",
						Operator: domain.IsNull,
						Value:    true,
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "deleted_at IS NULL")
	})

	t.Run("Search operator", func(t *testing.T) {
		query := &domain.Query{
			Model:     "articles",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Conditions: []domain.Condition{
					{
						Field:    "content",
						Operator: domain.Search,
						Value:    "prisma",
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		// PostgreSQL fulltext search
		assert.Contains(t, compiled.SQL.Query, "to_tsvector")
	})
}

func TestCompileComplexQueries(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("Combined DISTINCT, cursor, and filters", func(t *testing.T) {
		query := &domain.Query{
			Model:     "posts",
			Operation: domain.FindMany,
			Distinct:  []string{"category"},
			Cursor: &domain.Cursor{
				Field: "id",
				Value: 50,
			},
			Filter: domain.Filter{
				Conditions: []domain.Condition{
					{Field: "published", Operator: domain.Equals, Value: true},
				},
			},
			Ordering: []domain.OrderBy{
				{Field: "created_at", Direction: domain.Desc},
			},
			Pagination: domain.Pagination{
				Take: 20,
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "DISTINCT ON")
		assert.Contains(t, compiled.SQL.Query, "WHERE")
		assert.Contains(t, compiled.SQL.Query, "id >")
		assert.Contains(t, compiled.SQL.Query, "published")
		assert.Contains(t, compiled.SQL.Query, "ORDER BY")
		assert.Contains(t, compiled.SQL.Query, "LIMIT")
	})
}

func TestCompileCreateMany(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("CreateMany with RETURNING", func(t *testing.T) {
		query := &domain.Query{
			Model:     "users",
			Operation: domain.CreateMany,
			CreateManyData: []map[string]interface{}{
				{"name": "User1", "email": "user1@example.com"},
				{"name": "User2", "email": "user2@example.com"},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		assert.Contains(t, compiled.SQL.Query, "INSERT INTO users")
		assert.Contains(t, compiled.SQL.Query, "VALUES")
		assert.Contains(t, compiled.SQL.Query, "RETURNING *")
		// Should have 2 rows
		assert.Equal(t, 4, len(compiled.SQL.Args)) // 2 names + 2 emails
	})
}
