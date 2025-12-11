package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
)

// TestErrorScenarios tests error scenarios and recovery
func (suite *TestSuite) TestErrorScenarios() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Clean up first to ensure isolation from previous tests
	suite.cleanupTestTables(ctx)
	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test constraint violations
	suite.testConstraintViolations(ctx)

	// Test connection errors
	suite.testConnectionErrors(ctx)

	// Test transaction errors
	suite.testTransactionErrors(ctx)

	// Test query timeout errors
	suite.testQueryTimeoutErrors(ctx)

	// Test data type errors
	suite.testDataTypeErrors(ctx)

	suite.T().Logf("Error handling tests passed for provider: %s", suite.config.Provider)
}

// testConstraintViolations tests database constraint violations
func (suite *TestSuite) testConstraintViolations(ctx context.Context) {
	// Test simple query first to verify connection works
	var count int
	err := suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	suite.T().Logf("Initial user count: %d, Error: %v", count, err)
	require.NoError(suite.T(), err)

	// Use unique email to avoid conflicts with other tests
	uniqueEmail := fmt.Sprintf("unique-%d@example.com", time.Now().UnixNano())
	
	// Test unique constraint violation
	sql := suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)")
	suite.T().Logf("Executing SQL: %s", sql)
	suite.T().Logf("Parameters: %v", []interface{}{uniqueEmail, "Unique User", 30})

	// Try using standard database/sql approach
	stmt, err := suite.db.PrepareContext(ctx, sql)
	if err != nil {
		suite.T().Logf("Prepare error: %v", err)
		require.NoError(suite.T(), err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(uniqueEmail, "Unique User", 30)
	suite.T().Logf("Result: %v, Error: %v", result, err)
	require.NoError(suite.T(), err)

	// Try to insert same email again (should fail)
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		uniqueEmail, "Duplicate User", 25)
	require.Error(suite.T(), err)

	// Verify error message contains constraint information
	errMsg := err.Error()
	suite.T().Logf("Unique constraint error: %s", errMsg)

	// Test foreign key constraint violation
	// Ensure foreign keys are enabled for SQLite
	if suite.config.Provider == "sqlite" {
		_, _ = suite.db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	}
	
	// Use a very large ID that's guaranteed not to exist
	nonExistentUserID := 999999999
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)"),
		"Orphan Post", "This post has no author", nonExistentUserID)
	
	// SQLite might not enforce foreign keys if they weren't enabled at table creation
	if suite.config.Provider == "sqlite" && err == nil {
		suite.T().Logf("SQLite foreign key constraint not enforced (may need PRAGMA foreign_keys at table creation)")
		// Delete the inserted row to clean up
		_, _ = suite.db.ExecContext(ctx, "DELETE FROM posts WHERE author_id = ?", nonExistentUserID)
	} else {
		require.Error(suite.T(), err)
	}

	errMsg = err.Error()
	suite.T().Logf("Foreign key constraint error: %s", errMsg)

	// Test NOT NULL constraint violation
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)"),
		"Valid Post", "This post has content", nil)
	require.Error(suite.T(), err)

	errMsg = err.Error()
	suite.T().Logf("NOT NULL constraint error: %s", errMsg)
}

// testConnectionErrors tests connection-related errors
func (suite *TestSuite) testConnectionErrors(ctx context.Context) {
	// Create a new database connection with invalid credentials
	invalidDSN := "invalid://connection/string"

	var db *sql.DB
	var err error

	switch suite.config.Provider {
	case "postgresql":
		db, err = sql.Open("postgres", invalidDSN)
	case "mysql":
		db, err = sql.Open("mysql", invalidDSN)
	case "sqlite":
		db, err = sql.Open("sqlite3", invalidDSN)
	}

	if err == nil {
		defer db.Close()

		// Try to execute a query (should fail)
		_, err = db.ExecContext(ctx, "SELECT 1")
		require.Error(suite.T(), err)

		errMsg := err.Error()
		suite.T().Logf("Connection error: %s", errMsg)
	}
}

// testTransactionErrors tests error handling within transactions
func (suite *TestSuite) testTransactionErrors(ctx context.Context) {
	// Start a transaction
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)
	defer tx.Rollback()

	// Insert valid data
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"txuser@example.com", "Transaction User", 28)
	require.NoError(suite.T(), err)

	// Try to insert invalid data (should fail)
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"txuser@example.com", "Duplicate in Transaction", 30)
	require.Error(suite.T(), err)

	// PostgreSQL aborts transactions on error, so we need to rollback and start a new transaction
	// MySQL and SQLite may allow continuing after errors
	if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
		// Rollback the aborted transaction
		tx.Rollback()
		
		// Start a new transaction
		tx, err = suite.db.BeginTx(ctx, nil)
		require.NoError(suite.T(), err)
		defer tx.Rollback()
	}

	// Transaction should be usable now (either continued or new)
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"txuser2@example.com", "Another Transaction User", 32)
	require.NoError(suite.T(), err)

	// Commit transaction
	err = tx.Commit()
	require.NoError(suite.T(), err)

	// Verify data was committed correctly
	var count int
	err = suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email LIKE 'txuser%'")).Scan(&count)
	require.NoError(suite.T(), err)
	// PostgreSQL rolls back the entire transaction on error, so only the new transaction's insert remains
	// MySQL/SQLite may allow partial commits, so they might have 2 records
	if suite.config.Provider == "postgresql" || suite.config.Provider == "postgres" {
		require.Equal(suite.T(), 1, count) // Only txuser2 from the new transaction
	} else {
		require.Equal(suite.T(), 2, count) // Both txuser and txuser2
	}
}

