package errors

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniqueConstraintViolation(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("DuplicateEmail", func(t *testing.T) {
		// Create first user
		_, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "unique@example.com",
			"name":  "First User",
		})
		require.NoError(t, err)

		// Try to create duplicate
		_, err = svc.Create(ctx, "users", map[string]interface{}{
			"email": "unique@example.com",
			"name":  "Second User",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("DuplicateSlug", func(t *testing.T) {
		// Create user for posts
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "slug_author@example.com",
		})

		// Create first post
		_, err := svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "First Post",
			"slug":      "unique-slug",
			"author_id": user["id"],
		})
		require.NoError(t, err)

		// Try to create post with same slug
		_, err = svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Second Post",
			"slug":      "unique-slug",
			"author_id": user["id"],
		})
		require.Error(t, err)
	})

	t.Run("UpdateCausesDuplicate", func(t *testing.T) {
		// Create two users
		user1, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "user1_update@example.com",
		})
		_, _ = svc.Create(ctx, "users", map[string]interface{}{
			"email": "user2_update@example.com",
		})

		// Try to update user1 to have user2's email
		_, err := svc.Update(ctx, "users",
			map[string]interface{}{"email": "user2_update@example.com"},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user1["id"],
			}))
		require.Error(t, err)
	})
}

func TestForeignKeyViolation(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("InvalidForeignKey", func(t *testing.T) {
		// Try to create post with non-existent author
		_, err := svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Orphan Post",
			"slug":      "orphan-post",
			"author_id": 99999, // Non-existent user
		})
		require.Error(t, err)
		// PostgreSQL returns foreign key violation
	})

	t.Run("InvalidProfileUserID", func(t *testing.T) {
		// Try to create profile for non-existent user
		_, err := svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "Orphan profile",
			"user_id": 99999,
		})
		require.Error(t, err)
	})

	t.Run("InvalidCommentPostID", func(t *testing.T) {
		// Create user
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "comment_author@example.com",
		})

		// Try to create comment on non-existent post
		_, err := svc.Create(ctx, "comments", map[string]interface{}{
			"content":   "Orphan comment",
			"post_id":   99999,
			"author_id": user["id"],
		})
		require.Error(t, err)
	})
}

func TestNotFoundError(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

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

	t.Run("FindFirstOrThrowEmpty", func(t *testing.T) {
		_, err := svc.FindFirstOrThrow(ctx, "users")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no users record was found")
	})

	t.Run("FindUniqueReturnsEmpty", func(t *testing.T) {
		result, err := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    99999,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 0)
	})
}

func TestNotNullViolation(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("MissingRequiredEmail", func(t *testing.T) {
		// Try to create user without email (required field)
		_, err := svc.Create(ctx, "users", map[string]interface{}{
			"name": "No Email User",
		})
		require.Error(t, err)
	})

	t.Run("MissingRequiredTitle", func(t *testing.T) {
		// Create user first
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "post_author@example.com",
		})

		// Try to create post without title
		_, err := svc.Create(ctx, "posts", map[string]interface{}{
			"content":   "Content without title",
			"author_id": user["id"],
		})
		require.Error(t, err)
	})

	t.Run("MissingRequiredCommentContent", func(t *testing.T) {
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "comment_user@example.com",
		})
		post, _ := svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Post for Comment",
			"slug":      "post-for-comment",
			"author_id": user["id"],
		})

		// Try to create comment without content
		_, err := svc.Create(ctx, "comments", map[string]interface{}{
			"post_id":   post["id"],
			"author_id": user["id"],
		})
		require.Error(t, err)
	})
}

func TestInvalidEnumValue(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("InvalidUserRole", func(t *testing.T) {
		_, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "invalid_role@example.com",
			"role":  "INVALID_ROLE",
		})
		require.Error(t, err)
	})

	t.Run("InvalidPostStatus", func(t *testing.T) {
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "post_status_author@example.com",
		})

		_, err := svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Invalid Status Post",
			"slug":      "invalid-status",
			"status":    "INVALID_STATUS",
			"author_id": user["id"],
		})
		require.Error(t, err)
	})
}

func TestDataTypeErrors(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("InvalidIntegerType", func(t *testing.T) {
		_, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "type_test@example.com",
			"age":   "not_a_number",
		})
		require.Error(t, err)
	})

	t.Run("InvalidBooleanType", func(t *testing.T) {
		_, err := svc.Create(ctx, "users", map[string]interface{}{
			"email":     "bool_test@example.com",
			"is_active": "maybe",
		})
		require.Error(t, err)
	})
}
