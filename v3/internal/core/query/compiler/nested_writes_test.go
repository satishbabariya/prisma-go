package compiler

import (
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileNestedWrites(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)

	t.Run("NestedCreate", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedCreate,
				Data: map[string]interface{}{
					"title":   "New Post",
					"content": "Post content",
				},
			},
		}

		statements, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		sql := statements[0]
		assert.Contains(t, sql.Query, "INSERT INTO posts")
		assert.Contains(t, sql.Query, "users_id")
		assert.Contains(t, sql.Query, "title")
		assert.Contains(t, sql.Query, "content")
		assert.Contains(t, sql.Query, "RETURNING *")
		assert.Equal(t, 3, len(sql.Args)) // users_id, title, content
		assert.Equal(t, 1, sql.Args[0])
	})

	t.Run("NestedConnect", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedConnect,
				Where: []domain.Condition{
					{Field: "id", Operator: domain.Equals, Value: 10},
				},
			},
		}

		statements, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		sql := statements[0]
		assert.Contains(t, sql.Query, "UPDATE posts")
		assert.Contains(t, sql.Query, "SET users_id =")
		assert.Contains(t, sql.Query, "WHERE")
		assert.Contains(t, sql.Query, "id")
	})

	t.Run("NestedDisconnect", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedDisconnect,
				Where: []domain.Condition{
					{Field: "id", Operator: domain.Equals, Value: 10},
				},
			},
		}

		statements, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		sql := statements[0]
		assert.Contains(t, sql.Query, "UPDATE posts")
		assert.Contains(t, sql.Query, "SET users_id = NULL")
		assert.Contains(t, sql.Query, "WHERE")
	})

	t.Run("NestedUpdate", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedUpdate,
				Data: map[string]interface{}{
					"title": "Updated Title",
				},
				Where: []domain.Condition{
					{Field: "id", Operator: domain.Equals, Value: 10},
				},
			},
		}

		statements, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		sql := statements[0]
		assert.Contains(t, sql.Query, "UPDATE posts")
		assert.Contains(t, sql.Query, "SET title =")
		assert.Contains(t, sql.Query, "WHERE users_id =")
		assert.Contains(t, sql.Query, "AND id")
	})

	t.Run("NestedDelete", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedDelete,
				Where: []domain.Condition{
					{Field: "id", Operator: domain.Equals, Value: 10},
				},
			},
		}

		statements, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		sql := statements[0]
		assert.Contains(t, sql.Query, "DELETE FROM posts")
		assert.Contains(t, sql.Query, "WHERE users_id =")
		assert.Contains(t, sql.Query, "AND id")
	})

	t.Run("NestedSet", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedSet,
				Where: []domain.Condition{
					{Field: "id", Operator: domain.In, Value: []interface{}{10, 20, 30}},
				},
			},
		}

		statements, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		require.NoError(t, err)
		// Should have disconnect + connect statements
		assert.GreaterOrEqual(t, len(statements), 2)

		// First statement should disconnect all
		assert.Contains(t, statements[0].Query, "UPDATE posts")
		assert.Contains(t, statements[0].Query, "SET users_id = NULL")

		// Second statement should connect specified ones
		assert.Contains(t, statements[1].Query, "UPDATE posts")
		assert.Contains(t, statements[1].Query, "SET users_id =")
		assert.Contains(t, statements[1].Query, "WHERE")
	})

	t.Run("MultipleNestedOperations", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedCreate,
				Data: map[string]interface{}{
					"title": "New Post",
				},
			},
			{
				Relation:  "comments",
				Operation: domain.NestedCreate,
				Data: map[string]interface{}{
					"text": "New Comment",
				},
			},
		}

		statements, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		require.NoError(t, err)
		assert.Len(t, statements, 2)

		assert.Contains(t, statements[0].Query, "INSERT INTO posts")
		assert.Contains(t, statements[1].Query, "INSERT INTO comments")
	})
}

func TestNestedWriteValidation(t *testing.T) {
	compiler := NewSQLCompiler(domain.PostgreSQL)

	t.Run("NestedCreateRequiresData", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedCreate,
				Data:      map[string]interface{}{}, // Empty data
			},
		}

		_, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nested create requires data")
	})

	t.Run("NestedConnectRequiresWhere", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedConnect,
				Where:     []domain.Condition{}, // Empty where
			},
		}

		_, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nested connect requires where conditions")
	})

	t.Run("NestedUpdateRequiresData", func(t *testing.T) {
		nestedWrites := []domain.NestedWrite{
			{
				Relation:  "posts",
				Operation: domain.NestedUpdate,
				Data:      map[string]interface{}{}, // Empty data
				Where: []domain.Condition{
					{Field: "id", Operator: domain.Equals, Value: 1},
				},
			},
		}

		_, err := compiler.CompileNestedWrites("users", 1, nestedWrites)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nested update requires data")
	})
}
