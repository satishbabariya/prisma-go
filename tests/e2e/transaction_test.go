package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
)

// TestTransactions tests transaction functionality
func (suite *TestSuite) TestTransactions() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test transaction commit
	suite.testTransactionCommit(ctx)

	// Test transaction rollback
	suite.testTransactionRollback(ctx)

	// Test nested transactions (savepoints)
	suite.testNestedTransactions(ctx)

	// Test transaction isolation levels
	suite.testTransactionIsolation(ctx)

	suite.T().Logf("Transaction tests passed for provider: %s", suite.config.Provider)
}

// testTransactionCommit tests successful transaction commit
func (suite *TestSuite) testTransactionCommit(ctx context.Context) {
	// Start transaction
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)

	// Insert data within transaction
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"txcommit@example.com", "Transaction User", 30)
	require.NoError(suite.T(), err)

	// Verify data exists within transaction
	var count int
	err = tx.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"), "txcommit@example.com").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)

	// Commit transaction
	err = tx.Commit()
	require.NoError(suite.T(), err)

	// Verify data still exists after commit
	err = suite.db.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"), "txcommit@example.com").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)
}

// testTransactionRollback tests transaction rollback
func (suite *TestSuite) testTransactionRollback(ctx context.Context) {
	// Start transaction
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)

	// Insert data within transaction
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"txrollback@example.com", "Rollback User", 25)
	require.NoError(suite.T(), err)

	// Verify data exists within transaction
	var count int
	err = tx.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"), "txrollback@example.com").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)

	// Rollback transaction
	err = tx.Rollback()
	require.NoError(suite.T(), err)

	// Verify data does not exist after rollback
	err = suite.db.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"), "txrollback@example.com").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 0, count)
}

// testNestedTransactions tests savepoint functionality
func (suite *TestSuite) testNestedTransactions(ctx context.Context) {
	// Start main transaction
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)

	// Insert first record
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"savepoint1@example.com", "Savepoint User 1", 35)
	require.NoError(suite.T(), err)

	// Create savepoint (database-specific syntax)
	var savepointSQL string
	switch suite.config.Provider {
	case "postgresql":
		savepointSQL = "SAVEPOINT test_savepoint"
	case "mysql":
		savepointSQL = "SAVEPOINT test_savepoint"
	case "sqlite":
		savepointSQL = "SAVEPOINT test_savepoint"
	}

	_, err = tx.ExecContext(ctx, savepointSQL)
	require.NoError(suite.T(), err)

	// Insert second record
	_, err = tx.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		"savepoint2@example.com", "Savepoint User 2", 40)
	require.NoError(suite.T(), err)

	// Verify both records exist
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email LIKE 'savepoint%'").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, count)

	// Rollback to savepoint
	var rollbackSQL string
	switch suite.config.Provider {
	case "postgresql":
		rollbackSQL = "ROLLBACK TO SAVEPOINT test_savepoint"
	case "mysql":
		rollbackSQL = "ROLLBACK TO SAVEPOINT test_savepoint"
	case "sqlite":
		rollbackSQL = "ROLLBACK TO SAVEPOINT test_savepoint"
	}

	_, err = tx.ExecContext(ctx, rollbackSQL)
	require.NoError(suite.T(), err)

	// Verify only first record exists
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email LIKE 'savepoint%'").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)

	// Commit main transaction
	err = tx.Commit()
	require.NoError(suite.T(), err)

	// Verify first record still exists
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = 'savepoint1@example.com'").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)
}

// testTransactionIsolation tests different isolation levels
func (suite *TestSuite) testTransactionIsolation(ctx context.Context) {
	// Test with different isolation levels based on database support
	isolationLevels := []sql.IsolationLevel{
		sql.LevelReadCommitted,
		sql.LevelRepeatableRead,
		sql.LevelSerializable,
	}

	for _, level := range isolationLevels {
		// Skip unsupported isolation levels for certain databases
		if suite.config.Provider == "sqlite" && level == sql.LevelRepeatableRead {
			continue // SQLite doesn't support REPEATABLE READ
		}

		// Start transaction with specific isolation level
		tx, err := suite.db.BeginTx(ctx, &sql.TxOptions{
			Isolation: level,
			ReadOnly:  false,
		})

		// Some databases might not support all isolation levels
		if err != nil {
			suite.T().Logf("Skipping isolation level %v for provider %s: %v", level, suite.config.Provider, err)
			continue
		}

		// Insert test data
		_, err = tx.ExecContext(ctx,
			suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
			"isolation@example.com", "Isolation Test", 45)
		require.NoError(suite.T(), err)

		// Test read within transaction
		var name string
		err = tx.QueryRowContext(ctx, suite.convertPlaceholders("SELECT name FROM users WHERE email = ?"), "isolation@example.com").Scan(&name)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), "Isolation Test", name)

		// Commit transaction
		err = tx.Commit()
		require.NoError(suite.T(), err)

		// Clean up for next test
		_, err = suite.db.ExecContext(ctx, suite.convertPlaceholders("DELETE FROM users WHERE email = ?"), "isolation@example.com")
		require.NoError(suite.T(), err)
	}
}

// TestConcurrentTransactions tests concurrent transaction behavior
func (suite *TestSuite) TestConcurrentTransactions() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test concurrent writes
	suite.testConcurrentWrites(ctx)

	// Test concurrent reads
	suite.testConcurrentReads(ctx)

	suite.T().Logf("Concurrent transaction tests passed for provider: %s", suite.config.Provider)
}

