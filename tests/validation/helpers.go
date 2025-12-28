// Package validation provides helper functions for validation tests
package validation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/satishbabariya/prisma-go/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/internal/adapters/database/postgres"
	"github.com/satishbabariya/prisma-go/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/internal/core/query/executor"
	"github.com/satishbabariya/prisma-go/internal/service"
)

const (
	// DefaultDBURL is the default database URL for testing
	DefaultDBURL = "postgresql://prisma:prisma@localhost:5433/prisma_test?sslmode=disable"
)

// TestContext holds shared test dependencies
type TestContext struct {
	DB       *sql.DB
	Harness  *TestHarness
	Service  *service.QueryService
	Adapter  database.Adapter
	Compiler *compiler.SQLCompiler
	Executor *executor.QueryExecutor
}

// GetDBURL returns the database URL from environment or default
func GetDBURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return DefaultDBURL
}

// SetupValidationSchema creates the comprehensive test schema
func SetupValidationSchema(dbURL string) error {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()

	schema := `
	-- Drop existing tables
	DROP TABLE IF EXISTS _prisma_migrations CASCADE;
	DROP TABLE IF EXISTS comment_likes CASCADE;
	DROP TABLE IF EXISTS post_tags CASCADE;
	DROP TABLE IF EXISTS tags CASCADE;
	DROP TABLE IF EXISTS comments CASCADE;
	DROP TABLE IF EXISTS posts CASCADE;
	DROP TABLE IF EXISTS profiles CASCADE;
	DROP TABLE IF EXISTS users CASCADE;
	DROP TYPE IF EXISTS user_role CASCADE;
	DROP TYPE IF EXISTS post_status CASCADE;

	-- Enums
	CREATE TYPE user_role AS ENUM ('USER', 'ADMIN', 'MODERATOR');
	CREATE TYPE post_status AS ENUM ('DRAFT', 'PUBLISHED', 'ARCHIVED');

	-- Users table (comprehensive field types)
	CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255),
		password_hash VARCHAR(255),
		role user_role DEFAULT 'USER',
		is_active BOOLEAN DEFAULT TRUE,
		age INTEGER,
		balance DECIMAL(10,2) DEFAULT 0.00,
		metadata JSONB,
		avatar BYTEA,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		deleted_at TIMESTAMP WITH TIME ZONE
	);

	-- Profiles table (1:1 relation)
	CREATE TABLE profiles (
		id SERIAL PRIMARY KEY,
		bio TEXT,
		website VARCHAR(255),
		avatar_url VARCHAR(255),
		location VARCHAR(255),
		birth_date DATE,
		user_id INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Posts table (1:N relation)
	CREATE TABLE posts (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		slug VARCHAR(255) UNIQUE,
		content TEXT,
		excerpt TEXT,
		status post_status DEFAULT 'DRAFT',
		view_count INTEGER DEFAULT 0,
		like_count INTEGER DEFAULT 0,
		published_at TIMESTAMP WITH TIME ZONE,
		author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Tags table (for M:N relation)
	CREATE TABLE tags (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) UNIQUE NOT NULL,
		slug VARCHAR(100) UNIQUE NOT NULL,
		description TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Post-Tags junction table (M:N relation)
	CREATE TABLE post_tags (
		post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (post_id, tag_id)
	);

	-- Comments table (1:N with posts)
	CREATE TABLE comments (
		id SERIAL PRIMARY KEY,
		content TEXT NOT NULL,
		is_approved BOOLEAN DEFAULT FALSE,
		post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Comment Likes (for testing aggregations)
	CREATE TABLE comment_likes (
		id SERIAL PRIMARY KEY,
		comment_id INTEGER NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		UNIQUE(comment_id, user_id)
	);

	-- Indexes
	CREATE INDEX idx_users_email ON users(email);
	CREATE INDEX idx_users_role ON users(role);
	CREATE INDEX idx_users_created_at ON users(created_at);
	CREATE INDEX idx_posts_author ON posts(author_id);
	CREATE INDEX idx_posts_status ON posts(status);
	CREATE INDEX idx_posts_published_at ON posts(published_at);
	CREATE INDEX idx_comments_post ON comments(post_id);
	CREATE INDEX idx_comments_author ON comments(author_id);
	`

	_, err = db.Exec(schema)
	return err
}

var schemaSetupOnce sync.Once
var schemaSetupErr error
var dbMutex sync.Mutex // Mutex to prevent concurrent table operations

