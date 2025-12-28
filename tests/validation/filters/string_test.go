package filters

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/pkg/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringFilterEquals(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("ExactMatch", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Equals,
				Value:    "alice@example.com",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
		assert.Equal(t, "alice@example.com", resultSlice[0]["email"])
	})

	t.Run("NotEquals", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.NotEquals,
				Value:    "alice@example.com",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.NotEqual(t, "alice@example.com", r["email"])
		}
	})

	t.Run("EmptyStringMatch", func(t *testing.T) {
		// Create user with empty name
		svc.Create(ctx, "users", map[string]interface{}{
			"email": "empty_name@example.com",
			"name":  "",
		})

		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "name",
				Operator: domain.Equals,
				Value:    "",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.Equal(t, "", r["name"])
		}
	})
}

func TestStringFilterContains(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("ContainsSubstring", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Contains,
				Value:    "@example.com",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		// All seeded users have @example.com
		assert.GreaterOrEqual(t, len(resultSlice), 5)
	})

	t.Run("ContainsNoMatch", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Contains,
				Value:    "@nonexistent.com",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 0)
	})
}

func TestStringFilterStartsWith(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("StartsWithMatch", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.StartsWith,
				Value:    "alice",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
		assert.Equal(t, "alice@example.com", resultSlice[0]["email"])
	})

	t.Run("StartsWithNoMatch", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.StartsWith,
				Value:    "xyz",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 0)
	})
}

func TestStringFilterEndsWith(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("EndsWithMatch", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.EndsWith,
				Value:    "@example.com",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.GreaterOrEqual(t, len(resultSlice), 5)
	})
}

func TestStringFilterIn(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("InArrayMatch", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.In,
				Value:    []string{"alice@example.com", "bob@example.com"},
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 2)
	})

	t.Run("InArrayNoMatch", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.In,
				Value:    []string{"nonexistent1@example.com", "nonexistent2@example.com"},
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		assert.Len(t, resultSlice, 0)
	})

	t.Run("NotIn", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.NotIn,
				Value:    []string{"alice@example.com", "bob@example.com"},
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.NotEqual(t, "alice@example.com", r["email"])
			assert.NotEqual(t, "bob@example.com", r["email"])
		}
	})
}

func TestStringFilterSpecialCharacters(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("UnicodeCharacters", func(t *testing.T) {
		// Create user with unicode name
		user, err := svc.Create(ctx, "users", map[string]interface{}{
			"email": "unicode@example.com",
			"name":  "日本語名前",
		})
		require.NoError(t, err)
		_ = user

		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "name",
				Operator: domain.Equals,
				Value:    "日本語名前",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		// Create user with special characters
		svc.Create(ctx, "users", map[string]interface{}{
			"email": "special@example.com",
			"name":  "O'Brien-Smith",
		})

		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "name",
				Operator: domain.Equals,
				Value:    "O'Brien-Smith",
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
	})

	t.Run("SQLInjectionPrevention", func(t *testing.T) {
		// This should not cause SQL injection
		_, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "name",
				Operator: domain.Equals,
				Value:    "'; DROP TABLE users; --",
			}))
		require.NoError(t, err) // Should not error, just return empty
	})
}
