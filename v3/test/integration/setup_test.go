package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database/postgres"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/executor"
	"github.com/satishbabariya/prisma-go/v3/internal/service"
	"github.com/stretchr/testify/require"
)

const (
	testDBURL = "postgresql://prisma:prisma@localhost:5433/prisma_test?sslmode=disable"
)

// TestMain sets up and tears down the test database
func TestMain(m *testing.M) {
	// Check if integration tests should run
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
		os.Exit(0)
	}

	// Wait for database to be ready
	if err := waitForDB(); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Setup database schema
	if err := setupSchema(); err != nil {
		fmt.Printf("Failed to setup schema: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()

	os.Exit(code)
}

func waitForDB() error {
	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database")
		case <-ticker.C:
			if err := db.Ping(); err == nil {
				return nil
			}
		}
	}
}

func setupSchema() error {
	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		return err
	}
	defer db.Close()

	schema := `
	DROP TABLE IF EXISTS comments CASCADE;
	DROP TABLE IF EXISTS posts CASCADE;
	DROP TABLE IF EXISTS profiles CASCADE;
	DROP TABLE IF EXISTS users CASCADE;

	CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255),
		status VARCHAR(50) DEFAULT 'active',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		deleted_at TIMESTAMP
	);

	CREATE TABLE profiles (
		id SERIAL PRIMARY KEY,
		bio TEXT,
		avatar VARCHAR(255),
		user_id INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE posts (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		content TEXT,
		published BOOLEAN DEFAULT FALSE,
		category VARCHAR(100),
		tags TEXT[],
		author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX idx_posts_author ON posts(author_id);
	CREATE INDEX idx_posts_category ON posts(category);

	CREATE TABLE comments (
		id SERIAL PRIMARY KEY,
		text TEXT NOT NULL,
		post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX idx_comments_post ON comments(post_id);
	`

	_, err = db.Exec(schema)
	return err
}

func teardownSchema() {
	db, _ := sql.Open("postgres", testDBURL)
	if db != nil {
		db.Exec("DROP TABLE IF EXISTS comments CASCADE")
		db.Exec("DROP TABLE IF EXISTS posts CASCADE")
		db.Exec("DROP TABLE IF EXISTS profiles CASCADE")
		db.Exec("DROP TABLE IF EXISTS users CASCADE")
		db.Close()
	}
}

func setupTestService(t *testing.T) (*service.QueryService, func()) {
	ctx := context.Background()

	// Create adapter
	dbConfig := database.Config{
		URL:            testDBURL,
		MaxConnections: 5,
		ConnectTimeout: 30,
	}

	adapter, err := postgres.NewPostgresAdapter(dbConfig)
	require.NoError(t, err)

	err = adapter.Connect(ctx)
	require.NoError(t, err)

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

func cleanupTestData(t *testing.T) {
	db, err := sql.Open("postgres", testDBURL)
	require.NoError(t, err)
	defer db.Close()

	db.Exec("TRUNCATE users, profiles, posts, comments RESTART IDENTITY CASCADE")
}
