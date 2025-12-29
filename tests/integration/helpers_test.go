package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/satishbabariya/prisma-go/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/internal/adapters/database/postgres"
	"github.com/satishbabariya/prisma-go/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/internal/core/query/executor"
	"github.com/satishbabariya/prisma-go/internal/service"
	"github.com/satishbabariya/prisma-go/pkg/domain"
)

const (
	// Test database URL - can be overridden with DATABASE_URL env var
	defaultDBURL = "postgresql://prisma:prisma@localhost:5433/prisma_test?sslmode=disable"
)

// setupTestService creates a test service with database connection
func setupTestService(t *testing.T) (*service.QueryService, func()) {
	t.Helper()

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDBURL
	}

	// Skip test if no database available
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	ctx := context.Background()

	// Create adapter
	dbConfig := database.Config{
		URL:            dbURL,
		MaxConnections: 5,
		ConnectTimeout: 30,
	}

	adapter, err := postgres.NewPostgresAdapter(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	if err := adapter.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect adapter: %v", err)
	}

	// Create compiler and executor
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)
	exec := executor.NewQueryExecutor(adapter)

	// Create service
	svc := service.NewQueryService(comp, exec)

	cleanup := func() {
		adapter.Disconnect(ctx)
	}

	return svc, cleanup
}

// cleanupTestData cleans up test data after each test
func cleanupTestData(t *testing.T) {
	t.Helper()

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDBURL
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Logf("Warning: Failed to open database for cleanup: %v", err)
		return
	}
	defer db.Close()

	// Truncate all test tables
	_, err = db.Exec(`
		TRUNCATE TABLE IF EXISTS users, profiles, posts, tags, post_tags, comments, comment_likes 
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test data: %v", err)
	}
}
