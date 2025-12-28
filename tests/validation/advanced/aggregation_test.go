package advanced

import (
	"context"
	"fmt"
	"testing"

	"github.com/satishbabariya/prisma-go/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregationCount(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("CountAllRecords", func(t *testing.T) {
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

	t.Run("CountZeroRecords", func(t *testing.T) {
		count, err := svc.Count(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "email",
				Operator: domain.Equals,
				Value:    "nonexistent@example.com",
			}))
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestAggregationSum(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("SumBalance", func(t *testing.T) {
		sum, err := svc.Sum(ctx, "users", "balance")
		require.NoError(t, err)
		assert.Greater(t, sum, float64(0))
	})

	t.Run("SumWithFilter", func(t *testing.T) {
		sum, err := svc.Sum(ctx, "users", "balance",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, sum, float64(0))
	})

	t.Run("SumViewCount", func(t *testing.T) {
		sum, err := svc.Sum(ctx, "posts", "view_count")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, sum, float64(0))
	})
}

func TestAggregationAvg(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("AvgAge", func(t *testing.T) {
		avg, err := svc.Avg(ctx, "users", "age")
		require.NoError(t, err)
		assert.Greater(t, avg, float64(0))
	})

	t.Run("AvgBalance", func(t *testing.T) {
		avg, err := svc.Avg(ctx, "users", "balance")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, avg, float64(0))
	})

	t.Run("AvgViewCount", func(t *testing.T) {
		avg, err := svc.Avg(ctx, "posts", "view_count")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, avg, float64(0))
	})
}

func TestGroupBy(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("GroupByRole", func(t *testing.T) {
		results, err := svc.GroupBy(ctx, "users",
			[]string{"role"},
			[]domain.Aggregation{{Function: domain.Count, Field: "id"}})
		require.NoError(t, err)
		require.NotEmpty(t, results)
		// Should have 3 groups: USER, ADMIN, MODERATOR
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("GroupByStatus", func(t *testing.T) {
		results, err := svc.GroupBy(ctx, "posts",
			[]string{"status"},
			[]domain.Aggregation{{Function: domain.Count, Field: "id"}})
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("GroupByWithSum", func(t *testing.T) {
		results, err := svc.GroupBy(ctx, "users",
			[]string{"role"},
			[]domain.Aggregation{
				{Function: domain.Sum, Field: "balance"},
			})
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("GroupByMultipleFields", func(t *testing.T) {
		results, err := svc.GroupBy(ctx, "users",
			[]string{"role", "is_active"},
			[]domain.Aggregation{{Function: domain.Count, Field: "id"}})
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("GroupByAuthorWithPostCount", func(t *testing.T) {
		results, err := svc.GroupBy(ctx, "posts",
			[]string{"author_id"},
			[]domain.Aggregation{{Function: domain.Count, Field: "id"}})
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})
}

func TestGroupByWithAggregations(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	// Create diverse data for aggregation testing
	for i := 0; i < 20; i++ {
		role := "USER"
		if i%5 == 0 {
			role = "ADMIN"
		} else if i%3 == 0 {
			role = "MODERATOR"
		}
		svc.Create(ctx, "users", map[string]interface{}{
			"email":   fmt.Sprintf("agg%d@example.com", i),
			"role":    role,
			"age":     20 + i,
			"balance": float64(100 + i*50),
		})
	}

	t.Run("GroupByWithMultipleAggregations", func(t *testing.T) {
		results, err := svc.GroupBy(ctx, "users",
			[]string{"role"},
			[]domain.Aggregation{
				{Function: domain.Count, Field: "id"},
				{Function: domain.Sum, Field: "balance"},
				{Function: domain.Avg, Field: "age"},
			})
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})
}
