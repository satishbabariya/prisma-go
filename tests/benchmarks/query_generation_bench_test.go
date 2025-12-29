package benchmarks

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/pkg/domain"
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
		Selection: domain.Selection{
			Fields: []string{"id", "email"},
		},
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
			{Function: domain.Count, Field: "id"},
			{Function: domain.Sum, Field: "amount"},
			{Function: domain.Avg, Field: "amount"},
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
		Selection: domain.Selection{
			Fields: []string{"title", "content"},
		},
		Relations: []domain.RelationInclusion{
			{Relation: "author"},
			{Relation: "comments"},
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
		CreateData: map[string]interface{}{
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
		UpdateData: map[string]interface{}{
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

	take := 50
	skip := 0
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
		Ordering: []domain.OrderBy{
			{Field: "created_at", Direction: domain.Desc},
		},
		Pagination: domain.Pagination{
			Take: &take,
			Skip: &skip,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(ctx, query)
	}
}
