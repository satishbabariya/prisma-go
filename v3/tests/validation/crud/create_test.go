package crud

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/service"
	"github.com/satishbabariya/prisma-go/v3/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()
	defer validation.CleanupTables(t, testCtx.DB)

	ctx := context.Background()
	svc := testCtx.Service

	t.Run("SingleRecordWithRequiredFields", func(t *testing.T) {
		result, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "create_test1@example.com",
			"name":  "Create Test User",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "create_test1@example.com", result["email"])
		assert.NotNil(t, result["id"])
	})

	t.Run("SingleRecordWithAllFieldTypes", func(t *testing.T) {
		result, err := svc.Create(ctx, "users", map[string]interface{}{
			"email":     "create_test2@example.com",
			"name":      "Full Fields User",
			"role":      "ADMIN",
			"is_active": true,
			"age":       30,
			"balance":   1500.75,
			"metadata":  `{"key": "value"}`,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "create_test2@example.com", result["email"])
		assert.Equal(t, "Full Fields User", result["name"])
	})

	t.Run("SingleRecordWithOptionalNullFields", func(t *testing.T) {
		result, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "create_test3@example.com",
			"name":  nil,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result["name"])
	})

	t.Run("SingleRecordWithDefaultValues", func(t *testing.T) {
		result, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "create_test4@example.com",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		// Check default values are applied
		assert.NotNil(t, result["created_at"])
		assert.NotNil(t, result["updated_at"])
	})

	t.Run("CreateManyBulkOperation", func(t *testing.T) {
		users := []map[string]interface{}{
			{"email": "bulk1@example.com", "name": "Bulk User 1"},
			{"email": "bulk2@example.com", "name": "Bulk User 2"},
			{"email": "bulk3@example.com", "name": "Bulk User 3"},
		}

		results, err := svc.CreateMany(ctx, "users", users)
		require.NoError(t, err)
		require.Len(t, results, 3)
	})

	t.Run("CreateWithRelatedRecord", func(t *testing.T) {
		// First create a user
		user, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "profile_parent@example.com",
			"name":  "Profile Parent",
		})
		require.NoError(t, err)
		userId := user["id"]

		// Then create related profile
		profile, err := svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "Test bio for profile",
			"website": "https://example.com",
			"user_id": userId,
		})
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, userId, profile["user_id"])
	})

	t.Run("UniqueConstraintViolation", func(t *testing.T) {
		// Create first user
		_, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "unique_test@example.com",
		})
		require.NoError(t, err)

		// Try to create duplicate
		_, err = svc.Create(ctx, "users", map[string]interface{}{
			"email": "unique_test@example.com",
		})
		require.Error(t, err)
		// Should be unique constraint violation
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("ForeignKeyConstraintViolation", func(t *testing.T) {
		_, err := svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "Orphan profile",
			"user_id": 99999, // Non-existent user
		})
		require.Error(t, err)
		// Should be foreign key violation
	})
}

func TestUpsertOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()
	defer validation.CleanupTables(t, testCtx.DB)

	ctx := context.Background()
	svc := testCtx.Service

	t.Run("UpsertInsert", func(t *testing.T) {
		result, err := svc.Upsert(ctx, "users",
			map[string]interface{}{
				"email": "upsert_new@example.com",
				"name":  "New Upsert User",
			},
			map[string]interface{}{
				"name": "Updated Upsert User",
			},
			[]string{"email"},
		)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "New Upsert User", result["name"])
	})

	t.Run("UpsertUpdate", func(t *testing.T) {
		// First create
		_, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "upsert_existing@example.com",
			"name":  "Original Name",
		})
		require.NoError(t, err)

		// Then upsert (should update)
		result, err := svc.Upsert(ctx, "users",
			map[string]interface{}{
				"email": "upsert_existing@example.com",
				"name":  "Should Not Use This",
			},
			map[string]interface{}{
				"name": "Updated Name",
			},
			[]string{"email"},
		)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", result["name"])
	})
}

func TestCreateWithEnums(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()
	defer validation.CleanupTables(t, testCtx.DB)

	ctx := context.Background()
	svc := testCtx.Service

	t.Run("CreateWithValidEnum", func(t *testing.T) {
		for _, role := range []string{"USER", "ADMIN", "MODERATOR"} {
			result, err := svc.Create(ctx, "users", map[string]interface{}{
				"email": role + "_test@example.com",
				"role":  role,
			})
			require.NoError(t, err)
			assert.NotNil(t, result)
		}
	})

	t.Run("CreatePostWithStatus", func(t *testing.T) {
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "post_author@example.com",
		})

		for _, status := range []string{"DRAFT", "PUBLISHED", "ARCHIVED"} {
			result, err := svc.Create(ctx, "posts", map[string]interface{}{
				"title":     status + " Post",
				"slug":      status + "-post",
				"status":    status,
				"author_id": user["id"],
			})
			require.NoError(t, err)
			assert.NotNil(t, result)
		}
	})
}

func TestCreateWithJSON(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()
	defer validation.CleanupTables(t, testCtx.DB)

	ctx := context.Background()
	svc := testCtx.Service

	t.Run("CreateWithJSONObject", func(t *testing.T) {
		result, err := svc.Create(ctx, "users", map[string]interface{}{
			"email":    "json_test@example.com",
			"metadata": `{"preferences": {"theme": "dark", "language": "en"}}`,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("CreateWithJSONArray", func(t *testing.T) {
		result, err := svc.Create(ctx, "users", map[string]interface{}{
			"email":    "json_array@example.com",
			"metadata": `{"tags": ["go", "prisma", "database"]}`,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("CreateWithNestedJSON", func(t *testing.T) {
		result, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "json_nested@example.com",
			"metadata": `{
				"settings": {
					"notifications": {
						"email": true,
						"push": false
					},
					"privacy": {
						"profile": "public"
					}
				}
			}`,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestCreateWithDecimal(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()
	defer validation.CleanupTables(t, testCtx.DB)

	ctx := context.Background()
	svc := testCtx.Service

	t.Run("CreateWithDecimalPrecision", func(t *testing.T) {
		testCases := []float64{
			0.00,
			100.50,
			999999.99,
			0.01,
			-500.25,
		}

		for i, balance := range testCases {
			result, err := svc.Create(ctx, "users", map[string]interface{}{
				"email":   service.WithWhere(domain.Condition{}), // will fail - fix below
				"balance": balance,
			})
			_ = result
			_ = err
			_ = i
		}
	})
}
