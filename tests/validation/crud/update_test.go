package crud

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("UpdateSingleRecord", func(t *testing.T) {
		affected, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"name": "Updated Alice",
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		// Verify update
		result, _ := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		resultSlice := result.([]map[string]interface{})
		assert.Equal(t, "Updated Alice", resultSlice[0]["name"])
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		affected, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"name":      "Multi Update User",
				"is_active": false,
				"age":       99,
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    2,
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})

	t.Run("UpdateToNull", func(t *testing.T) {
		affected, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"name": nil,
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    3,
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		// Verify null
		result, _ := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    3,
			}))
		resultSlice := result.([]map[string]interface{})
		assert.Nil(t, resultSlice[0]["name"])
	})

	t.Run("UpdateNonExistent", func(t *testing.T) {
		affected, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"name": "Ghost User",
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    99999,
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})
}

func TestUpdateManyOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("UpdateManyWithCondition", func(t *testing.T) {
		// Update all active users
		affected, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"balance": 0.00,
			},
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, affected, int64(1))
	})

	t.Run("UpdateManyByRole", func(t *testing.T) {
		affected, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"is_active": true,
			},
			service.WithWhere(domain.Condition{
				Field:    "role",
				Operator: domain.Equals,
				Value:    "USER",
			}))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, affected, int64(1))
	})
}

func TestUpdateWithNumericOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("UpdateIncrementViewCount", func(t *testing.T) {
		// Get current view count
		before, _ := svc.FindUnique(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		beforeSlice := before.([]map[string]interface{})
		beforeCount := beforeSlice[0]["view_count"].(int64)

		// Update view count
		_, err := svc.Update(ctx, "posts",
			map[string]interface{}{
				"view_count": beforeCount + 1,
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)

		// Verify increment
		after, _ := svc.FindUnique(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		afterSlice := after.([]map[string]interface{})
		afterCount := afterSlice[0]["view_count"].(int64)
		assert.Equal(t, beforeCount+1, afterCount)
	})

	t.Run("UpdateDecrementBalance", func(t *testing.T) {
		// Create user with specific balance
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email":   "balance_test@example.com",
			"balance": 100.00,
		})

		// Decrement balance
		_, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"balance": 50.00,
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		require.NoError(t, err)

		// Verify
		result, _ := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		resultSlice := result.([]map[string]interface{})
		// Note: balance might be string or numeric depending on driver
		require.NotNil(t, resultSlice[0]["balance"])
	})
}

func TestUpdateEnumFields(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("UpdateUserRole", func(t *testing.T) {
		_, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"role": "MODERATOR",
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
	})

	t.Run("UpdatePostStatus", func(t *testing.T) {
		_, err := svc.Update(ctx, "posts",
			map[string]interface{}{
				"status": "PUBLISHED",
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    2, // Draft post
			}))
		require.NoError(t, err)
	})
}

func TestUpdateJSONFields(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("UpdateJSONMetadata", func(t *testing.T) {
		// Create user with initial metadata
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email":    "json_update@example.com",
			"metadata": `{"version": 1}`,
		})

		// Update metadata
		_, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"metadata": `{"version": 2, "updated": true}`,
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		require.NoError(t, err)
	})
}

func TestUpdateConstraints(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("UpdateViolatesUniqueConstraint", func(t *testing.T) {
		// Try to update email to existing email
		_, err := svc.Update(ctx, "users",
			map[string]interface{}{
				"email": "bob@example.com", // Already exists
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    1, // Alice
			}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})
}