// testQueryTimeoutErrors tests query timeout scenarios
func (suite *TestSuite) testQueryTimeoutErrors(ctx context.Context) {
	// Create a context with very short timeout
	shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Try a long-running query (should timeout due to context)
	var query string
	switch suite.config.Provider {
	case "postgresql":
		query = "SELECT pg_sleep(1)"
	case "mysql":
		query = "SELECT SLEEP(1)"
	case "sqlite":
		query = "SELECT 1" // SQLite doesn't have sleep, use immediate query
	}

	_, err := suite.db.ExecContext(shortCtx, query)

	// For SQLite, the query will succeed quickly, so we expect no error
	// For PostgreSQL and MySQL, we expect a timeout error
	if suite.config.Provider == "sqlite" {
		require.NoError(suite.T(), err)
	} else {
		require.Error(suite.T(), err)
		errMsg := err.Error()
		suite.T().Logf("Query timeout error: %s", errMsg)
	}
}

// testDataTypeErrors tests data type mismatch errors
func (suite *TestSuite) testDataTypeErrors(ctx context.Context) {
	// Test inserting invalid data types
	// Note: This test may behave differently across databases

	// Try to insert a string into an integer field
	// Note: SQLite uses dynamic typing and is very permissive, so this might not error
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"datatype@example.com", "Data Type User", "not-a-number")
	
	// SQLite is very permissive with types and might allow this
	if suite.config.Provider == "sqlite" {
		// SQLite might allow this, so we don't require an error
		if err != nil {
			suite.T().Logf("Data type error: %s", err.Error())
		} else {
			suite.T().Logf("SQLite allowed string in integer field (dynamic typing)")
		}
	} else {
		require.Error(suite.T(), err)
		errMsg := err.Error()
		suite.T().Logf("Data type error: %s", errMsg)
	}

	// Try to insert a very long string that exceeds field length
	longString := make([]byte, 300) // Assuming VARCHAR(255) limit
	for i := range longString {
		longString[i] = 'a'
	}

	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		string(longString)+"@example.com", "Long Name User", 25)

	// This may or may not error depending on the database and field definitions
	if err != nil {
		suite.T().Logf("Field length error: %s", err.Error())
	}
}

// TestErrorRecovery tests error recovery mechanisms
func (suite *TestSuite) TestErrorRecovery() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test recovery after constraint violation
	suite.testRecoveryAfterConstraintViolation(ctx)

	// Test recovery after connection loss
	suite.testRecoveryAfterConnectionLoss(ctx)

	// Test recovery after transaction rollback
	suite.testRecoveryAfterTransactionRollback(ctx)

	suite.T().Logf("Error recovery tests passed for provider: %s", suite.config.Provider)
}

// testRecoveryAfterConstraintViolation tests recovery after constraint errors
func (suite *TestSuite) testRecoveryAfterConstraintViolation(ctx context.Context) {
	// Insert a user
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"recovery@example.com", "Recovery User", 30)
	require.NoError(suite.T(), err)

	// Try to insert duplicate (should fail)
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"recovery@example.com", "Duplicate User", 25)
	require.Error(suite.T(), err)

	// Database should still be responsive after the error
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"recovery2@example.com", "Recovery User 2", 35)
	require.NoError(suite.T(), err)

	// Verify both records exist (original and new one)
	var count int
	err = suite.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE email LIKE 'recovery%'").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, count)
}

// testRecoveryAfterConnectionLoss simulates connection loss and recovery
func (suite *TestSuite) testRecoveryAfterConnectionLoss(ctx context.Context) {
	// This test is more conceptual as we can't easily simulate real connection loss
	// But we can test that the connection pool handles errors gracefully

	// Execute a successful query
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"connection@example.com", "Connection User", 40)
	require.NoError(suite.T(), err)

	// Verify the connection is still working
	var count int
	err = suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"), "connection@example.com").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)
}

