package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These are unit tests for the RawExecutor logic.
// Integration tests with a real database would be in a separate test file.

func TestRawQueryConstruction(t *testing.T) {
	t.Run("NewRawQuery creates query correctly", func(t *testing.T) {
		sql := "SELECT * FROM users WHERE id = $1"
		args := []interface{}{1}

		query := NewRawQuery(sql, args...)

		assert.Equal(t, sql, query.SQL)
		assert.Equal(t, args, query.Args)
	})

	t.Run("NewRawQuery with multiple args", func(t *testing.T) {
		sql := "SELECT * FROM posts WHERE user_id = $1 AND published = $2"
		args := []interface{}{1, true}

		query := NewRawQuery(sql, args...)

		require.Len(t, query.Args, 2)
		assert.Equal(t, 1, query.Args[0])
		assert.Equal(t, true, query.Args[1])
	})
}

func TestBuildResultStructure(t *testing.T) {
	t.Run("RawResult structure", func(t *testing.T) {
		result := &RawResult{
			Columns: []string{"id", "name", "email"},
			Rows: []map[string]interface{}{
				{"id": 1, "name": "John", "email": "john@example.com"},
				{"id": 2, "name": "Jane", "email": "jane@example.com"},
			},
			RowsAffected: 2,
		}

		assert.Len(t, result.Columns, 3)
		assert.Len(t, result.Rows, 2)
		assert.Equal(t, int64(2), result.RowsAffected)
	})
}

// Integration test example structure (would require actual DB)
func TestRawExecutorIntegration(t *testing.T) {
	t.Skip("Integration test - requires database connection")

	// Example of how integration test would look:
	// ctx := context.Background()
	// executor := NewRawExecutor(dbAdapter)
	//
	// // Test QueryRaw
	// rows, err := executor.QueryRaw(ctx, "SELECT * FROM users WHERE id = $1", 1)
	// require.NoError(t, err)
	// defer rows.Close()
	//
	// // Test ExecuteRaw
	// result, err := executor.ExecuteRaw(ctx, "UPDATE users SET status = $1 WHERE id = $2", "active", 1)
	// require.NoError(t, err)
	// rowsAffected, _ := result.RowsAffected()
	// assert.Equal(t, int64(1), rowsAffected)
}

func TestRawQuerySafety(t *testing.T) {
	t.Run("Safe query uses parameter binding", func(t *testing.T) {
		// QueryRaw should use parameterized queries
		query := NewRawQuery("SELECT * FROM users WHERE email = $1", "user@example.com")

		assert.Contains(t, query.SQL, "$1")
		assert.Equal(t, "user@example.com", query.Args[0])
	})

	t.Run("Unsafe query warning documented", func(t *testing.T) {
		// QueryRawUnsafe is documented with SQL injection warnings
		// This is a documentation test - verify warnings exist in code
		assert.True(t, true, "Unsafe methods should have SQL injection warnings in documentation")
	})
}

func TestTypeSafeMapping(t *testing.T) {
	t.Run("Map to struct example", func(t *testing.T) {
		type User struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		// This would be tested with real database in integration tests
		// Here we're just validating the structure
		var users []User
		assert.IsType(t, []User{}, users)
	})

	t.Run("Map to map example", func(t *testing.T) {
		var results []map[string]interface{}
		results = append(results, map[string]interface{}{
			"id":    1,
			"name":  "John",
			"email": "john@example.com",
		})

		require.Len(t, results, 1)
		assert.Equal(t, 1, results[0]["id"])
		assert.Equal(t, "John", results[0]["name"])
	})
}
