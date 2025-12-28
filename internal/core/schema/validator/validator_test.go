package validator

import (
	"context"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
	"github.com/stretchr/testify/assert"
)

func TestValidator_ValidateSchema(t *testing.T) {
	validator := NewValidator()

	t.Run("Valid schema passes", func(t *testing.T) {
		schema := &domain.Schema{
			Datasources: []domain.Datasource{
				{
					Name:     "db",
					Provider: "postgresql",
					URL:      "postgresql://localhost:5432/test",
				},
			},
			Models: []domain.Model{
				{
					Name: "User",
					Fields: []domain.Field{
						{
							Name:       "id",
							Type:       domain.FieldType{Name: "String"},
							IsRequired: true,
							Attributes: []domain.Attribute{{Name: "id"}},
						},
						{
							Name:       "email",
							Type:       domain.FieldType{Name: "String"},
							IsRequired: true,
						},
					},
				},
			},
		}

		err := validator.Validate(context.Background(), schema)
		assert.NoError(t, err)
	})

	t.Run("Schema without datasource fails", func(t *testing.T) {
		schema := &domain.Schema{
			Models: []domain.Model{
				{Name: "User", Fields: []domain.Field{{Name: "id", Type: domain.FieldType{Name: "String"}}}},
			},
		}

		err := validator.Validate(context.Background(), schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "datasource")
	})

	t.Run("Model without ID field fails", func(t *testing.T) {
		schema := &domain.Schema{
			Datasources: []domain.Datasource{
				{Name: "db", Provider: "postgresql", URL: "postgresql://localhost:5432/test"},
			},
			Models: []domain.Model{
				{
					Name: "User",
					Fields: []domain.Field{
						{Name: "email", Type: domain.FieldType{Name: "String"}},
					},
				},
			},
		}

		err := validator.Validate(context.Background(), schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "@id")
	})
}

func TestValidator_ValidateModel(t *testing.T) {
	validator := NewValidator()

	t.Run("Valid model passes", func(t *testing.T) {
		model := &domain.Model{
			Name: "User",
			Fields: []domain.Field{
				{
					Name:       "id",
					Type:       domain.FieldType{Name: "String"},
					IsRequired: true,
					Attributes: []domain.Attribute{{Name: "id"}},
				},
			},
		}

		err := validator.ValidateModel(context.Background(), model)
		assert.NoError(t, err)
	})

	t.Run("Model with lowercase name fails", func(t *testing.T) {
		model := &domain.Model{
			Name: "user",
			Fields: []domain.Field{
				{Name: "id", Type: domain.FieldType{Name: "String"}, Attributes: []domain.Attribute{{Name: "id"}}},
			},
		}

		err := validator.ValidateModel(context.Background(), model)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PascalCase")
	})

	t.Run("Model with duplicate fields fails", func(t *testing.T) {
		model := &domain.Model{
			Name: "User",
			Fields: []domain.Field{
				{Name: "id", Type: domain.FieldType{Name: "String"}, Attributes: []domain.Attribute{{Name: "id"}}},
				{Name: "id", Type: domain.FieldType{Name: "String"}},
			},
		}

		err := validator.ValidateModel(context.Background(), model)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})
}

func TestValidator_ValidateField(t *testing.T) {
	validator := NewValidator()

	t.Run("Valid field passes", func(t *testing.T) {
		field := &domain.Field{
			Name:       "email",
			Type:       domain.FieldType{Name: "String"},
			IsRequired: true,
		}

		err := validator.ValidateField(context.Background(), field)
		assert.NoError(t, err)
	})

	t.Run("Field with uppercase name fails", func(t *testing.T) {
		field := &domain.Field{
			Name: "Email",
			Type: domain.FieldType{Name: "String"},
		}

		err := validator.ValidateField(context.Background(), field)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "camelCase")
	})

	t.Run("List field with default fails", func(t *testing.T) {
		field := &domain.Field{
			Name:         "tags",
			Type:         domain.FieldType{Name: "String"},
			IsList:       true,
			DefaultValue: []string{},
		}

		err := validator.ValidateField(context.Background(), field)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "list fields cannot have default")
	})

	t.Run("Field with @default and @updatedAt fails", func(t *testing.T) {
		field := &domain.Field{
			Name: "timestamp",
			Type: domain.FieldType{Name: "DateTime"},
			Attributes: []domain.Attribute{
				{Name: "default"},
				{Name: "updatedAt"},
			},
		}

		err := validator.ValidateField(context.Background(), field)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "@default and @updatedAt")
	})
}

