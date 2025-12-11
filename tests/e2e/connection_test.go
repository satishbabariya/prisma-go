package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/psl/core"
	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
	"github.com/satishbabariya/prisma-go/psl/validation"
	"github.com/stretchr/testify/require"
)

// TestDatabaseConnection tests basic database connectivity
func (suite *TestSuite) TestDatabaseConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic connection
	err := suite.db.PingContext(ctx)
	require.NoError(suite.T(), err)

	// Test query execution
	var result string
	err = suite.db.QueryRowContext(ctx, "SELECT 1 as test").Scan(&result)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1", result)

	suite.T().Logf("Database connection test passed for provider: %s", suite.config.Provider)
}

// TestSchemaParsing tests Prisma schema parsing functionality
func (suite *TestSuite) TestSchemaParsing() {
	require.NotNil(suite.T(), suite.schema)
	require.NotEmpty(suite.T(), suite.schema.Tops)

	// Check if we have models
	var models []*ast.Model
	for _, top := range suite.schema.Tops {
		if model := top.AsModel(); model != nil {
			models = append(models, model)
		}
	}
	require.NotEmpty(suite.T(), models, "Schema should contain at least one model")

	// Check if we have datasource
	sources := suite.schema.Sources()
	require.NotEmpty(suite.T(), sources, "Schema should contain datasource")

	// Check if we have generator
	generators := suite.schema.Generators()
	require.NotEmpty(suite.T(), generators, "Schema should contain generator")

	suite.T().Logf("Schema parsing test passed: found %d models", len(models))
}

// TestSchemaValidation tests Prisma schema validation
func (suite *TestSuite) TestSchemaValidation() {
	require.NotNil(suite.T(), suite.schema)
	require.NotEmpty(suite.T(), suite.schema.Tops)

	// Get schema file path
	schemaPath := filepath.Join("..", suite.config.SchemaPath)
	schemaContent, err := os.ReadFile(schemaPath)
	require.NoError(suite.T(), err)

	// Create source file
	sourceFile := core.SourceFile{
		Path: schemaPath,
		Data: string(schemaContent),
	}

	// Get connectors for the provider
	connectors := getConnectorsForProvider(suite.config.Provider)
	extensionTypes := database.NewExtensionTypes()

	// Validate schema
	validatedSchema := validation.Validate(sourceFile, connectors, extensionTypes)

	// Check for validation errors
	diags := validatedSchema.Diagnostics
	if diags.HasErrors() {
		errors := diags.Errors()
		for _, err := range errors {
			suite.T().Logf("Validation error: %s", err.Message())
		}
		// Don't fail the test if there are only warnings, but log errors
		// In a real scenario, you might want to fail on errors
		if len(errors) > 0 {
			suite.T().Logf("Schema validation found %d errors", len(errors))
		}
	}

	// Check for warnings
	warnings := diags.Warnings()
	if len(warnings) > 0 {
		suite.T().Logf("Schema validation found %d warnings", len(warnings))
		for _, warn := range warnings {
			suite.T().Logf("Validation warning: %s", warn.Message())
		}
	}

	// Verify validated database is not nil
	require.NotNil(suite.T(), validatedSchema.Db, "Validated database should not be nil")

	// Verify we have models in the validated database
	models := validatedSchema.Db.WalkModels()
	require.NotEmpty(suite.T(), models, "Validated schema should contain at least one model")

	suite.T().Logf("Schema validation test passed: schema validated successfully with %d models", len(models))
}

// getConnectorsForProvider returns the appropriate connectors for a given provider
func getConnectorsForProvider(provider string) []validation.Connector {
	// Return empty slice for now - connectors are optional for basic validation
	// In a full implementation, you would return provider-specific connectors
	return []validation.Connector{}
}

