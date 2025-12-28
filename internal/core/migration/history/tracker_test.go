package history

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryTracker(t *testing.T) {
	t.Skip("Requires database connection")

	// This test would require a real database connection
	// Shown for documentation purposes

	db, err := sql.Open("postgres", "postgresql://localhost:5432/test")
	require.NoError(t, err)
	defer db.Close()

	tracker := NewTracker(db)

	// Ensure table exists
	err = tracker.EnsureTable()
	require.NoError(t, err)

	// Record a successful migration
	err = tracker.RecordSuccess("001_initial", "abc123", 150)
	require.NoError(t, err)

	// Get status
	status, err := tracker.GetStatus("001_initial")
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.Success)
	assert.Equal(t, "abc123", status.Checksum)
}

func TestChecksumCalculation(t *testing.T) {
	content := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
	checksum := CalculateChecksum(content)

	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64) // SHA256 produces 64 hex chars

	// Same content should produce same checksum
	checksum2 := CalculateChecksum(content)
	assert.Equal(t, checksum, checksum2)

	// Different content should produce different checksum
	checksum3 := CalculateChecksum("CREATE TABLE posts (id SERIAL);")
	assert.NotEqual(t, checksum, checksum3)
}

func TestGetPending(t *testing.T) {
	t.Skip("Requires database connection")

	db, _ := sql.Open("postgres", "postgresql://localhost:5432/test")
	defer db.Close()

	tracker := NewTracker(db)
	tracker.EnsureTable()

	// Record some applied migrations
	tracker.RecordSuccess("001_initial", "abc", 100)
	tracker.RecordSuccess("002_add_users", "def", 120)

	// Check pending migrations
	all := []string{"001_initial", "002_add_users", "003_add_posts", "004_add_comments"}
	pending, err := tracker.GetPending(all)

	require.NoError(t, err)
	assert.Len(t, pending, 2)
	assert.Contains(t, pending, "003_add_posts")
	assert.Contains(t, pending, "004_add_comments")
}