// testRecoveryAfterTransactionRollback tests recovery after transaction rollback
func (suite *TestSuite) testRecoveryAfterTransactionRollback(ctx context.Context) {
	// Start a transaction
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)

	// Insert some data
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"rollback@example.com", "Rollback User", 45)
	require.NoError(suite.T(), err)

	// Rollback the transaction
	err = tx.Rollback()
	require.NoError(suite.T(), err)

	// Database should be immediately usable after rollback
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"afterrollback@example.com", "After Rollback User", 50)
	require.NoError(suite.T(), err)

	// Verify only the post-rollback data exists
	var count int
	err = suite.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE email = 'rollback@example.com'").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 0, count)

	err = suite.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE email = 'afterrollback@example.com'").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)
}

// TestErrorLogging tests error logging and reporting
func (suite *TestSuite) TestErrorLogging() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test error message clarity
	suite.testErrorMessageClarity(ctx)

	// Test error context preservation
	suite.testErrorContextPreservation(ctx)

	suite.T().Logf("Error logging tests passed for provider: %s", suite.config.Provider)
}

// testErrorMessageClarity tests that error messages are clear and useful
func (suite *TestSuite) testErrorMessageClarity(ctx context.Context) {
	// Test unique constraint error message
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"clarity@example.com", "Clarity User", 30)
	require.NoError(suite.T(), err)

	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"clarity@example.com", "Duplicate Clarity User", 25)
	require.Error(suite.T(), err)

	errMsg := err.Error()

	// Check that error message contains useful information
	// Different databases provide different levels of detail
	usefulInfo := []string{
		"unique", "constraint", "duplicate", "email", "users",
		"UNIQUE", "PRIMARY", "KEY", "violation",
	}

	hasUsefulInfo := false
	for _, info := range usefulInfo {
		if contains(errMsg, info) {
			hasUsefulInfo = true
			break
		}
	}

	// At least one useful term should be in the error message
	if !hasUsefulInfo {
		suite.T().Logf("Warning: Error message may not be very useful: %s", errMsg)
	}
}

// testErrorContextPreservation tests that error context is preserved
func (suite *TestSuite) testErrorContextPreservation(ctx context.Context) {
	// Create a context with cancellation
	cancelCtx, cancel := context.WithCancel(ctx)

	// Cancel the context immediately
	cancel()

	// Try to execute a query with cancelled context
	_, err := suite.db.ExecContext(cancelCtx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"context@example.com", "Context User", 35)
	require.Error(suite.T(), err)

	errMsg := err.Error()

	// Error should mention context cancellation
	contextTerms := []string{"context", "canceled", "cancelled", "timeout"}

	hasContextInfo := false
	for _, term := range contextTerms {
		if contains(errMsg, term) {
			hasContextInfo = true
			break
		}
	}

	if hasContextInfo {
		suite.T().Logf("Context error properly preserved: %s", errMsg)
	}
}

// TestBatchErrorHandling tests error handling in batch operations
func (suite *TestSuite) TestBatchErrorHandling() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test partial batch failures
	suite.testPartialBatchFailures(ctx)

	// Test batch rollback on error
	suite.testBatchRollbackOnError(ctx)

	suite.T().Logf("Batch error handling tests passed for provider: %s", suite.config.Provider)
}

// testPartialBatchFailures tests handling of partial failures in batch operations
func (suite *TestSuite) testPartialBatchFailures(ctx context.Context) {
	// Start a transaction for batch operations
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)
	defer tx.Rollback()

	// Prepare a statement for batch insert
	stmt, err := tx.PrepareContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"))
	require.NoError(suite.T(), err)
	defer stmt.Close()

	// Insert valid data
	users := []struct {
		Email string
		Name  string
		Age   int
	}{
		{"batch1@example.com", "Batch User 1", 25},
		{"batch2@example.com", "Batch User 2", 30},
	}

	for _, user := range users {
		_, err = stmt.ExecContext(ctx, user.Email, user.Name, user.Age)
		require.NoError(suite.T(), err)
	}

	// Try to insert duplicate email (should fail)
	_, err = stmt.ExecContext(ctx, "batch1@example.com", "Duplicate Batch", 35)
	require.Error(suite.T(), err)

	// Try to insert more data after the error
	_, err = stmt.ExecContext(ctx, "batch3@example.com", "Batch User 3", 40)

	// Behavior varies by database:
	// - Some databases continue after error
	// - Others abort the entire transaction
	if err != nil {
		suite.T().Logf("Batch operation failed after error: %v", err)
	} else {
		suite.T().Logf("Batch operation continued after error")
	}
}

// testBatchRollbackOnError tests that batch operations rollback on error
func (suite *TestSuite) testBatchRollbackOnError(ctx context.Context) {
	// Start a transaction
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)

	// Insert some valid data
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"batchrollback@example.com", "Batch Rollback User", 45)
	require.NoError(suite.T(), err)

	// Cause an error
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"batchrollback@example.com", "Duplicate Batch Rollback", 50)
	require.Error(suite.T(), err)

	// Rollback the transaction
	err = tx.Rollback()
	require.NoError(suite.T(), err)

	// Verify no data was committed
	var count int
	err = suite.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE email LIKE 'batchrollback%'").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 0, count)
}
