package introspection_test

import (
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/generator"
	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/inference"
	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/mapper"
	schemadomain "github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgreSQLTypeMapping tests the PostgreSQL type mapper.
func TestPostgreSQLTypeMapping(t *testing.T) {
	mapper := mapper.NewPostgreSQLTypeMapper()

	tests := []struct {
		dbType     string
		isNullable bool
		expected   string
	}{
		{"VARCHAR", false, "String"},
		{"VARCHAR", true, "String?"},
		{"INTEGER", false, "Int"},
		{"BIGINT", true, "BigInt?"},
		{"BOOLEAN", false, "Boolean"},
		{"TIMESTAMP", false, "DateTime"},
		{"JSON", false, "Json"},
		{"BYTEA", false, "Bytes"},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			result := mapper.MapToPrismaType(tt.dbType, tt.isNullable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRelationInference tests relation inference logic.
func TestRelationInference(t *testing.T) {
	inferrer := inference.NewRelationInferrer()

	// Create test database with User and Post tables
	db := &domain.IntrospectedDatabase{
		Tables: []domain.IntrospectedTable{
			{
				Name: "users",
				Columns: []domain.IntrospectedColumn{
					{Name: "id", Type: "INTEGER", IsPrimaryKey: true},
					{Name: "email", Type: "VARCHAR", IsUnique: true},
				},
				PrimaryKey: &domain.IntrospectedPrimaryKey{
					Name:    "users_pkey",
					Columns: []string{"id"},
				},
			},
			{
				Name: "posts",
				Columns: []domain.IntrospectedColumn{
					{Name: "id", Type: "INTEGER", IsPrimaryKey: true},
					{Name: "title", Type: "VARCHAR"},
					{Name: "author_id", Type: "INTEGER"},
				},
				ForeignKeys: []domain.IntrospectedForeignKey{
					{
						Name:              "posts_author_fkey",
						ColumnNames:       []string{"author_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          domain.Cascade,
						OnUpdate:          domain.NoAction,
					},
				},
				PrimaryKey: &domain.IntrospectedPrimaryKey{
					Name:    "posts_pkey",
					Columns: []string{"id"},
				},
			},
		},
	}

	relations, err := inferrer.InferRelations(db)
	require.NoError(t, err)

	// Should have 2 relations (bidirectional)
	assert.Len(t, relations, 2)

	// Find the ManyToOne relation (Post -> User)
	var postToUser *domain.InferredRelation
	for i := range relations {
		if relations[i].FromTable == "posts" && relations[i].ToTable == "users" {
			postToUser = &relations[i]
			break
		}
	}

	require.NotNil(t, postToUser)
	assert.Equal(t, domain.ManyToOne, postToUser.RelationType)
	assert.Equal(t, "author", postToUser.RelationName)
	assert.Equal(t, []string{"author_id"}, postToUser.FromFields)
	assert.Equal(t, []string{"id"}, postToUser.ToFields)
	assert.Equal(t, domain.Cascade, postToUser.OnDelete)
}

// TestSchemaGeneration tests the full schema generation process.
func TestSchemaGeneration(t *testing.T) {
	typeMapper := mapper.NewPostgreSQLTypeMapper()
	relationInferrer := inference.NewRelationInferrer()
	schemaGen := generator.NewSchemaGenerator(typeMapper, relationInferrer)

	// Create introspected database
	db := &domain.IntrospectedDatabase{
		Tables: []domain.IntrospectedTable{
			{
				Name: "users",
				Columns: []domain.IntrospectedColumn{
					{Name: "id", Type: "SERIAL", IsPrimaryKey: true, IsAutoIncrement: true},
					{Name: "email", Type: "VARCHAR", IsUnique: true},
					{Name: "name", Type: "VARCHAR", IsNullable: true},
				},
				Indexes: []domain.IntrospectedIndex{
					{
						Name:     "users_email_idx",
						Columns:  []string{"email"},
						IsUnique: true,
					},
				},
				PrimaryKey: &domain.IntrospectedPrimaryKey{
					Name:    "users_pkey",
					Columns: []string{"id"},
				},
			},
			{
				Name: "posts",
				Columns: []domain.IntrospectedColumn{
					{Name: "id", Type: "SERIAL", IsPrimaryKey: true, IsAutoIncrement: true},
					{Name: "title", Type: "VARCHAR"},
					{Name: "content", Type: "TEXT", IsNullable: true},
					{Name: "published", Type: "BOOLEAN", DefaultValue: strPtr("false")},
					{Name: "author_id", Type: "INTEGER"},
				},
				ForeignKeys: []domain.IntrospectedForeignKey{
					{
						Name:              "posts_author_fkey",
						ColumnNames:       []string{"author_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
		Enums: []domain.IntrospectedEnum{
			{
				Name:   "user_role",
				Values: []string{"USER", "ADMIN"},
			},
		},
	}

	datasource := schemadomain.Datasource{
		Name:     "db",
		Provider: "postgresql",
		URL:      "postgresql://localhost:5432/test",
	}

	// Generate schema
	schema, err := schemaGen.Generate(db, datasource)
	require.NoError(t, err)

	// Verify datasource
	require.Len(t, schema.Datasources, 1)
	assert.Equal(t, "db", schema.Datasources[0].Name)
	assert.Equal(t, "postgresql", schema.Datasources[0].Provider)

	// Verify models
	assert.Len(t, schema.Models, 2)

	// Find User model
	var userModel *schemadomain.Model
	for i := range schema.Models {
		if schema.Models[i].Name == "Users" {
			userModel = &schema.Models[i]
			break
		}
	}

	require.NotNil(t, userModel)
	assert.Len(t, userModel.Fields, 3+1) // 3 columns + 1 relation field

	// Check ID field attributes
	var idField *schemadomain.Field
	for i := range userModel.Fields {
		if userModel.Fields[i].Name == "id" {
			idField = &userModel.Fields[i]
			break
		}
	}

	require.NotNil(t, idField)
	assert.True(t, hasAttribute(idField.Attributes, "id"))
	assert.True(t, hasAttribute(idField.Attributes, "default"))

	// Verify enums
	assert.Len(t, schema.Enums, 1)
	assert.Equal(t, "UserRole", schema.Enums[0].Name)
	assert.Equal(t, []string{"USER", "ADMIN"}, schema.Enums[0].Values)
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func hasAttribute(attrs []schemadomain.Attribute, name string) bool {
	for _, attr := range attrs {
		if attr.Name == name {
			return true
		}
	}
	return false
}
