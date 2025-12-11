package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	queryAst "github.com/satishbabariya/prisma-go/query/ast"
	"github.com/stretchr/testify/require"
)

// TestCRUDOperations tests basic CRUD operations
func (suite *TestSuite) TestCRUDOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test Create operation
	suite.testCreateOperation(ctx)

	// Test Read operation
	suite.testReadOperation(ctx)

	// Test Update operation
	suite.testUpdateOperation(ctx)

	// Test Delete operation
	suite.testDeleteOperation(ctx)

	suite.T().Logf("CRUD operations test passed for provider: %s", suite.config.Provider)
}

// createTestTables creates test tables for CRUD operations
func (suite *TestSuite) createTestTables(ctx context.Context) {
	var createSQL string

	switch suite.config.Provider {
	case "postgresql":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	case "mysql":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			author_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	case "sqlite":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			age INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			author_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	}

	// MySQL doesn't support multiple statements in one ExecContext call
	if suite.config.Provider == "mysql" {
		// Split by semicolon and execute each statement separately
		statements := strings.Split(createSQL, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				_, err := suite.db.ExecContext(ctx, stmt)
				require.NoError(suite.T(), err)
			}
		}
	} else {
		_, err := suite.db.ExecContext(ctx, createSQL)
		require.NoError(suite.T(), err)
	}
}

// cleanupTestTables removes test tables
func (suite *TestSuite) cleanupTestTables(ctx context.Context) {
	if suite.config.Provider == "sqlite" {
		// SQLite doesn't support dropping multiple tables in one statement
		// Also, delete data first to avoid foreign key issues
		_, _ = suite.db.ExecContext(ctx, "DELETE FROM posts")
		_, _ = suite.db.ExecContext(ctx, "DELETE FROM users")
		_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS posts")
		if err != nil {
			suite.T().Logf("Error dropping posts table: %v", err)
		}
		_, err = suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS users")
		if err != nil {
			suite.T().Logf("Error dropping users table: %v", err)
		}
	} else {
		_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS posts, users")
		if err != nil {
			suite.T().Logf("Error dropping tables: %v", err)
		}
	}
}

// testCreateOperation tests the Create operation
func (suite *TestSuite) testCreateOperation(ctx context.Context) {
	// Insert a user
	userEmail := "test@example.com"
	userName := "Test User"
	userAge := 25

	var userID int
	if suite.config.Provider == "sqlite" || suite.config.Provider == "mysql" {
		// SQLite and MySQL don't support RETURNING clause, use last_insert_rowid()/LastInsertId()
		result, err := suite.db.ExecContext(ctx,
			suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"),
			userEmail, userName, userAge)
		require.NoError(suite.T(), err)
		lastID, err := result.LastInsertId()
		require.NoError(suite.T(), err)
		userID = int(lastID)
	} else {
		err := suite.db.QueryRowContext(ctx,
			suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?) RETURNING id"),
			userEmail, userName, userAge).Scan(&userID)
		require.NoError(suite.T(), err)
	}
	require.Greater(suite.T(), userID, 0)

	// Insert a post for the user
	postTitle := "Test Post"
	postContent := "This is a test post content"

	var postID int
	if suite.config.Provider == "sqlite" || suite.config.Provider == "mysql" {
		// SQLite and MySQL don't support RETURNING clause, use last_insert_rowid()/LastInsertId()
		result, err := suite.db.ExecContext(ctx,
			suite.convertPlaceholders("INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)"),
			postTitle, postContent, userID)
		require.NoError(suite.T(), err)
		lastID, err := result.LastInsertId()
		require.NoError(suite.T(), err)
		postID = int(lastID)
	} else {
		err := suite.db.QueryRowContext(ctx,
			suite.convertPlaceholders("INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?) RETURNING id"),
			postTitle, postContent, userID).Scan(&postID)
		require.NoError(suite.T(), err)
	}
	require.Greater(suite.T(), postID, 0)

	// Verify the records were created
	var count int
	var err error
	err = suite.db.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"), userEmail).Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)

	err = suite.db.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM posts WHERE title = ?"), postTitle).Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)
}

// testReadOperation tests the Read operation
func (suite *TestSuite) testReadOperation(ctx context.Context) {
	// Test simple SELECT
	var email, name string
	var age int
	err := suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT email, name, age FROM users WHERE email = ?"), "test@example.com").
		Scan(&email, &name, &age)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "test@example.com", email)
	require.Equal(suite.T(), "Test User", name)
	require.Equal(suite.T(), 25, age)

	// Test SELECT with JOIN
	rows, err := suite.db.QueryContext(ctx, suite.convertPlaceholders(`
		SELECT u.email, u.name, p.title, p.content 
		FROM users u 
		JOIN posts p ON u.id = p.author_id 
		WHERE u.email = ?`), "test@example.com")
	require.NoError(suite.T(), err)
	defer rows.Close()

	var posts []struct {
		UserEmail string
		UserName  string
		Title     string
		Content   string
	}

	for rows.Next() {
		var userEmail, userName, title, content string
		err = rows.Scan(&userEmail, &userName, &title, &content)
		require.NoError(suite.T(), err)

		posts = append(posts, struct {
			UserEmail string
			UserName  string
			Title     string
			Content   string
		}{
			UserEmail: userEmail,
			UserName:  userName,
			Title:     title,
			Content:   content,
		})
	}

	require.NotEmpty(suite.T(), posts)
	require.Equal(suite.T(), "Test Post", posts[0].Title)
}