// SetupTestContext creates a full test context with all dependencies
func SetupTestContext(t *testing.T) (*TestContext, func()) {
	dbURL := GetDBURL()

	// Check if validation tests should run
	if os.Getenv("RUN_VALIDATION_TESTS") != "true" {
		t.Skip("Skipping validation tests. Set RUN_VALIDATION_TESTS=true to run.")
	}

	// Wait for database to be ready
	ctx := context.Background()
	if err := WaitForDB(ctx, dbURL, 30*time.Second); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Ensure schema exists - only run once per test session
	schemaSetupOnce.Do(func() {
		schemaSetupErr = SetupValidationSchema(dbURL)
	})
	if schemaSetupErr != nil {
		t.Fatalf("Failed to setup schema: %v", schemaSetupErr)
	}

	// Create harness
	harness, err := NewTestHarness(dbURL)
	if err != nil {
		t.Fatalf("Failed to create harness: %v", err)
	}

	// Create adapter
	dbConfig := database.Config{
		URL:            dbURL,
		MaxConnections: 5, // Reduced to prevent contention
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

	testCtx := &TestContext{
		DB:       harness.DB(),
		Harness:  harness,
		Service:  svc,
		Adapter:  adapter,
		Compiler: comp,
		Executor: exec,
	}

	cleanup := func() {
		adapter.Disconnect(ctx)
		harness.Close()
	}

	return testCtx, cleanup
}

// CleanupTables truncates all test tables
func CleanupTables(t *testing.T, db *sql.DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	cleanupTablesLocked(t, db)
}

// cleanupTablesLocked performs the actual cleanup (caller must hold dbMutex)
func cleanupTablesLocked(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`
		TRUNCATE users, profiles, posts, tags, post_tags, comments, comment_likes 
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Logf("Warning: Failed to cleanup tables: %v", err)
	}
}

// CleanupAndSeed atomically cleans up and seeds the database (holds lock for both)
func CleanupAndSeed(t *testing.T, db *sql.DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	cleanupTablesLocked(t, db)
	seedTestDataLocked(t, db)
}

// SeedTestData inserts standard test data
func SeedTestData(t *testing.T, db *sql.DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	seedTestDataLocked(t, db)
}

// seedTestDataLocked performs the actual seeding (caller must hold dbMutex)
func seedTestDataLocked(t *testing.T, db *sql.DB) {
	// Insert test users
	_, err := db.Exec(`
		INSERT INTO users (email, name, role, is_active, age, balance) VALUES
		('alice@example.com', 'Alice Smith', 'ADMIN', true, 30, 1000.50),
		('bob@example.com', 'Bob Jones', 'USER', true, 25, 500.00),
		('charlie@example.com', 'Charlie Brown', 'MODERATOR', true, 35, 750.25),
		('diana@example.com', 'Diana Prince', 'USER', false, 28, 250.00),
		('eve@example.com', NULL, 'USER', true, NULL, 0.00)
	`)
	if err != nil {
		t.Fatalf("Failed to seed users: %v", err)
	}

	// Insert profiles
	_, err = db.Exec(`
		INSERT INTO profiles (bio, website, user_id) VALUES
		('Alice bio', 'https://alice.com', 1),
		('Bob bio', NULL, 2),
		('Charlie bio', 'https://charlie.dev', 3)
	`)
	if err != nil {
		t.Fatalf("Failed to seed profiles: %v", err)
	}

	// Insert tags
	_, err = db.Exec(`
		INSERT INTO tags (name, slug) VALUES
		('Technology', 'technology'),
		('Programming', 'programming'),
		('Go', 'go'),
		('Database', 'database')
	`)
	if err != nil {
		t.Fatalf("Failed to seed tags: %v", err)
	}

	// Insert posts
	_, err = db.Exec(`
		INSERT INTO posts (title, slug, content, status, view_count, author_id, published_at) VALUES
		('First Post', 'first-post', 'Content of first post', 'PUBLISHED', 100, 1, NOW()),
		('Draft Post', 'draft-post', 'Draft content', 'DRAFT', 0, 1, NULL),
		('Bob Post', 'bob-post', 'Bob writes stuff', 'PUBLISHED', 50, 2, NOW()),
		('Archived Post', 'archived-post', 'Old content', 'ARCHIVED', 200, 1, NOW() - INTERVAL '1 year')
	`)
	if err != nil {
		t.Fatalf("Failed to seed posts: %v", err)
	}

	// Insert post-tags
	_, err = db.Exec(`
		INSERT INTO post_tags (post_id, tag_id) VALUES
		(1, 1), (1, 2), (1, 3),
		(2, 2),
		(3, 1), (3, 4)
	`)
	if err != nil {
		t.Fatalf("Failed to seed post_tags: %v", err)
	}

	// Insert comments
	_, err = db.Exec(`
		INSERT INTO comments (content, is_approved, post_id, author_id) VALUES
		('Great post!', true, 1, 2),
		('Thanks for sharing', true, 1, 3),
		('Needs work', false, 2, 2),
		('Interesting', true, 3, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to seed comments: %v", err)
	}
}

// SkipIfNoValidation skips the test if validation tests are not enabled
func SkipIfNoValidation(t *testing.T) {
	if os.Getenv("RUN_VALIDATION_TESTS") != "true" {
		t.Skip("Skipping validation tests. Set RUN_VALIDATION_TESTS=true to run.")
	}
}

// MustExec executes a SQL query and fails the test on error
func MustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	_, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
}

// MustQuery executes a query and returns results, failing the test on error
func MustQuery(t *testing.T, db *sql.DB, query string, args ...interface{}) *sql.Rows {
	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	return rows
}

// RunWithDB is a helper to run a function with a database connection
func RunWithDB(t *testing.T, fn func(db *sql.DB)) {
	dbURL := GetDBURL()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	fn(db)
}

// CreateTestUser is a helper to create a test user
func CreateTestUser(t *testing.T, db *sql.DB, email string) int {
	var id int
	err := db.QueryRow(`
		INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id
	`, email, "Test User").Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return id
}

// CreateTestPost is a helper to create a test post
func CreateTestPost(t *testing.T, db *sql.DB, title string, authorID int) int {
	var id int
	slug := fmt.Sprintf("test-post-%d", time.Now().UnixNano())
	err := db.QueryRow(`
		INSERT INTO posts (title, slug, author_id) VALUES ($1, $2, $3) RETURNING id
	`, title, slug, authorID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}
	return id
}
