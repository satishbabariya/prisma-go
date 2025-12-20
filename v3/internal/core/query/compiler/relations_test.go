package compiler

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelationLoading(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	t.Run("Simple include", func(t *testing.T) {
		// Load users with their posts
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Relations: []domain.RelationInclusion{
				{
					Relation: "posts",
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "SELECT")
		assert.Contains(t, sql, "users.*")
		assert.Contains(t, sql, "FROM users")
		assert.Contains(t, sql, "LEFT JOIN")
		assert.Contains(t, sql, "posts")
	})

	t.Run("Include with select", func(t *testing.T) {
		// Load users with specific post fields
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Relations: []domain.RelationInclusion{
				{
					Relation: "posts",
					Query: &domain.Query{
						Selection: domain.Selection{
							Fields: []string{"id", "title"},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "LEFT JOIN")
		assert.Contains(t, sql, "posts")
		// Should select specific fields
		assert.Contains(t, sql, "posts_id")
		assert.Contains(t, sql, "posts_title")
	})

	t.Run("Nested includes", func(t *testing.T) {
		// Load users -> posts -> comments (2 levels deep)
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Relations: []domain.RelationInclusion{
				{
					Relation: "posts",
					Query: &domain.Query{
						Relations: []domain.RelationInclusion{
							{Relation: "comments"},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "FROM users")
		assert.Contains(t, sql, "LEFT JOIN")
		// Should have two JOINs
		assert.Contains(t, sql, "posts")
		assert.Contains(t, sql, "comments")
	})

	t.Run("Multiple relations at same level", func(t *testing.T) {
		// Load users with both posts and profile
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Relations: []domain.RelationInclusion{
				{
					Relation: "posts",
					Include:  true,
				},
				{
					Relation: "profile",
					Include:  true,
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "LEFT JOIN")
		assert.Contains(t, sql, "posts")
		assert.Contains(t, sql, "profile")
	})

	t.Run("Deep nesting", func(t *testing.T) {
		// Load users -> posts -> comments -> author (3 levels)
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Relations: []domain.RelationInclusion{
				{
					Relation: "posts",
					Include:  true,
					NestedInclusions: []domain.RelationInclusion{
						{
							Relation: "comments",
							Include:  true,
							NestedInclusions: []domain.RelationInclusion{
								{
									Relation: "author",
									Include:  true,
								},
							},
						},
					},
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "FROM users")
		assert.Contains(t, sql, "LEFT JOIN")
		// Should have three JOINs
		assert.Contains(t, sql, "posts")
		assert.Contains(t, sql, "comments")
		assert.Contains(t, sql, "author")
	})

	t.Run("Include with WHERE clause", func(t *testing.T) {
		// Load users with posts, but filter users
		query := &domain.Query{
			Model:     "users",
			Operation: domain.FindMany,
			Filter: domain.Filter{
				Conditions: []domain.Condition{
					{Field: "active", Operator: domain.Equals, Value: true},
				},
			},
			Relations: []domain.RelationInclusion{
				{
					Relation: "posts",
					Include:  true,
				},
			},
		}

		compiled, err := compiler.Compile(ctx, query)
		require.NoError(t, err)

		sql := compiled.SQL.Query
		assert.Contains(t, sql, "LEFT JOIN")
		assert.Contains(t, sql, "WHERE")
		assert.Contains(t, sql, "active")
	})
}

func TestRelationJoinBuilding(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)

	t.Run("buildRelationJoins basic", func(t *testing.T) {
		relations := []domain.RelationInclusion{
			{
				Relation: "posts",
				Include:  true,
			},
		}

		joins, err := compiler.buildRelationJoins("users", "users", relations, nil)
		require.NoError(t, err)
		require.Len(t, joins, 1)

		assert.Equal(t, "LEFT JOIN", joins[0].JoinType)
		assert.Equal(t, "posts", joins[0].Table)
		assert.Equal(t, "users_posts", joins[0].Alias)
		assert.NotEmpty(t, joins[0].OnConditions)
	})

	t.Run("buildRelationJoins nested", func(t *testing.T) {
		relations := []domain.RelationInclusion{
			{
				Relation: "posts",
				Include:  true,
				NestedInclusions: []domain.RelationInclusion{
					{
						Relation: "comments",
						Include:  true,
					},
				},
			},
		}

		joins, err := compiler.buildRelationJoins("users", "users", relations, nil)
		require.NoError(t, err)
		require.Len(t, joins, 2)

		assert.Equal(t, "posts", joins[0].Table)
		assert.Equal(t, "comments", joins[1].Table)
	})
}