func TestValidator_ValidateEnum(t *testing.T) {
	validator := NewValidator()

	t.Run("Valid enum passes", func(t *testing.T) {
		schema := &domain.Schema{
			Datasources: []domain.Datasource{
				{Name: "db", Provider: "postgresql", URL: "url"},
			},
			Models: []domain.Model{
				{
					Name: "User",
					Fields: []domain.Field{
						{Name: "id", Type: domain.FieldType{Name: "String"}, Attributes: []domain.Attribute{{Name: "id"}}},
					},
				},
			},
			Enums: []domain.Enum{
				{
					Name:   "Role",
					Values: []string{"ADMIN", "USER"},
				},
			},
		}

		err := validator.Validate(context.Background(), schema)
		assert.NoError(t, err)
	})

	t.Run("Enum with lowercase name fails", func(t *testing.T) {
		schema := &domain.Schema{
			Datasources: []domain.Datasource{{Name: "db", Provider: "postgresql", URL: "url"}},
			Models: []domain.Model{
				{Name: "User", Fields: []domain.Field{{Name: "id", Type: domain.FieldType{Name: "String"}, Attributes: []domain.Attribute{{Name: "id"}}}}},
			},
			Enums: []domain.Enum{
				{Name: "role", Values: []string{"ADMIN"}},
			},
		}

		err := validator.Validate(context.Background(), schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PascalCase")
	})

	t.Run("Enum with duplicate values fails", func(t *testing.T) {
		schema := &domain.Schema{
			Datasources: []domain.Datasource{{Name: "db", Provider: "postgresql", URL: "url"}},
			Models: []domain.Model{
				{Name: "User", Fields: []domain.Field{{Name: "id", Type: domain.FieldType{Name: "String"}, Attributes: []domain.Attribute{{Name: "id"}}}}},
			},
			Enums: []domain.Enum{
				{Name: "Role", Values: []string{"ADMIN", "ADMIN"}},
			},
		}

		err := validator.Validate(context.Background(), schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})
}

func TestValidator_ValidateRelation(t *testing.T) {
	validator := NewValidator()

	t.Run("Valid relation passes", func(t *testing.T) {
		relation := &domain.Relation{
			Name:         "author",
			FromModel:    "Post",
			ToModel:      "User",
			FromFields:   []string{"authorId"},
			ToFields:     []string{"id"},
			RelationType: domain.ManyToOne,
			OnDelete:     domain.Cascade,
		}

		err := validator.ValidateRelation(context.Background(), relation)
		assert.NoError(t, err)
	})

	t.Run("Relation without models fails", func(t *testing.T) {
		relation := &domain.Relation{
			Name: "author",
		}

		err := validator.ValidateRelation(context.Background(), relation)
		assert.Error(t, err)
	})

	t.Run("Relation with invalid action fails", func(t *testing.T) {
		relation := &domain.Relation{
			Name:         "author",
			FromModel:    "Post",
			ToModel:      "User",
			RelationType: domain.ManyToOne,
			OnDelete:     domain.ReferentialAction("INVALID"),
		}

		err := validator.ValidateRelation(context.Background(), relation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid onDelete")
	})
}

func TestValidator_ValidateDatasource(t *testing.T) {
	validator := NewValidator()

	t.Run("Invalid provider fails", func(t *testing.T) {
		schema := &domain.Schema{
			Datasources: []domain.Datasource{
				{Name: "db", Provider: "invalid", URL: "url"},
			},
			Models: []domain.Model{
				{Name: "User", Fields: []domain.Field{{Name: "id", Type: domain.FieldType{Name: "String"}, Attributes: []domain.Attribute{{Name: "id"}}}}},
			},
		}

		err := validator.Validate(context.Background(), schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid provider")
	})
}

func TestValidator_ValidateIndexes(t *testing.T) {
	validator := NewValidator()

	t.Run("Index referencing unknown field fails", func(t *testing.T) {
		model := &domain.Model{
			Name: "User",
			Fields: []domain.Field{
				{Name: "id", Type: domain.FieldType{Name: "String"}, Attributes: []domain.Attribute{{Name: "id"}}},
			},
			Indexes: []domain.Index{
				{Fields: []string{"nonexistent"}},
			},
		}

		err := validator.ValidateModel(context.Background(), model)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})
}
