package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCRUDOperations tests basic Create, Read, Update, Delete operations.
func TestCRUDOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgresTest(t)
	defer db.Cleanup()

	require.NoError(t, db.RunMigrations(TestSchemaSQL))

	t.Run("Create", func(t *testing.T) {
		result, err := db.DB.Exec(`
			INSERT INTO users (email, name, active) 
			VALUES ($1, $2, $3)
		`, "test@example.com", "Test User", true)

		require.NoError(t, err)
		_, err = result.LastInsertId()
		// PostgreSQL doesn't support LastInsertId, use RETURNING instead
		if err != nil {
			var userID int
			err = db.DB.QueryRow(`
				INSERT INTO users (email, name, active) 
				VALUES ($1, $2, $3) RETURNING id
			`, "test2@example.com", "Test User 2", true).Scan(&userID)
			require.NoError(t, err)
			assert.Greater(t, userID, 0)
		}
	})

	t.Run("Read", func(t *testing.T) {
		var email, name string
		var active bool

		err := db.DB.QueryRow(`
			SELECT email, name, active FROM users WHERE email = $1
		`, "test@example.com").Scan(&email, &name, &active)

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", email)
		assert.Equal(t, "Test User", name)
		assert.True(t, active)
	})

	t.Run("Update", func(t *testing.T) {
		result, err := db.DB.Exec(`
			UPDATE users SET name = $1 WHERE email = $2
		`, "Updated User", "test@example.com")

		require.NoError(t, err)
		rows, err := result.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(1), rows)

		// Verify update
		var name string
		db.DB.QueryRow("SELECT name FROM users WHERE email = $1", "test@example.com").Scan(&name)
		assert.Equal(t, "Updated User", name)
	})

	t.Run("Delete", func(t *testing.T) {
		result, err := db.DB.Exec(`DELETE FROM users WHERE email = $1`, "test@example.com")

		require.NoError(t, err)
		rows, err := result.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(1), rows)
	})
}

// TestRelations tests foreign key relationships.
func TestRelations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgresTest(t)
	defer db.Cleanup()

	require.NoError(t, db.RunMigrations(TestSchemaSQL))

	// Create user
	var userID int
	err := db.DB.QueryRow(`
		INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id
	`, "author@example.com", "Author").Scan(&userID)
	require.NoError(t, err)

	// Create post for user
	var postID int
	err = db.DB.QueryRow(`
		INSERT INTO posts (title, content, author_id) 
		VALUES ($1, $2, $3) RETURNING id
	`, "Test Post", "Test Content", userID).Scan(&postID)
	require.NoError(t, err)

	// Verify relation
	var authorName string
	err = db.DB.QueryRow(`
		SELECT u.name FROM users u
		JOIN posts p ON p.author_id = u.id
		WHERE p.id = $1
	`, postID).Scan(&authorName)

	require.NoError(t, err)
	assert.Equal(t, "Author", authorName)

	// Test cascade delete
	_, err = db.DB.Exec("DELETE FROM users WHERE id = $1", userID)
	require.NoError(t, err)

	// Verify post was also deleted
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE id = $1", postID).Scan(&count)
	assert.Equal(t, 0, count, "Post should be deleted on cascade")
}

// TestTransactions tests transaction commit and rollback.
func TestTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgresTest(t)
	defer db.Cleanup()

	require.NoError(t, db.RunMigrations(TestSchemaSQL))

	t.Run("Commit", func(t *testing.T) {
		tx, err := db.DB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec("INSERT INTO users (email, name) VALUES ($1, $2)", "tx@example.com", "TX User")
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Verify data persisted
		var count int
		db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", "tx@example.com").Scan(&count)
		assert.Equal(t, 1, count)
	})

	t.Run("Rollback", func(t *testing.T) {
		tx, err := db.DB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec("INSERT INTO users (email, name) VALUES ($1, $2)", "rollback@example.com", "Rollback User")
		require.NoError(t, err)

		err = tx.Rollback()
		require.NoError(t, err)

		// Verify data not persisted
		var count int
		db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", "rollback@example.com").Scan(&count)
		assert.Equal(t, 0, count)
	})
}

// TestConcurrency tests concurrent database access.
func TestConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgresTest(t)
	defer db.Cleanup()

	require.NoError(t, db.RunMigrations(TestSchemaSQL))

	const concurrency = 10
	done := make(chan bool, concurrency)

	// Concurrent inserts
	for i := 0; i < concurrency; i++ {
		go func(n int) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := db.DB.ExecContext(ctx, `
				INSERT INTO users (email, name) VALUES ($1, $2)
			`, string(rune('a'+n))+"@example.com", "User "+string(rune('A'+n)))

			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Verify all inserted
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	assert.Equal(t, concurrency, count)
}
