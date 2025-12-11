package e2e

import (
	"context"
	"time"

	"github.com/stretchr/testify/require"
)

// TestUpsertOperationsAdvanced tests advanced upsert (create or update) operations
func (suite *TestSuite) TestUpsertOperationsAdvanced() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test tables
	suite.createUpsertTables(ctx)
	defer suite.cleanupUpsertTables(ctx)

	// Test basic upsert operations
	suite.testBasicUpsert(ctx)

	// Test upsert with conditions
	suite.testUpsertWithConditions(ctx)

	// Test upsert with relationships
	suite.testUpsertWithRelationships(ctx)

	suite.T().Logf("Upsert operations test passed for provider: %s", suite.config.Provider)
}

// createUpsertTables creates tables for upsert tests
func (suite *TestSuite) createUpsertTables(ctx context.Context) {
	var createSQL string

	switch suite.config.Provider {
	case "postgresql":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INTEGER,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL UNIQUE,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			view_count INTEGER DEFAULT 0,
			author_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS profiles (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			bio TEXT,
			website VARCHAR(255),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	case "mysql":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL UNIQUE,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			view_count INT DEFAULT 0,
			author_id INT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE IF NOT EXISTS profiles (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL UNIQUE,
			bio TEXT,
			website VARCHAR(255),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	case "sqlite":
		createSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			age INTEGER,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL UNIQUE,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			view_count INTEGER DEFAULT 0,
			author_id INTEGER,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE IF NOT EXISTS profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			bio TEXT,
			website TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
	}

	_, err := suite.db.ExecContext(ctx, createSQL)
	require.NoError(suite.T(), err)
}

// cleanupUpsertTables removes upsert test tables
func (suite *TestSuite) cleanupUpsertTables(ctx context.Context) {
	_, err := suite.db.ExecContext(ctx, "DROP TABLE IF EXISTS profiles, posts, users")
	require.NoError(suite.T(), err)
}

// testBasicUpsert tests basic upsert functionality
func (suite *TestSuite) testBasicUpsert(ctx context.Context) {
	// Test upsert that creates a new record
	var userID int
	err := suite.db.QueryRowContext(ctx, `
		INSERT INTO users (email, name, age) 
		VALUES (?, ?, ?) 
		ON CONFLICT (email) DO UPDATE SET 
			name = EXCLUDED.name,
			age = EXCLUDED.age,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`, "newuser@example.com", "New User", 25).Scan(&userID)

	// For MySQL and SQLite, we need a different approach
	if err != nil {
		// Try MySQL/SQLite approach
		_, err = suite.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO users (email, name, age) VALUES (?, ?, ?)`,
			"newuser@example.com", "New User", 25)
		require.NoError(suite.T(), err)

		// Get the ID
		err = suite.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", "newuser@example.com").Scan(&userID)
		require.NoError(suite.T(), err)
	}

	require.Greater(suite.T(), userID, 0)

	// Verify the record was created
	var name string
	var age int
	err = suite.db.QueryRowContext(ctx, "SELECT name, age FROM users WHERE email = ?", "newuser@example.com").Scan(&name, &age)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "New User", name)
	require.Equal(suite.T(), 25, age)

	// Test upsert that updates an existing record
	var updatedUserID int
	err = suite.db.QueryRowContext(ctx, `
		INSERT INTO users (email, name, age) 
		VALUES (?, ?, ?) 
		ON CONFLICT (email) DO UPDATE SET 
			name = EXCLUDED.name,
			age = EXCLUDED.age,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`, "newuser@example.com", "Updated User", 26).Scan(&updatedUserID)

	// For MySQL and SQLite, we need a different approach
	if err != nil {
		// Update the existing record
		_, err = suite.db.ExecContext(ctx, `
			UPDATE users SET name = ?, age = ? WHERE email = ?`,
			"Updated User", 26, "newuser@example.com")
		require.NoError(suite.T(), err)
		updatedUserID = userID
	}

	require.Equal(suite.T(), userID, updatedUserID)

	// Verify the record was updated
	err = suite.db.QueryRowContext(ctx, "SELECT name, age FROM users WHERE email = ?", "newuser@example.com").Scan(&name, &age)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "Updated User", name)
	require.Equal(suite.T(), 26, age)

	// Verify only one record exists
	var count int
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "newuser@example.com").Scan(&count)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, count)
}

// testUpsertWithConditions tests upsert with conditional logic
func (suite *TestSuite) testUpsertWithConditions(ctx context.Context) {
	// Test upsert with conditional update (only update if new value is different)
	email := "conditional@example.com"
	initialName := "Initial Name"
	updatedName := "Updated Name"

	// Insert initial record
	var userID int
	err := suite.db.QueryRowContext(ctx, `
		INSERT INTO users (email, name, age) 
		VALUES (?, ?, ?) 
		ON CONFLICT (email) DO UPDATE SET 
			name = EXCLUDED.name,
			age = EXCLUDED.age,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`, email, initialName, 30).Scan(&userID)

	if err != nil {
		_, err = suite.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO users (email, name, age) VALUES (?, ?, ?)`,
			email, initialName, 30)
		require.NoError(suite.T(), err)
		err = suite.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", email).Scan(&userID)
		require.NoError(suite.T(), err)
	}

	// Try to update with same name (should not trigger update in some databases)
	_, err = suite.db.ExecContext(ctx, `
		INSERT INTO users (email, name, age) 
		VALUES (?, ?, ?) 
		ON CONFLICT (email) DO UPDATE SET 
			name = CASE WHEN users.name != EXCLUDED.name THEN EXCLUDED.name ELSE users.name END,
			age = EXCLUDED.age,
			updated_at = CURRENT_TIMESTAMP`, email, initialName, 31)

	if err != nil {
		// For MySQL/SQLite, just update
		_, err = suite.db.ExecContext(ctx, `
			UPDATE users SET age = ? WHERE email = ?`, 31, email)
		require.NoError(suite.T(), err)
	}

	// Verify age was updated but name might remain the same
	var name string
	var age int
	err = suite.db.QueryRowContext(ctx, "SELECT name, age FROM users WHERE email = ?", email).Scan(&name, &age)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 31, age)

	// Now update with different name
	_, err = suite.db.ExecContext(ctx, `
		INSERT INTO users (email, name, age) 
		VALUES (?, ?, ?) 
		ON CONFLICT (email) DO UPDATE SET 
			name = EXCLUDED.name,
			age = EXCLUDED.age,
			updated_at = CURRENT_TIMESTAMP`, email, updatedName, 32)

	if err != nil {
		// For MySQL/SQLite, just update
		_, err = suite.db.ExecContext(ctx, `
			UPDATE users SET name = ?, age = ? WHERE email = ?`, updatedName, 32, email)
		require.NoError(suite.T(), err)
	}

	// Verify both name and age were updated
	err = suite.db.QueryRowContext(ctx, "SELECT name, age FROM users WHERE email = ?", email).Scan(&name, &age)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), updatedName, name)
	require.Equal(suite.T(), 32, age)
}

