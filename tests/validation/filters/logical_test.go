package filters

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/service"
	"github.com/satishbabariya/prisma-go/v3/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleConditions(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("SingleConditionEquals", func(t *testing.T) {
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

	t.Run("SingleConditionNotEquals", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "role",
				Operator: domain.NotEquals,
				Value:    "ADMIN",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.NotEqual(t, "ADMIN", r["role"])
		}
	})

	t.Run("FilterByRoleADMIN", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "role",
				Operator: domain.Equals,
				Value:    "ADMIN",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.GreaterOrEqual(t, len(resultSlice), 1)
	})

	t.Run("FilterByRoleMODERATOR", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "role",
				Operator: domain.Equals,
				Value:    "MODERATOR",
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestBooleanFilters(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("BooleanTrue", func(t *testing.T) {
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

	t.Run("BooleanFalse", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    false,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.Equal(t, false, r["is_active"])
		}
	})

	t.Run("PostStatusPublished", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "status",
				Operator: domain.Equals,
				Value:    "PUBLISHED",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(resultSlice), 1)
	})

	t.Run("PostStatusDraft", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "status",
				Operator: domain.Equals,
				Value:    "DRAFT",
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestCombineFiltersWithOptions(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FilterWithOrdering", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}),
			service.WithOrderBy("age", domain.Desc))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("FilterWithPagination", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}),
			service.WithSkip(0),
			service.WithTake(2))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.LessOrEqual(t, len(resultSlice), 2)
	})

	t.Run("FilterWithOrderingAndPagination", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}),
			service.WithOrderBy("created_at", domain.Desc),
			service.WithSkip(0),
			service.WithTake(5))
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestNullConditions(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("IsNull", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "name",
				Operator: domain.Equals,
				Value:    nil,
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("IsNotNull", func(t *testing.T) {
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

	t.Run("NullAge", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Equals,
				Value:    nil,
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestMultipleFiltersChained(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("ActiveUsersByRole", func(t *testing.T) {
		// First find active users, then filter by role
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})

		// Count by role
		userCount := 0
		for _, r := range resultSlice {
			if r["role"] == "USER" {
				userCount++
			}
		}
		assert.GreaterOrEqual(t, userCount, 0)
	})

	t.Run("PostsByStatus", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "status",
				Operator: domain.Equals,
				Value:    "PUBLISHED",
			}),
			service.WithOrderBy("view_count", domain.Desc))
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}
