package crud

import (
	"context"
	"fmt"
	"testing"

	"github.com/satishbabariya/prisma-go/pkg/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindUniqueOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindUniqueByID", func(t *testing.T) {
		result, err := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FindUniqueByEmail", func(t *testing.T) {
		result, err := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Equals,
				Value:    "alice@example.com",
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FindUniqueNotFound", func(t *testing.T) {
		result, err := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    99999,
			}))
		require.NoError(t, err)
		// Should return empty result, not error
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 0)
	})

	t.Run("FindUniqueOrThrowFound", func(t *testing.T) {
		result, err := svc.FindUniqueOrThrow(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FindUniqueOrThrowNotFound", func(t *testing.T) {
		_, err := svc.FindUniqueOrThrow(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    99999,
			}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no users record was found")
	})
}

func TestFindFirstOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindFirstWithoutCondition", func(t *testing.T) {
		result, err := svc.FindFirst(ctx, "users")
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FindFirstWithCondition", func(t *testing.T) {
		result, err := svc.FindFirst(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "role",
				Operator: domain.Equals,
				Value:    "ADMIN",
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FindFirstWithOrderBy", func(t *testing.T) {
		result, err := svc.FindFirst(ctx, "users",
			service.WithOrderBy("created_at", domain.Desc))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FindFirstOrThrowEmpty", func(t *testing.T) {
		// Clear and test on empty table case
		validation.CleanupTables(t, testCtx.DB)

		_, err := svc.FindFirstOrThrow(ctx, "users")
		require.Error(t, err)

		// Restore data
		validation.SeedTestData(t, testCtx.DB)
	})
}

func TestFindManyOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindManyAll", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users")
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(resultSlice), 5)
	})

	t.Run("FindManyWithFilter", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.Equal(t, true, r["is_active"])
		}
	})

	t.Run("FindManyWithPagination", func(t *testing.T) {
		// First page
		page1, err := svc.FindMany(ctx, "users",
			service.WithOrderBy("id", domain.Asc),
			service.WithSkip(0),
			service.WithTake(2))
		require.NoError(t, err)
		page1Slice := page1.([]map[string]interface{})
		require.Len(t, page1Slice, 2)

		// Second page
		page2, err := svc.FindMany(ctx, "users",
			service.WithOrderBy("id", domain.Asc),
			service.WithSkip(2),
			service.WithTake(2))
		require.NoError(t, err)
		page2Slice := page2.([]map[string]interface{})
		require.Len(t, page2Slice, 2)

		// Verify different records
		assert.NotEqual(t, page1Slice[0]["id"], page2Slice[0]["id"])
	})

	t.Run("FindManyWithCursor", func(t *testing.T) {
		// Get first 2 records
		first, err := svc.FindMany(ctx, "users",
			service.WithOrderBy("id", domain.Asc),
			service.WithTake(2))
		require.NoError(t, err)
		firstSlice := first.([]map[string]interface{})
		lastId := firstSlice[1]["id"]

		// Get next records using cursor
		next, err := svc.FindMany(ctx, "users",
			service.WithCursor("id", lastId),
			service.WithOrderBy("id", domain.Asc),
			service.WithTake(2))
		require.NoError(t, err)
		nextSlice := next.([]map[string]interface{})

		// Verify cursor works (next records should have higher ID)
		for _, r := range nextSlice {
			assert.Greater(t, r["id"], lastId)
		}
	})

	t.Run("FindManyWithOrdering", func(t *testing.T) {
		// Order by name ascending
		result, err := svc.FindMany(ctx, "users",
			service.WithOrderBy("email", domain.Asc))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})

		// Verify ordering
		for i := 1; i < len(resultSlice); i++ {
			prev := resultSlice[i-1]["email"].(string)
			curr := resultSlice[i]["email"].(string)
			assert.LessOrEqual(t, prev, curr)
		}
	})

	t.Run("FindManyWithOrderingDesc", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithOrderBy("id", domain.Desc))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})

		// Verify descending order
		for i := 1; i < len(resultSlice); i++ {
			prev := resultSlice[i-1]["id"]
			curr := resultSlice[i]["id"]
			assert.Greater(t, prev, curr)
		}
	})
}

func TestFindManyWithNullHandling(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindRecordsWithNullField", func(t *testing.T) {
		// Eve has name = NULL in seed data
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "name",
				Operator: domain.Equals,
				Value:    nil,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(resultSlice), 1)
	})

	t.Run("FindRecordsWithNotNull", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "name",
				Operator: domain.NotEquals,
				Value:    nil,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.NotNil(t, r["name"])
		}
	})
}

func TestDistinctQueries(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("DistinctOnSingleField", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithDistinct("role"))
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestCount(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("CountAll", func(t *testing.T) {
		count, err := svc.Count(ctx, "users")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})

	t.Run("CountWithFilter", func(t *testing.T) {
		count, err := svc.Count(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "role",
				Operator: domain.Equals,
				Value:    "USER",
			}))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})
}

func TestRelatedRecordQueries(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindPostsByAuthor", func(t *testing.T) {
		// Alice (id=1) has multiple posts
		result, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(resultSlice), 1)

		for _, r := range resultSlice {
			assert.Equal(t, int64(1), r["author_id"])
		}
	})

	t.Run("FindCommentsByPost", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "comments",
			service.WithWhere(domain.Condition{
				Field:    "post_id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(resultSlice), 1)
	})

	t.Run("FindProfileForUser", func(t *testing.T) {
		result, err := svc.FindUnique(ctx, "profiles",
			service.WithWhere(domain.Condition{
				Field:    "user_id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestEmptyResults(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("FindManyEmptyTable", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users")
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 0)
	})

	t.Run("CountEmptyTable", func(t *testing.T) {
		count, err := svc.Count(ctx, "users")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestLargeDatasetQueries(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	// Create 100 records for pagination testing
	t.Run("CreateLargeDataset", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			_, err := svc.Create(ctx, "users", map[string]interface{}{
				"email": fmt.Sprintf("user%d@example.com", i),
				"name":  fmt.Sprintf("User %d", i),
				"age":   20 + (i % 50),
			})
			require.NoError(t, err)
		}
	})

	t.Run("PaginateLargeDataset", func(t *testing.T) {
		// Get count
		count, err := svc.Count(ctx, "users")
		require.NoError(t, err)
		assert.Equal(t, int64(100), count)

		// Paginate through all
		pageSize := 10
		totalFetched := 0
		for skip := 0; skip < 100; skip += pageSize {
			result, err := svc.FindMany(ctx, "users",
				service.WithOrderBy("id", domain.Asc),
				service.WithSkip(skip),
				service.WithTake(pageSize))
			require.NoError(t, err)
			resultSlice := result.([]map[string]interface{})
			totalFetched += len(resultSlice)
		}
		assert.Equal(t, 100, totalFetched)
	})
}
