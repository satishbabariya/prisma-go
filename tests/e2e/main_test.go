package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/psl/parsing"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
	"github.com/satishbabariya/prisma-go/query/compiler"
	"github.com/satishbabariya/prisma-go/runtime/client"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestConfig holds configuration for E2E tests
type TestConfig struct {
	Provider     string
	DatabaseURL  string
	TestDatabase string
	SchemaPath   string
}

// TestSuite is a main E2E test suite
type TestSuite struct {
	suite.Suite
	config       *TestConfig
	db           *sql.DB
	prisma       *client.PrismaClient
	schema       *ast.SchemaAst
	introspector introspect.Introspector
	compiler     *compiler.Compiler
	tempDir      string
}

// getTestConfigs returns all supported database providers for testing
func getTestConfigs() []TestConfig {
	return []TestConfig{
		{
			Provider:     "postgresql",
			DatabaseURL:  getEnvOrDefault("POSTGRESQL_TEST_URL", "postgres://postgres:password@localhost:5432/prisma_test?sslmode=disable"),
			TestDatabase: "prisma_test",
			SchemaPath:   "testdata/schema.prisma",
		},
		{
			Provider:     "mysql",
			DatabaseURL:  getEnvOrDefault("MYSQL_TEST_URL", "mysql://root:password@tcp(localhost:3306)/prisma_test"),
			TestDatabase: "prisma_test",
			SchemaPath:   "testdata/schema.prisma",
		},
		{
			Provider:     "sqlite",
			DatabaseURL:  getEnvOrDefault("SQLITE_TEST_URL", "file:./test.db?cache=shared"),
			TestDatabase: "test.db",
			SchemaPath:   "testdata/schema.prisma",
		},
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupSuite runs once per test suite
func (suite *TestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite.T().Logf("Setting up E2E test suite for provider: %s", suite.config.Provider)
	suite.T().Logf("Database URL: %s", suite.config.DatabaseURL)

	// Connect to database
	db, err := sql.Open(getDriverName(suite.config.Provider), suite.config.DatabaseURL)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), db)

	// SQLite-specific setup: enable foreign keys and WAL mode for better concurrency
	if suite.config.Provider == "sqlite" {
		_, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
		if err != nil {
			suite.T().Logf("Warning: Could not enable foreign keys: %v", err)
		}
		_, err = db.ExecContext(ctx, "PRAGMA journal_mode = WAL")
		if err != nil {
			suite.T().Logf("Warning: Could not enable WAL mode: %v", err)
		}
	}

	// Test database connection
	err = db.PingContext(ctx)
	require.NoError(suite.T(), err)

	suite.db = db

	// Create Prisma client
	suite.prisma, err = client.NewPrismaClientFromDB(suite.config.Provider, db)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), suite.prisma)

	// Parse schema
	schemaPath := filepath.Join("..", suite.config.SchemaPath)
	schemaContent, err := os.ReadFile(schemaPath)
	require.NoError(suite.T(), err)

	suite.schema, _ = parsing.ParseSchema(string(schemaContent))

	// Initialize introspector
	suite.introspector, err = introspect.NewIntrospector(suite.db, suite.config.Provider)
	require.NoError(suite.T(), err)

	// Initialize compiler
	suite.compiler = compiler.NewCompiler(suite.config.Provider)

	suite.T().Logf("E2E test suite setup completed for provider: %s", suite.config.Provider)
}

// SetupTest runs before each test to ensure clean state
func (suite *TestSuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// For SQLite, clean up any existing test data
	if suite.config.Provider == "sqlite" {
		// Delete data from common test tables
		tables := []string{"posts", "users", "departments", "employees", "projects", "employee_projects", "concurrent_test", "test_introspection"}
		for _, table := range tables {
			_, err := suite.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table))
			if err != nil {
				// Table might not exist, which is fine
				suite.T().Logf("Could not delete from %s (may not exist): %v", table, err)
			}
		}
	}
}

// TearDownSuite runs once per test suite
func (suite *TestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// getDriverName maps Prisma provider names to Go database driver names
func getDriverName(provider string) string {
	switch provider {
	case "postgresql", "postgres":
		return "postgres"
	case "mysql":
		return "mysql"
	case "sqlite":
		return "sqlite3"
	default:
		return ""
	}
}

// convertPlaceholders converts SQL placeholders based on provider
// PostgreSQL uses $1, $2, $3... while MySQL/SQLite use ?
func (suite *TestSuite) convertPlaceholders(query string) string {
	if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
		// Convert ? to $1, $2, $3...
		parts := strings.Split(query, "?")
		if len(parts) == 1 {
			return query
		}
		result := parts[0]
		for i := 1; i < len(parts); i++ {
			result += fmt.Sprintf("$%d%s", i, parts[i])
		}
		return result
	}
	// MySQL and SQLite use ? as-is
	return query
}


// TestE2ESuite runs E2E test suite for all database providers
func TestE2ESuite(t *testing.T) {
	for _, config := range getTestConfigs() {
		envVar := strings.ToUpper(config.Provider) + "_TEST_URL"
		if os.Getenv(envVar) == "" {
			t.Logf("Skipping %s tests: %s not provided", config.Provider, envVar)
			continue
		}

		t.Run(fmt.Sprintf("E2E_%s", config.Provider), func(t *testing.T) {
			testSuite := &TestSuite{
				config: &config,
			}
			suite.Run(t, testSuite)
		})
	}
}
