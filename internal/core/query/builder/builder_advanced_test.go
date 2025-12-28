package builder

import (
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrThrowMethods(t *testing.T) {
	t.Run("FindFirstOrThrow sets ThrowIfNotFound flag", func(t *testing.T) {
		qb := NewQueryBuilder("users").FindFirstOrThrow()
		query := qb.GetQuery()

		assert.Equal(t, domain.FindFirst, query.Operation)
		assert.True(t, query.ThrowIfNotFound, "ThrowIfNotFound should be true")
	})

	t.Run("FindUniqueOrThrow sets ThrowIfNotFound flag", func(t *testing.T) {
		qb := NewQueryBuilder("users").FindUniqueOrThrow()
		query := qb.GetQuery()

		assert.Equal(t, domain.FindUnique, query.Operation)
		assert.True(t, query.ThrowIfNotFound, "ThrowIfNotFound should be true")
	})

	t.Run("Regular FindFirst does not set ThrowIfNotFound", func(t *testing.T) {
		qb := NewQueryBuilder("users").FindFirst()
		query := qb.GetQuery()

		assert.Equal(t, domain.FindFirst, query.Operation)
		assert.False(t, query.ThrowIfNotFound, "ThrowIfNotFound should be false")
	})
}

func TestDistinctQuery(t *testing.T) {
	t.Run("Distinct sets fields correctly", func(t *testing.T) {
		qb := NewQueryBuilder("users").
			FindMany().
			Distinct("email", "name")

		query := qb.GetQuery()

		require.Len(t, query.Distinct, 2)
		assert.Contains(t, query.Distinct, "email")
		assert.Contains(t, query.Distinct, "name")
	})

	t.Run("Distinct with single field", func(t *testing.T) {
		qb := NewQueryBuilder("posts").
			FindMany().
			Distinct("category")

		query := qb.GetQuery()

		assert.Equal(t, []string{"category"}, query.Distinct)
	})
}

func TestCursorPagination(t *testing.T) {
	t.Run("Cursor sets field and value", func(t *testing.T) {
		qb := NewQueryBuilder("posts").
			FindMany().
			Cursor("id", 100)

		query := qb.GetQuery()

		require.NotNil(t, query.Cursor)
		assert.Equal(t, "id", query.Cursor.Field)
		assert.Equal(t, 100, query.Cursor.Value)
	})

	t.Run("Cursor with string value", func(t *testing.T) {
		qb := NewQueryBuilder("users").
			FindMany().
			Cursor("created_at", "2024-01-01")

		query := qb.GetQuery()

		require.NotNil(t, query.Cursor)
		assert.Equal(t, "created_at", query.Cursor.Field)
		assert.Equal(t, "2024-01-01", query.Cursor.Value)
	})
}

func TestGroupByAndHaving(t *testing.T) {
	t.Run("GroupBy sets fields", func(t *testing.T) {
		qb := NewQueryBuilder("orders").
			GroupByFields("customer_id", "status")

		query := qb.GetQuery()

		assert.Equal(t, []string{"customer_id", "status"}, query.GroupBy)
	})

	t.Run("Having sets filter conditions", func(t *testing.T) {
		qb := NewQueryBuilder("orders").
			GroupByFields("customer_id").
			Having(domain.Condition{
				Field:    "COUNT(*)",
				Operator: domain.Gt,
				Value:    5,
			})

		query := qb.GetQuery()

		require.Len(t, query.Having.Conditions, 1)
		assert.Equal(t, "COUNT(*)", query.Having.Conditions[0].Field)
		assert.Equal(t, domain.Gt, query.Having.Conditions[0].Operator)
	})
}

func TestFilterOperators(t *testing.T) {
	t.Run("Case insensitive filter", func(t *testing.T) {
		qb := NewQueryBuilder("users").
			Where(domain.Condition{
				Field:    "email",
				Operator: domain.Contains,
				Value:    "example",
				Mode:     domain.ModeInsensitive,
			})

		query := qb.GetQuery()

		require.Len(t, query.Filter.Conditions, 1)
		assert.Equal(t, domain.ModeInsensitive, query.Filter.Conditions[0].Mode)
	})

	t.Run("Array has operator", func(t *testing.T) {
		qb := NewQueryBuilder("posts").
			Where(domain.Condition{
				Field:    "tags",
				Operator: domain.Has,
				Value:    "golang",
			})

		query := qb.GetQuery()

		require.Len(t, query.Filter.Conditions, 1)
		assert.Equal(t, domain.Has, query.Filter.Conditions[0].Operator)
	})

	t.Run("IsNull operator", func(t *testing.T) {
		qb := NewQueryBuilder("users").
			Where(domain.Condition{
				Field:    "deleted_at",
				Operator: domain.IsNull,
				Value:    true,
			})

		query := qb.GetQuery()

		require.Len(t, query.Filter.Conditions, 1)
		assert.Equal(t, domain.IsNull, query.Filter.Conditions[0].Operator)
	})

	t.Run("Full-text search", func(t *testing.T) {
		qb := NewQueryBuilder("articles").
			Where(domain.Condition{
				Field:    "content",
				Operator: domain.Search,
				Value:    "prisma golang",
			})

		query := qb.GetQuery()

		require.Len(t, query.Filter.Conditions, 1)
		assert.Equal(t, domain.Search, query.Filter.Conditions[0].Operator)
	})
}

func TestComplexQueries(t *testing.T) {
	t.Run("Combined distinct, cursor, and filters", func(t *testing.T) {
		qb := NewQueryBuilder("posts").
			FindMany().
			Distinct("category").
			Cursor("id", 50).
			Where(domain.Condition{
				Field:    "published",
				Operator: domain.Equals,
				Value:    true,
			}).
			OrderBy("created_at", domain.Desc).
			Take(20)

		query := qb.GetQuery()

		assert.Equal(t, []string{"category"}, query.Distinct)
		assert.Equal(t, 50, query.Cursor.Value)
		assert.Len(t, query.Filter.Conditions, 1)
		assert.NotNil(t, query.Pagination.Take)
		assert.Equal(t, 20, *query.Pagination.Take)
	})
}

func TestNestedWriteBuilders(t *testing.T) {
	t.Run("NestedCreate", func(t *testing.T) {
		qb := NewQueryBuilder("users").
			Create(map[string]interface{}{"name": "John"}).
			NestedCreate("posts", map[string]interface{}{
				"title": "First Post",
			})

		query := qb.GetQuery()

		require.Len(t, query.NestedWrites, 1)
		assert.Equal(t, domain.NestedCreate, query.NestedWrites[0].Operation)
		assert.Equal(t, "posts", query.NestedWrites[0].Relation)
	})

	t.Run("NestedConnect", func(t *testing.T) {
		qb := NewQueryBuilder("users").
			Update(map[string]interface{}{"name": "Updated"}).
			NestedConnect("profile",
				domain.Condition{Field: "id", Operator: domain.Equals, Value: 1},
			)

		query := qb.GetQuery()

		require.Len(t, query.NestedWrites, 1)
		assert.Equal(t, domain.NestedConnect, query.NestedWrites[0].Operation)
	})

	t.Run("NestedDisconnect", func(t *testing.T) {
		qb := NewQueryBuilder("users").
			Update(map[string]interface{}{"status": "inactive"}).
			NestedDisconnect("posts",
				domain.Condition{Field: "draft", Operator: domain.Equals, Value: true},
			)

		query := qb.GetQuery()

		require.Len(t, query.NestedWrites, 1)
		assert.Equal(t, domain.NestedDisconnect, query.NestedWrites[0].Operation)
	})
}
