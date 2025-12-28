package benchmarks

import (
	"context"
	"fmt"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/service"
	"github.com/satishbabariya/prisma-go/v3/tests/validation"
)

// BenchmarkCreateSingle measures single record creation performance
func BenchmarkCreateSingle(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service
	validation.CleanupTables(&testing.T{}, testCtx.DB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Create(ctx, "users", map[string]interface{}{
			"email": fmt.Sprintf("bench%d@example.com", i),
			"name":  fmt.Sprintf("Bench User %d", i),
		})
	}
}

// BenchmarkCreateBatch measures batch creation performance
func BenchmarkCreateBatch(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	batchSize := 100
	batch := make([]map[string]interface{}, batchSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validation.CleanupTables(&testing.T{}, testCtx.DB)
		for j := 0; j < batchSize; j++ {
			batch[j] = map[string]interface{}{
				"email": fmt.Sprintf("batch%d_%d@example.com", i, j),
				"name":  fmt.Sprintf("Batch User %d", j),
			}
		}
		svc.CreateMany(ctx, "users", batch)
	}
}

// BenchmarkFindMany measures find many performance
func BenchmarkFindMany(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(&testing.T{}, testCtx.DB)
	for i := 0; i < 1000; i++ {
		svc.Create(ctx, "users", map[string]interface{}{
			"email":     fmt.Sprintf("perf%d@example.com", i),
			"name":      fmt.Sprintf("Perf User %d", i),
			"is_active": i%2 == 0,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}),
			service.WithOrderBy("created_at", domain.Desc),
			service.WithTake(50))
	}
}

// BenchmarkFindManyWithPagination measures pagination performance
func BenchmarkFindManyWithPagination(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(&testing.T{}, testCtx.DB)
	for i := 0; i < 1000; i++ {
		svc.Create(ctx, "users", map[string]interface{}{
			"email": fmt.Sprintf("page%d@example.com", i),
			"name":  fmt.Sprintf("Page User %d", i),
		})
	}

	pageSize := 50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		offset := (i % 20) * pageSize // Rotate through 20 pages
		svc.FindMany(ctx, "users",
			service.WithOrderBy("id", domain.Asc),
			service.WithSkip(offset),
			service.WithTake(pageSize))
	}
}

// BenchmarkCount measures count aggregation performance
func BenchmarkCount(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(&testing.T{}, testCtx.DB)
	for i := 0; i < 1000; i++ {
		svc.Create(ctx, "users", map[string]interface{}{
			"email": fmt.Sprintf("count%d@example.com", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Count(ctx, "users")
	}
}

// BenchmarkCountWithFilter measures filtered count performance
func BenchmarkCountWithFilter(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(&testing.T{}, testCtx.DB)
	for i := 0; i < 1000; i++ {
		role := "USER"
		if i%10 == 0 {
			role = "ADMIN"
		}
		svc.Create(ctx, "users", map[string]interface{}{
			"email": fmt.Sprintf("filter%d@example.com", i),
			"role":  role,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Count(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "role",
				Operator: domain.Equals,
				Value:    "ADMIN",
			}))
	}
}

// BenchmarkUpdate measures update performance
func BenchmarkUpdate(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(&testing.T{}, testCtx.DB)
	for i := 0; i < 100; i++ {
		svc.Create(ctx, "users", map[string]interface{}{
			"email":   fmt.Sprintf("update%d@example.com", i),
			"balance": 100.00,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := (i % 100) + 1
		svc.Update(ctx, "users",
			map[string]interface{}{
				"balance": float64(i),
			},
			service.WithWhere(domain.Condition{
				Field:    "id",
				Operator: domain.Equals,
				Value:    id,
			}))
	}
}

// BenchmarkComplexQuery measures complex query performance
func BenchmarkComplexQuery(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(&testing.T{}, testCtx.DB)
	for i := 0; i < 500; i++ {
		role := "USER"
		if i%5 == 0 {
			role = "ADMIN"
		} else if i%3 == 0 {
			role = "MODERATOR"
		}
		svc.Create(ctx, "users", map[string]interface{}{
			"email":     fmt.Sprintf("complex%d@example.com", i),
			"role":      role,
			"is_active": i%2 == 0,
			"age":       20 + (i % 50),
			"balance":   float64(i * 10),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.FindMany(ctx, "users",
			service.WithWhere(domain.Condition{
				Field:    "is_active",
				Operator: domain.Equals,
				Value:    true,
			}),
			service.WithOrderBy("balance", domain.Desc),
			service.WithSkip(0),
			service.WithTake(20))
	}
}

// BenchmarkRawQuery measures raw SQL query performance
func BenchmarkRawQuery(b *testing.B) {
	testCtx, cleanup := validation.SetupTestContext(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	svc := testCtx.Service

	// Seed data
	validation.CleanupTables(&testing.T{}, testCtx.DB)
	for i := 0; i < 1000; i++ {
		svc.Create(ctx, "users", map[string]interface{}{
			"email":     fmt.Sprintf("raw%d@example.com", i),
			"is_active": i%2 == 0,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.QueryRaw(ctx, `
			SELECT * FROM users 
			WHERE is_active = $1 
			ORDER BY created_at DESC 
			LIMIT $2
		`, true, 50)
	}
}
