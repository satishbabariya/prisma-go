package filters

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNumericFilterEquals(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("IntegerEquals", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Equals,
				Value:    30,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
		assert.Equal(t, int64(30), resultSlice[0]["age"])
	})

	t.Run("IntegerNotEquals", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.NotEquals,
				Value:    30,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			if r["age"] != nil {
				assert.NotEqual(t, int64(30), r["age"])
			}
		}
	})
}

func TestNumericFilterComparison(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("LessThan", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Lt,
				Value:    30,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			if r["age"] != nil {
				age := r["age"].(int64)
				assert.Less(t, age, int64(30))
			}
		}
	})

	t.Run("LessThanOrEqual", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Lte,
				Value:    30,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			if r["age"] != nil {
				age := r["age"].(int64)
				assert.LessOrEqual(t, age, int64(30))
			}
		}
	})

	t.Run("GreaterThan", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Gt,
				Value:    30,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			if r["age"] != nil {
				age := r["age"].(int64)
				assert.Greater(t, age, int64(30))
			}
		}
	})

	t.Run("GreaterThanOrEqual", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Gte,
				Value:    30,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			if r["age"] != nil {
				age := r["age"].(int64)
				assert.GreaterOrEqual(t, age, int64(30))
			}
		}
	})
}

func TestNumericFilterIn(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("InArrayOfIntegers", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.In,
				Value:    []int{25, 30, 35},
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			age := r["age"].(int64)
			assert.Contains(t, []int64{25, 30, 35}, age)
		}
	})

	t.Run("NotInArrayOfIntegers", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.NotIn,
				Value:    []int{25, 30, 35},
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			if r["age"] != nil {
				age := r["age"].(int64)
				assert.NotContains(t, []int64{25, 30, 35}, age)
			}
		}
	})
}

func TestNumericFilterEdgeCases(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)

	t.Run("ZeroValue", func(t *testing.T) {
		svc.Create(ctx, "users", map[string]interface{}{
			"email":   "zero@example.com",
			"age":     0,
			"balance": 0.00,
		})

		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Equals,
				Value:    0,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
	})

	t.Run("NegativeValue", func(t *testing.T) {
		svc.Create(ctx, "users", map[string]interface{}{
			"email":   "negative@example.com",
			"balance": -100.50,
		})

		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "balance",
				Operator: domain.Lt,
				Value:    0,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
	})

	t.Run("LargeValue", func(t *testing.T) {
		svc.Create(ctx, "users", map[string]interface{}{
			"email": "large@example.com",
			"age":   2147483647, // Max int32
		})

		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "age",
				Operator: domain.Equals,
				Value:    2147483647,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		require.Len(t, resultSlice, 1)
	})
}

func TestDecimalFilters(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("DecimalGreaterThan", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "balance",
				Operator: domain.Gt,
				Value:    500.00,
			}))
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("DecimalRange", func(t *testing.T) {
		// Balance between 250 and 1000
		result, err := svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "balance",
				Operator: domain.Gte,
				Value:    250.00,
			}),
			// Additional filter would need AND support
		)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestViewCountFilters(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	validation.CleanupTables(t, testCtx.DB)
	validation.SeedTestData(t, testCtx.DB)

	t.Run("FilterByViewCount", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "view_count",
				Operator: domain.Gte,
				Value:    100,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			viewCount := r["view_count"].(int64)
			assert.GreaterOrEqual(t, viewCount, int64(100))
		}
	})

	t.Run("FilterByZeroViewCount", func(t *testing.T) {
		result, err := svc.FindMany(ctx, "posts",
			service.WithWhere(domain.Condition{
				Field:    "view_count",
				Operator: domain.Equals,
				Value:    0,
			}))
		require.NoError(t, err)
		resultSlice := result.([]map[string]interface{})
		for _, r := range resultSlice {
			assert.Equal(t, int64(0), r["view_count"])
		}
	})
}
