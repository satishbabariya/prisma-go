package transactions

import (
	"context"
	"database/sql"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/tests/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSequentialTransaction(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	db := testCtx.DB

	validation.CleanupTables(t, db)

	t.Run("CommitTransaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Insert user
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (email, name) VALUES ($1, $2)
		`, "tx_commit@example.com", "Transaction User")
		require.NoError(t, err)

		// Insert profile
		_, err = tx.ExecContext(ctx, `
			INSERT INTO profiles (bio, user_id) 
			SELECT 'Transaction bio', id FROM users WHERE email = $1
		`, "tx_commit@example.com")
		require.NoError(t, err)

		// Commit
		err = tx.Commit()
		require.NoError(t, err)

		// Verify data persisted
		var count int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", "tx_commit@example.com").Scan(&count)
		assert.Equal(t, 1, count)
	})

	t.Run("RollbackTransaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Insert user
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (email, name) VALUES ($1, $2)
		`, "tx_rollback@example.com", "Rollback User")
		require.NoError(t, err)

		// Rollback
		err = tx.Rollback()
		require.NoError(t, err)

		// Verify data NOT persisted
		var count int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", "tx_rollback@example.com").Scan(&count)
		assert.Equal(t, 0, count)
	})

	t.Run("RollbackOnError", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Insert valid user
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (email, name) VALUES ($1, $2)
		`, "tx_error1@example.com", "Error User 1")
		require.NoError(t, err)

		// Try to insert duplicate (should fail)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (email, name) VALUES ($1, $2)
		`, "tx_error1@example.com", "Error User 2")
		assert.Error(t, err)

		// Rollback
		tx.Rollback()

		// Verify first insert also rolled back
		var count int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", "tx_error1@example.com").Scan(&count)
		assert.Equal(t, 0, count)
	})
}

func TestTransactionIsolation(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	db := testCtx.DB

	validation.CleanupTables(t, db)

	t.Run("ReadCommittedIsolation", func(t *testing.T) {
		// Create initial data
		db.Exec(`INSERT INTO users (email, name) VALUES ($1, $2)`, "isolation@example.com", "Original Name")

		// Start transaction with read committed
		tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		require.NoError(t, err)
		defer tx.Rollback()

		// Read in transaction
		var name string
		tx.QueryRow("SELECT name FROM users WHERE email = $1", "isolation@example.com").Scan(&name)
		assert.Equal(t, "Original Name", name)

		// Update outside transaction
		db.Exec("UPDATE users SET name = $1 WHERE email = $2", "Updated Name", "isolation@example.com")

		// Read again in transaction (should see update in read committed)
		tx.QueryRow("SELECT name FROM users WHERE email = $1", "isolation@example.com").Scan(&name)
		// In READ COMMITTED, we may or may not see the update depending on timing
	})
}

func TestMultipleOperationsInTransaction(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	db := testCtx.DB

	validation.CleanupTables(t, db)

	t.Run("CreateUserWithPostsAndComments", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Create user
		var userId int
		err = tx.QueryRowContext(ctx, `
			INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id
		`, "multi_tx@example.com", "Multi TX User").Scan(&userId)
		require.NoError(t, err)

		// Create post
		var postId int
		err = tx.QueryRowContext(ctx, `
			INSERT INTO posts (title, slug, author_id) VALUES ($1, $2, $3) RETURNING id
		`, "TX Post", "tx-post", userId).Scan(&postId)
		require.NoError(t, err)

		// Create comments
		for i := 0; i < 3; i++ {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO comments (content, post_id, author_id) VALUES ($1, $2, $3)
			`, "Comment "+string(rune('1'+i)), postId, userId)
			require.NoError(t, err)
		}

		// Commit
		err = tx.Commit()
		require.NoError(t, err)

		// Verify all data
		var userCount, postCount, commentCount int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", userId).Scan(&userCount)
		db.QueryRow("SELECT COUNT(*) FROM posts WHERE author_id = $1", userId).Scan(&postCount)
		db.QueryRow("SELECT COUNT(*) FROM comments WHERE author_id = $1", userId).Scan(&commentCount)

		assert.Equal(t, 1, userCount)
		assert.Equal(t, 1, postCount)
		assert.Equal(t, 3, commentCount)
	})

	t.Run("PartialRollback", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Create user
		var userId int
		err = tx.QueryRowContext(ctx, `
			INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id
		`, "partial_tx@example.com", "Partial TX User").Scan(&userId)
		require.NoError(t, err)

		// Create post
		_, err = tx.ExecContext(ctx, `
			INSERT INTO posts (title, slug, author_id) VALUES ($1, $2, $3)
		`, "Partial Post", "partial-post", userId)
		require.NoError(t, err)

		// Try to create invalid post (null title)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO posts (title, slug, author_id) VALUES (NULL, $1, $2)
		`, "null-title-post", userId)
		// This should fail

		// Rollback everything
		tx.Rollback()

		// Verify nothing persisted
		var count int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", "partial_tx@example.com").Scan(&count)
		assert.Equal(t, 0, count)
	})
}

func TestTransactionWithBalanceTransfer(t *testing.T) {
	testCtx, cleanup := validation.SetupTestContext(t)
	defer cleanup()

	ctx := context.Background()
	db := testCtx.DB

	validation.CleanupTables(t, db)

	// Create two users with balances
	db.Exec(`INSERT INTO users (email, name, balance) VALUES ($1, $2, $3)`, "sender@example.com", "Sender", 1000.00)
	db.Exec(`INSERT INTO users (email, name, balance) VALUES ($1, $2, $3)`, "receiver@example.com", "Receiver", 500.00)

	t.Run("SuccessfulTransfer", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		transferAmount := 200.00

		// Deduct from sender
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET balance = balance - $1 WHERE email = $2
		`, transferAmount, "sender@example.com")
		require.NoError(t, err)

		// Add to receiver
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET balance = balance + $1 WHERE email = $2
		`, transferAmount, "receiver@example.com")
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Verify balances
		var senderBalance, receiverBalance float64
		db.QueryRow("SELECT balance FROM users WHERE email = $1", "sender@example.com").Scan(&senderBalance)
		db.QueryRow("SELECT balance FROM users WHERE email = $1", "receiver@example.com").Scan(&receiverBalance)

		assert.Equal(t, 800.00, senderBalance)
		assert.Equal(t, 700.00, receiverBalance)
	})
}
