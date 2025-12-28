// Package validation provides test setup for validation tests
package validation

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestMain sets up and tears down the test database
func TestMain(m *testing.M) {
	// Check if validation tests should run
	if os.Getenv("RUN_VALIDATION_TESTS") != "true" {
		fmt.Println("Skipping validation tests. Set RUN_VALIDATION_TESTS=true to run.")
		os.Exit(0)
	}

	dbURL := GetDBURL()

	// Wait for database to be ready
	ctx := context.Background()
	if err := WaitForDB(ctx, dbURL, 30*time.Second); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Setup validation schema
	if err := SetupValidationSchema(dbURL); err != nil {
		fmt.Printf("Failed to setup schema: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}
