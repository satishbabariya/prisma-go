package benchmarks

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// BenchmarkSimpleSelect tests simple SELECT query generation.
func BenchmarkSimpleSelect(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "User",
		Operation: domain.FindMany,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}

// BenchmarkComplexWhere tests complex WHERE clause generation.
func BenchmarkComplexWhere(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "User",
		Operation: domain.FindMany,
		Select:    []string{"id", "email"},
		Filter: domain.Filter{
			Operator: domain.AND,
			Conditions: []domain.Condition{
				{Field: "active", Operator: domain.Equals, Value: true},
				{Field: "email", Operator: domain.Contains, Value: "@example.com"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}

// BenchmarkAggregation tests aggregation query generation.
func BenchmarkAggregation(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "Order",
		Operation: domain.Aggregate,
		Aggregations: []domain.Aggregation{
			{Function: domain.Count, Field: "id", Alias: "total_orders"},
			{Function: domain.Sum, Field: "amount", Alias: "total_amount"},
			{Function: domain.Avg, Field: "amount", Alias: "avg_amount"},
		},
		GroupBy: []string{"user_id"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}

// BenchmarkJoinGeneration tests JOIN clause generation.
func BenchmarkJoinGeneration(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "Post",
		Operation: domain.FindMany,
		Select:    []string{"title", "content"},
		Relations: []domain.RelationInclusion{
			{Relation: "author", Select: []string{"name", "email"}},
			{Relation: "comments", Select: []string{"content"}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}

// BenchmarkInsert tests INSERT query generation.
func BenchmarkInsert(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "User",
		Operation: domain.Create,
		Data: map[string]interface{}{
			"email":  "test@example.com",
			"name":   "Test User",
			"active": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}

// BenchmarkUpdate tests UPDATE query generation.
func BenchmarkUpdate(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "User",
		Operation: domain.Update,
		Data: map[string]interface{}{
			"name":   "Updated Name",
			"active": false,
		},
		Filter: domain.Filter{
			Conditions: []domain.Condition{
				{Field: "id", Operator: domain.Equals, Value: 1},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}

// BenchmarkDelete tests DELETE query generation.
func BenchmarkDelete(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "User",
		Operation: domain.Delete,
		Filter: domain.Filter{
			Conditions: []domain.Condition{
				{Field: "id", Operator: domain.Equals, Value: 1},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}

// BenchmarkComplexNestedQuery tests complex nested query generation.
func BenchmarkComplexNestedQuery(b *testing.B) {
	c := compiler.NewSQLCompiler(domain.PostgreSQL)
	ctx := context.Background()

	query := &domain.Query{
		Model:     "User",
		Operation: domain.FindMany,
		Filter: domain.Filter{
			Operator: domain.AND,
			Conditions: []domain.Condition{
				{Field: "active", Operator: domain.Equals, Value: true},
				{Field: "role", Operator: domain.In, Value: []interface{}{"admin", "moderator"}},
			},
		},
		OrderBy: []domain.OrderBy{
			{Field: "created_at", Direction: domain.DESC},
		},
		Take: 50,
		Skip: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}
