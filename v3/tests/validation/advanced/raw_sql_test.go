package advanced

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryRaw(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("SimpleSelect", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, "SELECT * FROM users WHERE email = $1", "alice@example.com")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "alice@example.com", results[0]["email"])
	})

	t.Run("SelectWithMultipleParams", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx,
			"SELECT * FROM users WHERE is_active = $1 AND role = $2",
			true, "ADMIN")
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("SelectWithJoin", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, `
			SELECT u.email, p.bio 
			FROM users u 
			LEFT JOIN profiles p ON u.id = p.user_id 
			WHERE u.email = $1
		`, "alice@example.com")
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("SelectWithAggregation", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, `
			SELECT role, COUNT(*) as count 
			FROM users 
			GROUP BY role
		`)
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("SelectWithOrderAndLimit", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, `
			SELECT email, name FROM users ORDER BY created_at DESC LIMIT $1
		`, 3)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 3)
	})

	t.Run("EmptyResult", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, "SELECT * FROM users WHERE email = $1", "nonexistent@example.com")
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestExecuteRaw(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("UpdateStatement", func(t *testing.T) {
		affected, err := svc.ExecuteRaw(ctx,
			"UPDATE users SET name = $1 WHERE email = $2",
			"Raw Updated", "alice@example.com")
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)

		// Verify update
		results, _ := svc.QueryRaw(ctx, "SELECT name FROM users WHERE email = $1", "alice@example.com")
		assert.Equal(t, "Raw Updated", results[0]["name"])
	})

	t.Run("UpdateManyRows", func(t *testing.T) {
		affected, err := svc.ExecuteRaw(ctx,
			"UPDATE users SET is_active = $1 WHERE role = $2",
			true, "USER")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, affected, int64(1))
	})

	t.Run("DeleteStatement", func(t *testing.T) {
		// Create user to delete
		svc.ExecuteRaw(ctx, "INSERT INTO users (email, name) VALUES ($1, $2)", "to_raw_delete@example.com", "Delete Me")

		affected, err := svc.ExecuteRaw(ctx,
			"DELETE FROM users WHERE email = $1",
			"to_raw_delete@example.com")
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})

	t.Run("InsertStatement", func(t *testing.T) {
		affected, err := svc.ExecuteRaw(ctx,
			"INSERT INTO users (email, name) VALUES ($1, $2)",
			"raw_insert@example.com", "Raw Insert User")
		require.NoError(t, err)
		assert.Equal(t, int64(1), affected)
	})

	t.Run("UpdateNoMatch", func(t *testing.T) {
		affected, err := svc.ExecuteRaw(ctx,
			"UPDATE users SET name = $1 WHERE email = $2",
			"Ghost", "ghost@nonexistent.com")
		require.NoError(t, err)
		assert.Equal(t, int64(0), affected)
	})
}

func TestRawQueryWithComplexSQL(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("SubquerySelect", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, `
			SELECT u.* FROM users u 
			WHERE u.id IN (
				SELECT author_id FROM posts WHERE status = $1
			)
		`, "PUBLISHED")
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("WithCTE", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, `
			WITH active_users AS (
				SELECT * FROM users WHERE is_active = true
			)
			SELECT email, name FROM active_users WHERE role = $1
		`, "ADMIN")
		require.NoError(t, err)
		require.NotNil(t, results)
	})

	t.Run("WindowFunction", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, `
			SELECT 
				email, 
				balance,
				ROW_NUMBER() OVER (ORDER BY balance DESC) as rank
			FROM users
			WHERE balance IS NOT NULL
		`)
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("CaseStatement", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, `
			SELECT 
				email,
				CASE 
					WHEN age < 25 THEN 'young'
					WHEN age < 35 THEN 'adult'
					ELSE 'senior'
				END as age_group
			FROM users
			WHERE age IS NOT NULL
		`)
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})
}

func TestRawQueryParameterBinding(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("StringParameter", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, "SELECT * FROM users WHERE email = $1", "alice@example.com")
		require.NoError(t, err)
		require.Len(t, results, 1)
	})

	t.Run("IntegerParameter", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, "SELECT * FROM users WHERE age >= $1", 30)
		require.NoError(t, err)
		require.NotNil(t, results)
	})

	t.Run("BooleanParameter", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, "SELECT * FROM users WHERE is_active = $1", true)
		require.NoError(t, err)
		require.NotNil(t, results)
	})

	t.Run("NullParameter", func(t *testing.T) {
		results, err := svc.QueryRaw(ctx, "SELECT * FROM users WHERE name IS NULL")
		require.NoError(t, err)
		require.NotNil(t, results)
	})

	t.Run("ArrayLikeParameter", func(t *testing.T) {
		// Using IN clause with multiple values
		results, err := svc.QueryRaw(ctx, `
			SELECT * FROM users WHERE email IN ($1, $2)
		`, "alice@example.com", "bob@example.com")
		require.NoError(t, err)
		require.Len(t, results, 2)
	})
}