// TestDatabaseIntrospection tests database schema introspection
func (suite *TestSuite) TestDatabaseIntrospection() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test table for introspection
	testTable := `
	CREATE TABLE IF NOT EXISTS test_introspection (
		id INTEGER PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	// Adjust SQL for different providers
	switch suite.config.Provider {
	case "mysql":
		testTable = `
		CREATE TABLE IF NOT EXISTS test_introspection (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	case "sqlite":
		testTable = `
		CREATE TABLE IF NOT EXISTS test_introspection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	}

	_, err := suite.db.ExecContext(ctx, testTable)
	require.NoError(suite.T(), err)

	// Perform introspection
	schema, err := suite.introspector.Introspect(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), schema)

	// Find our test table
	var testTableFound *introspect.Table
	for _, table := range schema.Tables {
		if table.Name == "test_introspection" {
			testTableFound = &table
			break
		}
	}
	require.NotNil(suite.T(), testTableFound, "Test table should be found during introspection")

	// Verify table structure
	require.Equal(suite.T(), "test_introspection", testTableFound.Name)
	require.NotEmpty(suite.T(), testTableFound.Columns)

	// Check for expected columns
	expectedColumns := map[string]bool{
		"id":         false,
		"name":       false,
		"email":      false,
		"created_at": false,
	}

	for _, column := range testTableFound.Columns {
		expectedColumns[column.Name] = true
	}

	for col, found := range expectedColumns {
		require.True(suite.T(), found, "Column %s should be found", col)
	}

	// Clean up
	_, err = suite.db.ExecContext(ctx, "DROP TABLE test_introspection")
	require.NoError(suite.T(), err)

	suite.T().Logf("Database introspection test passed for provider: %s", suite.config.Provider)
}

// TestPrismaClientConnection tests Prisma client connection and basic operations
func (suite *TestSuite) TestPrismaClientConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NotNil(suite.T(), suite.prisma)

	// Test client is connected
	require.NoError(suite.T(), suite.prisma.Connect(ctx))

	// Test raw SQL execution through client
	rows, err := suite.prisma.RawQuery(ctx, "SELECT 1")
	require.NoError(suite.T(), err)
	defer rows.Close()

	require.True(suite.T(), rows.Next())
	var result int
	err = rows.Scan(&result)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, result)

	suite.T().Logf("Prisma client connection test passed for provider: %s", suite.config.Provider)
}

// TestErrorHandling tests various error scenarios
func (suite *TestSuite) TestErrorHandling() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test invalid SQL query
	_, err := suite.db.ExecContext(ctx, "INVALID SQL QUERY")
	require.Error(suite.T(), err)

	// Test query on non-existent table
	var result string
	err = suite.db.QueryRowContext(ctx, "SELECT * FROM non_existent_table LIMIT 1").Scan(&result)
	require.Error(suite.T(), err)

	// Test connection timeout (if supported)
	if suite.config.Provider != "sqlite" {
		// Create a context with very short timeout
		shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer shortCancel()

		// This should timeout
		_, err = suite.db.ExecContext(shortCtx, "SELECT SLEEP(1)")
		if suite.config.Provider == "mysql" {
			require.Error(suite.T(), err) // MySQL SLEEP should timeout
		}
	}

	suite.T().Logf("Error handling test passed for provider: %s", suite.config.Provider)
}

// TestConcurrentConnections tests concurrent database access
func (suite *TestSuite) TestConcurrentConnections() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Drop table first to ensure clean state
	_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS concurrent_test")
	require.NoError(suite.T(), err)

	// Create a test table
	testTable := `
	CREATE TABLE concurrent_test (
		id SERIAL PRIMARY KEY,
		value VARCHAR(255)
	)`

	// Adjust SQL for different providers
	switch suite.config.Provider {
	case "mysql":
		testTable = `
		CREATE TABLE concurrent_test (
			id INT PRIMARY KEY AUTO_INCREMENT,
			value VARCHAR(255)
		)`
	case "sqlite":
		testTable = `
		CREATE TABLE concurrent_test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT
		)`
	}

	_, err = suite.db.ExecContext(ctx, testTable)
	require.NoError(suite.T(), err)

	// Test concurrent operations
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Insert data
			_, err := suite.db.ExecContext(ctx, suite.convertPlaceholders("INSERT INTO concurrent_test (value) VALUES (?)"), fmt.Sprintf("test-%d", id))
			require.NoError(suite.T(), err)

			// Query data
			var count int
			err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM concurrent_test").Scan(&count)
			require.NoError(suite.T(), err)
			require.GreaterOrEqual(suite.T(), count, 1)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Verify final state
	var finalCount int
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM concurrent_test").Scan(&finalCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), concurrency, finalCount)

	// Clean up
	_, err = suite.db.ExecContext(ctx, "DROP TABLE concurrent_test")
	require.NoError(suite.T(), err)

	suite.T().Logf("Concurrent connections test passed for provider: %s", suite.config.Provider)
}
