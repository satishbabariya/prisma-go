package e2e

import (
	"context"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDatabaseSeeding tests database seeding functionality with complex nested relationships
func (suite *TestSuite) TestDatabaseSeeding() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create test tables for seeding
	suite.createSeedingTables(ctx)
	defer suite.cleanupSeedingTables(ctx)

	// Test seeding with nested relationships
	suite.testNestedSeeding(ctx)

	// Test seeding with multiple records
	suite.testBulkSeeding(ctx)

	suite.T().Logf("Database seeding test passed for provider: %s", suite.config.Provider)
}

// createSeedingTables creates tables for seeding tests
func (suite *TestSuite) createSeedingTables(ctx context.Context) {
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
			view_count INTEGER DEFAULT 0,
			author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS comments (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
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
			view_count INT DEFAULT 0,
			author_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS comments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			content TEXT NOT NULL,
			post_id INT NOT NULL,
			author_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
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
			view_count INTEGER DEFAULT 0,
			author_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			post_id INTEGER NOT NULL,
			author_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	}

	_, err := suite.db.ExecContext(ctx, createSQL)
	require.NoError(suite.T(), err)
}

// cleanupSeedingTables removes seeding test tables
func (suite *TestSuite) cleanupSeedingTables(ctx context.Context) {
	_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS comments, posts, users")
	require.NoError(suite.T(), err)
}

// testNestedSeeding tests seeding with nested relationships
func (suite *TestSuite) testNestedSeeding(ctx context.Context) {
	// Insert a user with nested posts and comments
	var userID int
	err := suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?) RETURNING id"),
		"alice@prisma.io", "Alice", 27).Scan(&userID)
	require.NoError(suite.T(), err)

	// Insert posts for Alice
	var post1ID, post2ID int
	err = suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("INSERT INTO posts (title, content, published, view_count, author_id) VALUES (?, ?, ?, ?, ?) RETURNING id"),
		"Join the Prisma Slack", "https://slack.prisma.io", true, 42, userID).Scan(&post1ID)
	require.NoError(suite.T(), err)

	err = suite.db.QueryRowContext(ctx,
		suite.convertPlaceholders("INSERT INTO posts (title, content, published, view_count, author_id) VALUES (?, ?, ?, ?, ?) RETURNING id"),
		"Follow Prisma on Twitter", "https://www.twitter.com/prisma", true, 128, userID).Scan(&post2ID)
	require.NoError(suite.T(), err)

	// Insert comments on posts
	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO comments (content, post_id, author_id) VALUES (?, ?, ?)"),
		"Great resource!", post1ID, userID)
	require.NoError(suite.T(), err)

	_, err = suite.db.ExecContext(ctx,
		suite.convertPlaceholders("INSERT INTO comments (content, post_id, author_id) VALUES (?, ?, ?)"),
		"Thanks for sharing!", post2ID, userID)
	require.NoError(suite.T(), err)

	// Verify nested data was created correctly
	var userCount, postCount, commentCount int
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "alice@prisma.io").Scan(&userCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, userCount)

	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE author_id = ?", userID).Scan(&postCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, postCount)

	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM comments WHERE author_id = ?", userID).Scan(&commentCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 2, commentCount)
}

// testBulkSeeding tests bulk seeding operations
func (suite *TestSuite) testBulkSeeding(ctx context.Context) {
	// Insert multiple users in a transaction-like manner
	users := []struct {
		Email string
		Name  string
		Age   int
	}{
		{"nilu@prisma.io", "Nilu", 29},
		{"mahmoud@prisma.io", "Mahmoud", 30},
	}

	tx, err := suite.db.BeginTx(ctx, nil)
	require.NoError(suite.T(), err)
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, suite.convertPlaceholders("INSERT INTO users (email, name, age) VALUES (?, ?, ?)"))
	require.NoError(suite.T(), err)
	defer stmt.Close()

	for _, user := range users {
		_, err = stmt.ExecContext(ctx, user.Email, user.Name, user.Age)
		require.NoError(suite.T(), err)
	}

	// Verify bulk insert
	var userCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email IN (?, ?)", "nilu@prisma.io", "mahmoud@prisma.io").Scan(&userCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), len(users), userCount)

	// Commit the transaction
	err = tx.Commit()
	require.NoError(suite.T(), err)

	// Verify data persists after commit
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	require.NoError(suite.T(), err)
	require.GreaterOrEqual(suite.T(), userCount, 3) // At least Alice + Nilu + Mahmoud
}
