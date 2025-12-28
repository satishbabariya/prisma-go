package runtime

import (
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator_Count(t *testing.T) {
	// Note: This test would require a mock executor or database connection
	// For now, testing the structure and API
	t.Run("Count builds correct query structure", func(t *testing.T) {
		// This is a structural test - in production you'd use a mock executor
		agg := NewAggregator("User", nil)
		agg.Where(domain.Condition{
			Field:    "active",
			Operator: domain.Equals,
			Value:    true,
		})

		assert.Equal(t, "User", agg.model)
		assert.Len(t, agg.filter.Conditions, 1)
		assert.Equal(t, "active", agg.filter.Conditions[0].Field)
	})
}

func TestAggregator_GroupBy(t *testing.T) {
	t.Run("GroupBy adds fields correctly", func(t *testing.T) {
		agg := NewAggregator("Order", nil)
		agg.GroupBy("status", "customer_id")

		assert.Equal(t, []string{"status", "customer_id"}, agg.groupBy)
	})
}

func TestAggregator_Having(t *testing.T) {
	t.Run("Having adds conditions correctly", func(t *testing.T) {
		agg := NewAggregator("Order", nil)
		agg.GroupBy("customer_id").
			Having(domain.Condition{
				Field:    "COUNT(*)",
				Operator: domain.Gt,
				Value:    5,
			})

		require.Len(t, agg.having.Conditions, 1)
		assert.Equal(t, "COUNT(*)", agg.having.Conditions[0].Field)
		assert.Equal(t, domain.Gt, agg.having.Conditions[0].Operator)
	})
}

func TestAggregateOptions(t *testing.T) {
	t.Run("AggregateOptions structure", func(t *testing.T) {
		opts := AggregateOptions{
			Count:     true,
			AvgFields: []string{"price", "quantity"},
			SumFields: []string{"total"},
			MinFields: []string{"created_at"},
			MaxFields: []string{"updated_at"},
		}

		assert.True(t, opts.Count)
		assert.Equal(t, []string{"price", "quantity"}, opts.AvgFields)
		assert.Equal(t, []string{"total"}, opts.SumFields)
		assert.Equal(t, []string{"created_at"}, opts.MinFields)
		assert.Equal(t, []string{"updated_at"}, opts.MaxFields)
	})
}

func TestAggregateResult(t *testing.T) {
	t.Run("AggregateResult structure", func(t *testing.T) {
		count := int64(100)
		result := &AggregateResult{
			Count: &count,
			Avg: map[string]float64{
				"price": 29.99,
			},
			Sum: map[string]interface{}{
				"quantity": int64(500),
			},
			Min: map[string]interface{}{
				"created_at": "2024-01-01",
			},
			Max: map[string]interface{}{
				"updated_at": "2024-12-31",
			},
		}

		assert.NotNil(t, result.Count)
		assert.Equal(t, int64(100), *result.Count)
		assert.Equal(t, 29.99, result.Avg["price"])
		assert.Equal(t, int64(500), result.Sum["quantity"])
		assert.Equal(t, "2024-01-01", result.Min["created_at"])
		assert.Equal(t, "2024-12-31", result.Max["updated_at"])
	})
}

func TestAggregator_ChainedOperations(t *testing.T) {
	t.Run("Chained operations work correctly", func(t *testing.T) {
		agg := NewAggregator("Product", nil).
			Where(domain.Condition{
				Field:    "category",
				Operator: domain.Equals,
				Value:    "electronics",
			}).
			Where(domain.Condition{
				Field:    "in_stock",
				Operator: domain.Equals,
				Value:    true,
			}).
			GroupBy("brand").
			Having(domain.Condition{
				Field:    "COUNT(*)",
				Operator: domain.Gte,
				Value:    10,
			})

		assert.Equal(t, "Product", agg.model)
		assert.Len(t, agg.filter.Conditions, 2)
		assert.Equal(t, []string{"brand"}, agg.groupBy)
		assert.Len(t, agg.having.Conditions, 1)
	})
}
