package crud

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/pkg/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("DeleteSingleRecord", func(t *testing.T) {
		// Create a user to delete
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "to_delete@example.com",
			"name":  "Delete Me",
		})

		affected, err := svc.Delete(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		// Verify deletion
		result, _ := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 0)
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		affected, err := svc.Delete(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    99999,
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})

	t.Run("DeleteByUniqueField", func(t *testing.T) {
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "delete_by_email@example.com",
		})
		_ = user

		affected, err := svc.Delete(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Equals,
				Value:    "delete_by_email@example.com",
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func TestDeleteManyOperations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	// Create test data
	for i := 0; i < 10; i++ {
		isActive := i%2 == 0
		svc.Create(ctx, "users", map[string]interface{}{
			"email":     service.WithWhere(domain.Condition{}), // will override
			"is_active": isActive,
		})
	}

	t.Run("DeleteManyWithCondition", func(t *testing.T) {
		// First, recreate test data properly
		validation.CleanupTables(t, testCtx.DB)

		// Create 5 active and 5 inactive users
		for i := 0; i < 10; i++ {
			isActive := i%2 == 0
			_, err := svc.Create(ctx, "users", map[string]interface{}{
				"email":     "deletemany" + string(rune('a'+i)) + "@example.com",
				"is_active": isActive,
			})
			require.NoError(t, err)
		}

		// Delete inactive users
		affected, err := svc.DeleteMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    false,
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(5), affected)

		// Verify only active remain
		count, _ := svc.Count(ctx, "users")
		assert.Equal(t, int64(5), count)
	})
}

func TestDeleteCascade(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("DeleteUserCascadesProfile", func(t *testing.T) {
		// Create user with profile
		user, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "cascade_user@example.com",
		})
		require.NoError(t, err)

		_, err = svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "Cascade test bio",
			"user_id": user["id"],
		})
		require.NoError(t, err)

		// Verify profile exists
		profileCount, _ := svc.Count(ctx, "profiles",
			service.WithWhere(domain.Condition{
				Field:    "user_id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		assert.Equal(t, int64(1), profileCount)

		// Delete user
		_, err = svc.Delete(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		require.NoError(t, err)

		// Verify profile is also deleted (CASCADE)
		profileCount, _ = svc.Count(ctx, "profiles",
			service.WithWhere(domain.Condition{
				Field:    "user_id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		assert.Equal(t, int64(0), profileCount)
	})

	t.Run("DeleteUserCascadesPosts", func(t *testing.T) {
		// Create user with posts
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "cascade_posts@example.com",
		})

		for i := 0; i < 3; i++ {
			svc.Create(ctx, "posts", map[string]interface{}{
				"title":     "Cascade Post " + string(rune('1'+i)),
				"slug":      "cascade-post-" + string(rune('1'+i)),
				"author_id": user["id"],
			})
		}

		// Verify posts exist
		postCount, _ := svc.Count(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		assert.Equal(t, int64(3), postCount)

		// Delete user - should cascade to posts
		_, err := svc.Delete(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		require.NoError(t, err)

		// Verify posts are deleted
		postCount, _ = svc.Count(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		assert.Equal(t, int64(0), postCount)
	})

	t.Run("DeletePostCascadesComments", func(t *testing.T) {
		// Create user
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "comment_cascade@example.com",
		})

		// Create post
		post, _ := svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Post with comments",
			"slug":      "post-with-comments",
			"author_id": user["id"],
		})

		// Create comments
		for i := 0; i < 5; i++ {
			svc.Create(ctx, "comments", map[string]interface{}{
				"content":   "Comment " + string(rune('1'+i)),
				"post_id":   post["id"],
				"author_id": user["id"],
			})
		}

		// Verify comments exist
		commentCount, _ := svc.Count(ctx, "comments",
			service.WithWhere(domain.Condition{
				Field:    "post_id",
				Operator: domain.Equals,
				Value:    post["id"],
			}))
		assert.Equal(t, int64(5), commentCount)

		// Delete post
		_, err := svc.Delete(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    post["id"],
			}))
		require.NoError(t, err)

		// Verify comments are deleted
		commentCount, _ = svc.Count(ctx, "comments",
			service.WithWhere(domain.Condition{
				Field:    "post_id",
				Operator: domain.Equals,
				Value:    post["id"],
			}))
		assert.Equal(t, int64(0), commentCount)
	})
}

func TestDeleteWithRelationConstraints(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	// Note: With CASCADE delete, this won't fail
	// But we test the behavior is consistent
	t.Run("DeleteParentWithChildren", func(t *testing.T) {
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "parent_user@example.com",
		})

		svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "Child profile",
			"user_id": user["id"],
		})

		// With CASCADE, this should succeed
		affected, err := svc.Delete(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func TestDeleteAllRecords(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	// Create some records
	for i := 0; i < 5; i++ {
		svc.Create(ctx, "users", map[string]interface{}{
			"email": "deleteall" + string(rune('a'+i)) + "@example.com",
		})
	}

	t.Run("DeleteAllWithTrueCondition", func(t *testing.T) {
		// This simulates deleteMany with no filter (all records)
		// Using a condition that matches all records
		affected, err := svc.DeleteMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Gt,
				Value:    0,
			}))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, affected, int64(5))

		// Verify all deleted
		count, _ := svc.Count(ctx, "users")
		assert.Equal(t, int64(0), count)
	})
}