// testConcurrentWrites tests concurrent write operations
func (suite *TestSuite) testConcurrentWrites(ctx context.Context) {
	const numGoroutines = 5
	const recordsPerGoroutine = 3

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Start multiple goroutines with concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			// Each goroutine gets its own transaction
			tx, err := suite.db.BeginTx(ctx, nil)
			if err != nil {
				errors <- err
				return
			}
			defer tx.Rollback()

			// Insert multiple records
			for j := 0; j < recordsPerGoroutine; j++ {
				email := fmt.Sprintf("concurrent%d_%d@example.com", goroutineID, j)
				name := fmt.Sprintf("Concurrent User %d-%d", goroutineID, j)

				_, err = tx.ExecContext(ctx,
					suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
					email, name, 20+goroutineID*10+j)
				if err != nil {
					errors <- err
					return
				}
			}

			// Commit transaction
			err = tx.Commit()
			if err != nil {
				errors <- err
				return
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Goroutine completed successfully
		case err := <-errors:
			suite.T().Errorf("Concurrent write error: %v", err)
		case <-ctx.Done():
			suite.T().Fatal("Concurrent writes timed out")
		}
	}

	// Verify all records were inserted
	var totalRecords int
	err := suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email LIKE 'concurrent%'").Scan(&totalRecords)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), numGoroutines*recordsPerGoroutine, totalRecords)
}

// testConcurrentReads tests concurrent read operations
func (suite *TestSuite) testConcurrentReads(ctx context.Context) {
	// Insert test data first
	testEmail := "concurrentread@example.com"
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
		testEmail, "Concurrent Read Test", 50)
	require.NoError(suite.T(), err)

	const numReaders = 10
	done := make(chan bool, numReaders)
	errors := make(chan error, numReaders)

	// Start multiple goroutines with concurrent reads
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer func() { done <- true }()

			// Each reader gets its own transaction
			tx, err := suite.db.BeginTx(ctx, &sql.TxOptions{
				ReadOnly: true,
			})
			if err != nil {
				errors <- err
				return
			}
			defer tx.Rollback()

			// Perform multiple reads
			for j := 0; j < 5; j++ {
				var name string
				var age int
				err = tx.QueryRowContext(ctx,
					suite.convertPlaceholders("SELECT name, age FROM users WHERE email = ?"), testEmail).
					Scan(&name, &age)
				if err != nil {
					errors <- err
					return
				}

				// Verify data consistency
				if name != "Concurrent Read Test" || age != 50 {
					errors <- fmt.Errorf("data inconsistency: got %s, %d", name, age)
					return
				}
			}
		}(i)
	}

	// Wait for all readers to complete
	for i := 0; i < numReaders; i++ {
		select {
		case <-done:
			// Reader completed successfully
		case err := <-errors:
			suite.T().Errorf("Concurrent read error: %v", err)
		case <-ctx.Done():
			suite.T().Fatal("Concurrent reads timed out")
		}
	}
}

// TestTransactionDeadlocks tests deadlock detection and handling
func (suite *TestSuite) TestTransactionDeadlocks() {
	if suite.config.Provider == "sqlite" {
		suite.T().Skip("SQLite has limited deadlock detection")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Insert initial data
	_, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?), (?, ?, ?)"),
		"deadlock1@example.com", "Deadlock User 1", 30,
		"deadlock2@example.com", "Deadlock User 2", 35)
	require.NoError(suite.T(), err)

	// Test potential deadlock scenario
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	// Two transactions that could potentially deadlock
	go func() {
		defer func() { done <- true }()

		tx1, err := suite.db.BeginTx(ctx, nil)
		if err != nil {
			errors <- err
			return
		}
		defer tx1.Rollback()

		// Update first record
		_, err = tx1.ExecContext(ctx,
			suite.convertPlaceholders("UPDATE users SET age = age + 1 WHERE email = ?"), "deadlock1@example.com")
		if err != nil {
			errors <- err
			return
		}

		// Small delay to increase deadlock probability
		time.Sleep(10 * time.Millisecond)

		// Update second record
		_, err = tx1.ExecContext(ctx,
			suite.convertPlaceholders("UPDATE users SET age = age + 1 WHERE email = ?"), "deadlock2@example.com")
		if err != nil {
			errors <- err
			return
		}

		err = tx1.Commit()
		if err != nil {
			errors <- err
			return
		}
	}()

	go func() {
		defer func() { done <- true }()

		tx2, err := suite.db.BeginTx(ctx, nil)
		if err != nil {
			errors <- err
			return
		}
		defer tx2.Rollback()

		// Update second record first (reverse order)
		_, err = tx2.ExecContext(ctx,
			suite.convertPlaceholders("UPDATE users SET age = age + 2 WHERE email = ?"), "deadlock2@example.com")
		if err != nil {
			errors <- err
			return
		}

		// Small delay to increase deadlock probability
		time.Sleep(10 * time.Millisecond)

		// Update first record
		_, err = tx2.ExecContext(ctx,
			suite.convertPlaceholders("UPDATE users SET age = age + 2 WHERE email = ?"), "deadlock1@example.com")
		if err != nil {
			errors <- err
			return
		}

		err = tx2.Commit()
		if err != nil {
			errors <- err
			return
		}
	}()

	// Wait for both transactions
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// Transaction completed
		case err := <-errors:
			// Deadlock or other error is acceptable in this test
			suite.T().Logf("Transaction error (expected): %v", err)
		case <-ctx.Done():
			suite.T().Fatal("Deadlock test timed out")
		}
	}
}