// testUpdateOperation tests the Update operation
func (suite *TestSuite) testUpdateOperation(ctx context.Context) {
	// Update user name
	newName := "Updated User"
	result, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("UPDATE users SET name = ? WHERE email = ?"), newName, "test@example.com")
	require.NoError(suite.T(), err)

	rowsAffected, err := result.RowsAffected()
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), int64(1), rowsAffected)

	// Verify the update
	var name string
	err = suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT name FROM users WHERE email = ?"), "test@example.com").Scan(&name)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), newName, name)

	// Update post to published
	result, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("UPDATE posts SET published = TRUE WHERE title = ?"), "Test Post")
	require.NoError(suite.T(), err)

	rowsAffected, err = result.RowsAffected()
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), int64(1), rowsAffected)

	// Verify the update
	var published bool
	err = suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("SELECT published FROM posts WHERE title = ?"), "Test Post").Scan(&published)
	require.NoError(suite.T(), err)
	require.True(suite.T(), published)
}

// testDeleteOperation tests the Delete operation
func (suite *TestSuite) testDeleteOperation(ctx context.Context) {
	// Delete the post first (due to foreign key constraint)
	result, err := suite.db.ExecContext(ctx,
		suite.convertPlaceholders("DELETE FROM posts WHERE title = ?"), "Test Post")
	require.NoError(suite.T(), err)

	rowsAffected, err := result.RowsAffected()
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), int64(1), rowsAffected)

	// Delete the user
	result, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("DELETE FROM users WHERE email = ?"), "test@example.com")
	require.NoError(suite.T(), err)

	rowsAffected, err = result.RowsAffected()
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), int64(1), rowsAffected)

	// Verify the deletions
	var count int
	err = suite.db.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM users WHERE email = ?"), "test@example.com").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 0, count)

	err = suite.db.QueryRowContext(ctx, suite.convertPlaceholders("SELECT COUNT(*) FROM posts WHERE title = ?"), "Test Post").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 0, count)
}

// TestQueryCompilation tests query compilation functionality
func (suite *TestSuite) TestQueryCompilation() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a simple FindMany query
	findManyQuery := &queryAst.FindManyQuery{
		Model: "users",
		Where: &queryAst.WhereClause{
			Conditions: []queryAst.Condition{
				{
					Field:    "email",
					Operator: "=",
					Value:    "test@example.com",
				},
			},
		},
		Select: &queryAst.SelectClause{
			Fields: []string{"id", "email", "name"},
		},
	}

	// Compile the query
	sql, args, err := suite.compiler.Compile(findManyQuery)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), sql)
	require.NotNil(suite.T(), args)

	suite.T().Logf("Compiled SQL: %s, Args: %v", sql, args)

	// Test that the compiled SQL can be executed (if tables exist)
	// Clean up first to ensure isolation from previous tests
	suite.cleanupTestTables(ctx)
	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Insert test data first
	// Use a unique email to avoid conflicts with other tests
	testEmail := fmt.Sprintf("querycomp-%d@example.com", time.Now().UnixNano())
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name) VALUES (?, ?)"), testEmail, "Test User")
	require.NoError(suite.T(), err)
	
	// Update the query args to use the new email
	args = []interface{}{testEmail}

	// Execute the compiled query
	rows, err := suite.db.QueryContext(ctx, sql, args...)
	require.NoError(suite.T(), err)
	defer rows.Close()

	// Verify results
	var count int
	for rows.Next() {
		count++
		var id int
		var email, name string
		err = rows.Scan(&id, &email, &name)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), testEmail, email)
		require.Equal(suite.T(), "Test User", name)
	}

	require.Greater(suite.T(), count, 0)
}

// TestBatchOperations tests batch insert, update, and delete operations
func (suite *TestSuite) TestBatchOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite.createTestTables(ctx)
	defer suite.cleanupTestTables(ctx)

	// Test batch insert
	users := []struct {
		Email string
		Name  string
		Age   int
	}{
		{"user1@example.com", "User One", 25},
		{"user2@example.com", "User Two", 30},
		{"user3@example.com", "User Three", 35},
	}

	// Start transaction for batch operations
	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)
	defer tx.Rollback()

	// Insert multiple users
	stmt, err := tx.PrepareContext(ctx, suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"))
	require.NoError(suite.T(), err)
	defer stmt.Close()

	for _, user := range users {
		_, err = stmt.ExecContext(ctx, user.Email, user.Name, user.Age)
		require.NoError(suite.T(), err)
	}

	// Verify batch insert
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), len(users), count)

	// Test batch update
	_, err = tx.ExecContext(ctx, "UPDATE users SET age = age + 1 WHERE age >= 30")
	require.NoError(suite.T(), err)

	// Verify batch update
	var updatedCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE age >= 31").Scan(&updatedCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, updatedCount) // User Two and User Three

	// Test batch delete
	_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE age <= 26")
	require.NoError(suite.T(), err)

	// Verify batch delete
	var remainingCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&remainingCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, remainingCount) // User Two and User Three remain

	// Commit the transaction
	err = tx.Commit()
	require.NoError(suite.T(), err)

	suite.T().Logf("Batch operations test passed for provider: %s", suite.config.Provider)
}
