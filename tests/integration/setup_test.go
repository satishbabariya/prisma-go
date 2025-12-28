package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaSQL is a test schema for integration tests.
const TestSchemaSQL = `
CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) UNIQUE NOT NULL,
	name VARCHAR(255) NOT NULL,
	active BOOLEAN DEFAULT TRUE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts (
	id SERIAL PRIMARY KEY,
	title VARCHAR(255) NOT NULL,
	content TEXT,
	published BOOLEAN DEFAULT FALSE,
	author_id INTEGER NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comments (
	id SERIAL PRIMARY KEY,
	content TEXT NOT NULL,
	post_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE  
);
`

// TestPostgresSetup tests PostgreSQL database setup.
func TestPostgresSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgresTest(t)
	defer db.Cleanup()

	// Test connection
	require.NoError(t, db.DB.Ping())

	// Run migrations
	err := db.RunMigrations(TestSchemaSQL)
	require.NoError(t, err)

	// Verify tables exist
	var count int
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name IN ('users', 'posts', 'comments')
	`).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 3, count, "Expected 3 tables to be created")
}

// TestMySQLSetup tests MySQL database setup.
func TestMySQLSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupMySQLTest(t)
	defer db.Cleanup()

	require.NoError(t, db.DB.Ping())

	// Execute statements separately for MySQL
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			author_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, stmt := range statements {
		_, err := db.DB.Exec(stmt)
		require.NoError(t, err)
	}
}

// TestSQLiteSetup tests SQLite database setup.
func TestSQLiteSetup(t *testing.T) {
	db := SetupSQLiteTest(t)
	defer db.Cleanup()

	require.NoError(t, db.DB.Ping())

	sqliteSchema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			active INTEGER DEFAULT 1
		);

		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT,
			author_id INTEGER NOT NULL,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`

	err := db.RunMigrations(sqliteSchema)
	require.NoError(t, err)
}

// TestFixtureLoading tests fixture data loading.
func TestFixtureLoading(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgresTest(t)
	defer db.Cleanup()

	err := db.RunMigrations(TestSchemaSQL)
	require.NoError(t, err)

	// Load user fixtures
	users := []map[string]interface{}{
		{"email": "user1@example.com", "name": "User One", "active": true},
		{"email": "user2@example.com", "name": "User Two", "active": true},
	}

	err = db.LoadFixture("users", users)
	require.NoError(t, err)

	// Verify data loaded
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
