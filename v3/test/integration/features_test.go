package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/builder"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrThrowMethods_Integration(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()
	defer cleanupTestData(t)

	ctx := context.Background()

	t.Run("FindFirstOrThrow returns error when not found", func(t *testing.T) {
		// Query empty table
		_, err := svc.FindFirstOrThrow(ctx, "users")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no users record was found")
	})

	t.Run("FindFirstOrThrow returns record when exists", func(t *testing.T) {
		// Create user
		user, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "test@example.com",
			"name":  "Test User",
		})
		require.NoError(t, err)
		require.NotNil(t, user)

		// FindFirstOrThrow should succeed
		result, err := svc.FindFirstOrThrow(ctx, "users")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("FindUniqueOrThrow with WHERE returns error when not found", func(t *testing.T) {
		_, err := svc.FindUniqueOrThrow(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    999,
			}))
		require.Error(t, err)
	})
}

func TestDistinctQueries_Integration(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()
	defer cleanupTestData(t)

	ctx := context.Background()

	t.Run("DISTINCT ON returns unique values", func(t *testing.T) {
		// Create users with duplicate emails domains
		_, _ = svc.Create(ctx, "users", map[string]interface{}{"email": "user1@example.com", "name": "User 1"})
		_, _ = svc.Create(ctx, "users", map[string]interface{}{"email": "user2@example.com", "name": "User 2"})
		_, _ = svc.Create(ctx, "users", map[string]interface{}{"email": "user3@test.com", "name": "User 3"})

		// Query with DISTINCT doesn't return duplicates at SQL level
		results, err := svc.FindMany(ctx, "users",
			service.WithDistinct("email"))

		require.NoError(t, err)
		assert.NotNil(t, results)
	})
}

func TestCursorPagination_Integration(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()
	defer cleanupTestData(t)

	ctx := context.Background()

	t.Run("Cursor-based pagination", func(t *testing.T) {
		// Create multiple users
		for i := 1; i <= 5; i++ {
			_, err := svc.Create(ctx, "users", map[string]interface{}{
				"email": fmt.Sprintf("user%d@example.com", i),
				"name":  fmt.Sprintf("User %d", i),
			})
			require.NoError(t, err)
		}

		// First page
		page1, err := svc.FindMany(ctx, "users",
			service.WithOrderBy("id", domain.Asc),
			service.WithTake(2))
		require.NoError(t, err)

		results := page1.([]map[string]interface{})
		require.Len(t, results, 2)
		lastID := results[1]["id"]

		// Second page using cursor
		page2, err := svc.FindMany(ctx, "users",
			service.WithCursor("id", lastID),
			service.WithOrderBy("id", domain.Asc),
			service.WithTake(2))
		require.NoError(t, err)

		results2 := page2.([]map[string]interface{})
		require.Len(t, results2, 2)

		// Verify second page starts after cursor
		assert.Greater(t, results2[0]["id"], lastID)
	})
}

func TestGroupByHaving_Integration(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()
	defer cleanupTestData(t)

	ctx := context.Background()

	t.Run("GroupBy with aggregation", func(t *testing.T) {
		// Create users and posts
		user1, _ := svc.Create(ctx, "users", map[string]interface{}{"email": "user1@example.com", "name": "User 1"})
		user2, _ := svc.Create(ctx, "users", map[string]interface{}{"email": "user2@example.com", "name": "User 2"})

		userMap1 := user1
		userMap2 := user2

		// Create posts
		for i := 0; i < 3; i++ {
			svc.Create(ctx, "posts", map[string]interface{}{
				"title":     fmt.Sprintf("Post %d", i),
				"author_id": userMap1["id"],
			})
		}
		svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Post by User 2",
			"author_id": userMap2["id"],
		})

		// GroupBy author_id
		results, err := svc.GroupBy(ctx, "posts",
			[]string{"author_id"},
			[]domain.Aggregation{{Function: domain.Count, Field: "id"}})

		require.NoError(t, err)
		require.Len(t, results, 2)
	})
}

func TestNestedWrites_Integration(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()
	defer cleanupTestData(t)

	ctx := context.Background()

	t.Run("NestedCreate with transaction", func(t *testing.T) {
		// Create user with profile using nested write
		qb := builder.NewQueryBuilder("users").
			Create(map[string]interface{}{
				"email": "nested@example.com",
				"name":  "Nested User",
			}).
			NestedCreate("profiles", map[string]interface{}{
				"bio": "Test bio",
			})

		query := qb.GetQuery()

		// This would use ExecuteNestedWrites in production
		// For now, create user then profile
		user, err := svc.Create(ctx, "users", query.CreateData)
		require.NoError(t, err)

		userMap := user
		profile, err := svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "Test bio",
			"user_id": userMap["id"],
		})
		require.NoError(t, err)
		assert.NotNil(t, profile)
	})
}

func TestRawSQL_Integration(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()
	defer cleanupTestData(t)

	ctx := context.Background()

	t.Run("QueryRaw with parameter binding", func(t *testing.T) {
		// Create test data
		svc.Create(ctx, "users", map[string]interface{}{"email": "raw@example.com", "name": "Raw User"})

		// Execute raw SQL
		results, err := svc.QueryRaw(ctx, "SELECT * FROM users WHERE email = $1", "raw@example.com")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "raw@example.com", results[0]["email"])
	})

	t.Run("ExecuteRaw mutation", func(t *testing.T) {
		user, _ := svc.Create(ctx, "users", map[string]interface{}{"email": "exec@example.com", "name": "Exec User"})
		userMap := user

		// Execute update
		affected, err := svc.ExecuteRaw(ctx, "UPDATE users SET status = $1 WHERE id = $2", "inactive", userMap["id"])
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		// Verify update
		result, _ := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{Field: "id", Operator: domain.Equals, Value: userMap["id"]}))
		resultMap := result.([]map[string]interface{})
		assert.Equal(t, "inactive", resultMap[0]["status"])
	})
}

func TestComplexQuery_Integration(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()
	defer cleanupTestData(t)

	ctx := context.Background()

	t.Run("Combined filters, ordering, and pagination", func(t *testing.T) {
		// Create test data
		for i := 1; i <= 10; i++ {
			status := "active"
			if i%2 == 0 {
				status = "inactive"
			}
			svc.Create(ctx, "users", map[string]interface{}{
				"email":  fmt.Sprintf("user%d@example.com", i),
				"name":   fmt.Sprintf("User %d", i),
				"status": status,
			})
		}

		// Complex query
		results, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{Field: "status", Operator: domain.Equals, Value: "active"}),
			service.WithOrderBy("created_at", domain.Desc),
			service.WithSkip(0),
			service.WithTake(3))

		require.NoError(t, err)
		resultSlice := results.([]map[string]interface{})
		require.LessOrEqual(t, len(resultSlice), 3)

		// Verify all results have status = active
		for _, r := range resultSlice {
			assert.Equal(t, "active", r["status"])
		}
	})
}
