package schema

import (
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataRegistry(t *testing.T) {
	t.Run("LoadFromSchema and GetModel", func(t *testing.T) {
		schema := &domain.Schema{
			Models: []domain.Model{
				{
					Name: "User",
					Fields: []domain.Field{
						{Name: "id", Type: domain.FieldType{Name: "String"}},
						{Name: "email", Type: domain.FieldType{Name: "String"}},
					},
				},
				{
					Name: "Post",
					Fields: []domain.Field{
						{Name: "id", Type: domain.FieldType{Name: "String"}},
						{Name: "title", Type: domain.FieldType{Name: "String"}},
					},
				},
			},
		}

		registry := NewMetadataRegistry()
		err := registry.LoadFromSchema(schema)
		require.NoError(t, err)

		// Test GetModel
		user, err := registry.GetModel("User")
		require.NoError(t, err)
		assert.Equal(t, "User", user.Name)
		assert.Len(t, user.Fields, 2)

		post, err := registry.GetModel("Post")
		require.NoError(t, err)
		assert.Equal(t, "Post", post.Name)

		// Test non-existent model
		_, err = registry.GetModel("NonExistent")
		assert.Error(t, err)
	})

	t.Run("GetField", func(t *testing.T) {
		schema := &domain.Schema{
			Models: []domain.Model{
				{
					Name: "User",
					Fields: []domain.Field{
						{Name: "id", Type: domain.FieldType{Name: "String"}},
						{Name: "email", Type: domain.FieldType{Name: "String"}, IsUnique: true},
					},
				},
			},
		}

		registry := NewMetadataRegistry()
		err := registry.LoadFromSchema(schema)
		require.NoError(t, err)

		// Test GetField
		email, err := registry.GetField("User", "email")
		require.NoError(t, err)
		assert.Equal(t, "email", email.Name)
		assert.True(t, email.IsUnique)

		// Test non-existent field
		_, err = registry.GetField("User", "nonexistent")
		assert.Error(t, err)
	})

	t.Run("IsEnum", func(t *testing.T) {
		schema := &domain.Schema{
			Enums: []domain.Enum{
				{
					Name:   "Role",
					Values: []string{"USER", "ADMIN"},
				},
			},
		}

		registry := NewMetadataRegistry()
		err := registry.LoadFromSchema(schema)
		require.NoError(t, err)

		assert.True(t, registry.IsEnum("Role"))
		assert.False(t, registry.IsEnum("NotAnEnum"))
	})

	t.Run("Relation detection", func(t *testing.T) {
		schema := &domain.Schema{
			Models: []domain.Model{
				{
					Name: "User",
					Fields: []domain.Field{
						{Name: "id", Type: domain.FieldType{Name: "String"}},
						{Name: "posts", Type: domain.FieldType{Name: "Post"}, IsList: true},
					},
				},
				{
					Name: "Post",
					Fields: []domain.Field{
						{Name: "id", Type: domain.FieldType{Name: "String"}},
						{Name: "author", Type: domain.FieldType{Name: "User"}},
					},
				},
			},
		}

		registry := NewMetadataRegistry()
		err := registry.LoadFromSchema(schema)
		require.NoError(t, err)

		// Check that relations were detected
		userRelations, err := registry.GetAllRelations("User")
		require.NoError(t, err)
		assert.Len(t, userRelations, 1)
		assert.Equal(t, "posts", userRelations[0].Name)
		assert.Equal(t, "Post", userRelations[0].ToModel)
	})

	t.Run("GetTableName and GetColumnName", func(t *testing.T) {
		schema := &domain.Schema{
			Models: []domain.Model{
				{
					Name: "User",
					Fields: []domain.Field{
						{Name: "id", Type: domain.FieldType{Name: "String"}},
						{Name: "email", Type: domain.FieldType{Name: "String"}},
					},
				},
			},
		}

		registry := NewMetadataRegistry()
		err := registry.LoadFromSchema(schema)
		require.NoError(t, err)

		tableName, err := registry.GetTableName("User")
		require.NoError(t, err)
		assert.Equal(t, "User", tableName)

		columnName, err := registry.GetColumnName("User", "email")
		require.NoError(t, err)
		assert.Equal(t, "email", columnName)
	})
}

func TestMetadataRegistry_ConcurrentAccess(t *testing.T) {
	schema := &domain.Schema{
		Models: []domain.Model{
			{Name: "User", Fields: []domain.Field{{Name: "id", Type: domain.FieldType{Name: "String"}}}},
		},
	}

	registry := NewMetadataRegistry()
	err := registry.LoadFromSchema(schema)
	require.NoError(t, err)

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := registry.GetModel("User")
			assert.NoError(t, err)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
