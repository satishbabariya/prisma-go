package relations

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOneToOneRelation(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindUserWithProfile", func(t *testing.T) {
		// Find user that has a profile
		result, err := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Equals,
				Value:    "alice@example.com",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)

		// Find associated profile
		userId := resultSlice[0]["id"]
		profile, err := svc.FindUnique(ctx, "profiles",
			service.WithWhere(domain.Condition{
				Field:    "user_id",
				Operator: domain.Equals,
				Value:    userId,
			}))
		require.NoError(t, err)
		profileSlice := profile.([]map[string]interface{})
		require.Len(t, profileSlice, 1)
	})

	t.Run("FindUserWithoutProfile", func(t *testing.T) {
		// Diana doesn't have a profile
		result, err := svc.FindUnique(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Equals,
				Value:    "diana@example.com",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)

		userId := resultSlice[0]["id"]
		profile, err := svc.FindUnique(ctx, "profiles",
			service.WithWhere(domain.Condition{
				Field:    "user_id",
				Operator: domain.Equals,
				Value:    userId,
			}))
		require.NoError(t, err)
		profileSlice := profile.([]map[string]interface{})
		assert.Len(t, profileSlice, 0)
	})

	t.Run("CreateUserWithProfile", func(t *testing.T) {
		// Create user
		user, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "new_profile_user@example.com",
			"name":  "Profile User",
		})
		require.NoError(t, err)

		// Create profile
		profile, err := svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "New user bio",
			"website": "https://newuser.com",
			"user_id": user["id"],
		})
		require.NoError(t, err)
		assert.Equal(t, user["id"], profile["user_id"])
	})
}

func TestOneToManyRelation(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindUserPosts", func(t *testing.T) {
		// Alice (id=1) has multiple posts
		posts, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		postSlice := posts.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(postSlice), 2)
	})

	t.Run("FindPostsWithOrdering", func(t *testing.T) {
		posts, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    1,
			}),
			service.WithOrderBy("created_at", domain.Desc))
		require.NoError(t, err)
		require.NotNil(t, posts)
	})

	t.Run("CountUserPosts", func(t *testing.T) {
		count, err := svc.Count(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
	})

	t.Run("FindPostComments", func(t *testing.T) {
		// Post 1 has comments
		comments, err := svc.FindMany(ctx, "comments",
			service.WithWhere(domain.Condition{
				Field:    "post_id",
				Operator: domain.Equals,
				Value:    1,
			}))
		require.NoError(t, err)
		commentSlice := comments.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(commentSlice), 1)
	})

	t.Run("PaginateUserPosts", func(t *testing.T) {
		// Create user with many posts
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "many_posts@example.com",
		})

		for i := 0; i < 20; i++ {
			svc.Create(ctx, "posts", map[string]interface{}{
				"title":     "Post " + string(rune('A'+i)),
				"slug":      "many-post-" + string(rune('a'+i)),
				"author_id": user["id"],
			})
		}

		// Paginate
		page1, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    user["id"],
			}),
			service.WithOrderBy("id", domain.Asc),
			service.WithTake(5))
		require.NoError(t, err)
		page1Slice := page1.([]map[string]interface{})
		assert.Len(t, page1Slice, 5)

		page2, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "author_id",
				Operator: domain.Equals,
				Value:    user["id"],
			}),
			service.WithOrderBy("id", domain.Asc),
			service.WithSkip(5),
			service.WithTake(5))
		require.NoError(t, err)
		page2Slice := page2.([]map[string]interface{})
		assert.Len(t, page2Slice, 5)
	})
}

func TestManyToManyRelation(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindPostTags", func(t *testing.T) {
		// Find tags for post 1 via junction table
		results, err := svc.QueryRaw(ctx, `
			SELECT t.* FROM tags t
			INNER JOIN post_tags pt ON t.id = pt.tag_id
			WHERE pt.post_id = $1
		`, 1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("FindTagPosts", func(t *testing.T) {
		// Find posts with tag 1 (Technology)
		results, err := svc.QueryRaw(ctx, `
			SELECT p.* FROM posts p
			INNER JOIN post_tags pt ON p.id = pt.post_id
			WHERE pt.tag_id = $1
		`, 1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("AddTagToPost", func(t *testing.T) {
		// Create new post
		user, _ := svc.FindFirst(ctx, "users")
		userSlice := user.([]map[string]interface{})

		post, _ := svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Tag Test Post",
			"slug":      "tag-test-post",
			"author_id": userSlice[0]["id"],
		})

		// Add tags to post
		for _, tagId := range []int{1, 2, 3} {
			_, err := svc.ExecuteRaw(ctx,
				"INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2)",
				post["id"], tagId)
			require.NoError(t, err)
		}

		// Verify
		results, _ := svc.QueryRaw(ctx, `
			SELECT COUNT(*) as count FROM post_tags WHERE post_id = $1
		`, post["id"])
		assert.Equal(t, int64(3), results[0]["count"])
	})

	t.Run("RemoveTagFromPost", func(t *testing.T) {
		// Remove a tag from post 1
		affected, err := svc.ExecuteRaw(ctx,
			"DELETE FROM post_tags WHERE post_id = $1 AND tag_id = $2",
			1, 3)
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})
}

func TestNestedRelationQueries(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FindUsersWithPosts", func(t *testing.T) {
		// Find users who have at least one post
		results, err := svc.QueryRaw(ctx, `
			SELECT DISTINCT u.* FROM users u
			INNER JOIN posts p ON u.id = p.author_id
		`)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("FindPostsWithComments", func(t *testing.T) {
		// Find posts that have comments
		results, err := svc.QueryRaw(ctx, `
			SELECT p.*, COUNT(c.id) as comment_count 
			FROM posts p
			LEFT JOIN comments c ON p.id = c.post_id
			GROUP BY p.id
			HAVING COUNT(c.id) > 0
		`)
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("ThreeLevelNesting", func(t *testing.T) {
		// User -> Post -> Comment
		results, err := svc.QueryRaw(ctx, `
			SELECT u.email, p.title, c.content
			FROM users u
			INNER JOIN posts p ON u.id = p.author_id
			INNER JOIN comments c ON p.id = c.post_id
			WHERE u.email = $1
		`, "alice@example.com")
		require.NoError(t, err)
		require.NotNil(t, results)
	})
}

func TestRelationCascade(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("DeleteUserCascadesAll", func(t *testing.T) {
		// Create user with profile, posts, and comments
		user, _ := svc.Create(ctx, "users", map[string]interface{}{
			"email": "cascade_all@example.com",
		})

		svc.Create(ctx, "profiles", map[string]interface{}{
			"bio":     "Cascade test",
			"user_id": user["id"],
		})

		post, _ := svc.Create(ctx, "posts", map[string]interface{}{
			"title":     "Cascade Post",
			"slug":      "cascade-post",
			"author_id": user["id"],
		})

		svc.Create(ctx, "comments", map[string]interface{}{
			"content":   "Cascade comment",
			"post_id":   post["id"],
			"author_id": user["id"],
		})

		// Delete user
		_, err := svc.Delete(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    user["id"],
			}))
		require.NoError(t, err)

		// Verify all related data is deleted
		profileCount, _ := svc.Count(ctx, "profiles",
			service.WithWhere(domain.Condition{Field: "user_id", Operator: domain.Equals, Value: user["id"]}))
		postCount, _ := svc.Count(ctx, "posts",
			service.WithWhere(domain.Condition{Field: "author_id", Operator: domain.Equals, Value: user["id"]}))

		assert.Equal(t, int64(0), profileCount)
		assert.Equal(t, int64(0), postCount)
	})
}