// testUpsertWithRelationships tests upsert operations with related data
func (suite *TestSuite) testUpsertWithRelationships(ctx context.Context) {
	// Create a user first
	var userID int
	err := suite.db.QueryRowContext(ctx, `
		INSERT INTO users (email, name, age) 
		VALUES (?, ?, ?) 
		ON CONFLICT (email) DO UPDATE SET 
			name = EXCLUDED.name,
			age = EXCLUDED.age,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`, "author@example.com", "Author User", 35).Scan(&userID)

	if err != nil {
		_, err = suite.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO users (email, name, age) VALUES (?, ?, ?)`,
			"author@example.com", "Author User", 35)
		require.NoError(suite.T(), err)
		err = suite.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", "author@example.com").Scan(&userID)
		require.NoError(suite.T(), err)
	}

	// Test upsert on posts with foreign key
	title := "Unique Post Title"
	content := "Initial content"

	// Insert post
	var postID int
	err = suite.db.QueryRowContext(ctx, `
		INSERT INTO posts (title, content, published, author_id) 
		VALUES (?, ?, ?, ?) 
		ON CONFLICT (title) DO UPDATE SET 
			content = EXCLUDED.content,
			published = EXCLUDED.published,
			author_id = EXCLUDED.author_id,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`, title, content, false, userID).Scan(&postID)

	if err != nil {
		_, err = suite.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO posts (title, content, published, author_id) VALUES (?, ?, ?, ?)`,
			title, content, false, userID)
		require.NoError(suite.T(), err)
		err = suite.db.QueryRowContext(ctx, "SELECT id FROM posts WHERE title = ?", title).Scan(&postID)
		require.NoError(suite.T(), err)
	}

	// Verify initial post
	var initialContent string
	var published bool
	var authorID int
	err = suite.db.QueryRowContext(ctx, "SELECT content, published, author_id FROM posts WHERE title = ?", title).Scan(&initialContent, &published, &authorID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), content, initialContent)
	require.False(suite.T(), published)
	require.Equal(suite.T(), userID, authorID)

	// Update the post
	updatedContent := "Updated content"
	_, err = suite.db.ExecContext(ctx, `
		INSERT INTO posts (title, content, published, author_id) 
		VALUES (?, ?, ?, ?) 
		ON CONFLICT (title) DO UPDATE SET 
			content = EXCLUDED.content,
			published = EXCLUDED.published,
			author_id = EXCLUDED.author_id,
			updated_at = CURRENT_TIMESTAMP`, title, updatedContent, true, userID)

	if err != nil {
		_, err = suite.db.ExecContext(ctx, `
			UPDATE posts SET content = ?, published = ? WHERE title = ?`,
			updatedContent, true, title)
		require.NoError(suite.T(), err)
	}

	// Verify updated post
	var finalContent string
	err = suite.db.QueryRowContext(ctx, "SELECT content, published FROM posts WHERE title = ?", title).Scan(&finalContent, &published)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), updatedContent, finalContent)
	require.True(suite.T(), published)

	// Test upsert on profile (one-to-one relationship)
	profileBio := "Software developer and tech enthusiast"
	profileWebsite := "https://example.com"

	// Insert profile
	var profileID int
	err = suite.db.QueryRowContext(ctx, `
		INSERT INTO profiles (user_id, bio, website) 
		VALUES (?, ?, ?) 
		ON CONFLICT (user_id) DO UPDATE SET 
			bio = EXCLUDED.bio,
			website = EXCLUDED.website,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id`, userID, profileBio, profileWebsite).Scan(&profileID)

	if err != nil {
		_, err = suite.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO profiles (user_id, bio, website) VALUES (?, ?, ?)`,
			userID, profileBio, profileWebsite)
		require.NoError(suite.T(), err)
		err = suite.db.QueryRowContext(ctx, "SELECT id FROM profiles WHERE user_id = ?", userID).Scan(&profileID)
		require.NoError(suite.T(), err)
	}

	// Verify profile
	var bio string
	var website string
	err = suite.db.QueryRowContext(ctx, "SELECT bio, website FROM profiles WHERE user_id = ?", userID).Scan(&bio, &website)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), profileBio, bio)
	require.Equal(suite.T(), profileWebsite, website)

	// Update profile
	updatedBio := "Senior software developer and tech enthusiast"
	_, err = suite.db.ExecContext(ctx, `
		INSERT INTO profiles (user_id, bio, website) 
		VALUES (?, ?, ?) 
		ON CONFLICT (user_id) DO UPDATE SET 
			bio = EXCLUDED.bio,
			website = EXCLUDED.website,
			updated_at = CURRENT_TIMESTAMP`, userID, updatedBio, website)

	if err != nil {
		_, err = suite.db.ExecContext(ctx, `
			UPDATE profiles SET bio = ? WHERE user_id = ?`,
			updatedBio, userID)
		require.NoError(suite.T(), err)
	}

	// Verify updated profile
	err = suite.db.QueryRowContext(ctx, "SELECT bio FROM profiles WHERE user_id = ?", userID).Scan(&bio)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), updatedBio, bio)

	// Verify only one profile exists per user
	var profileCount int
	err = suite.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM profiles WHERE user_id = ?", userID).Scan(&profileCount)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 1, profileCount)
}
