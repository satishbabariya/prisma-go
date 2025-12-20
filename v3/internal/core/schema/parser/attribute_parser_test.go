package parser

import (
	"testing"

	"strings"

	pslast "github.com/satishbabariya/prisma-go/psl/ast/v2"
	pslparser "github.com/satishbabariya/prisma-go/psl/parsing/v2"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttributeParser_ParseFieldAttributes(t *testing.T) {
	schema := `
model User {
  id    String @id @default(uuid())
  email String @unique @map("user_email")
  name  String?
  age   Int    @default(0)
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt
}
`

	ast, err := pslparser.ParseSchema("test.prisma", strings.NewReader(schema))
	require.NoError(t, err)

	parser := NewAttributeParser()
	models := ast.Models()
	require.Len(t, models, 1)

	model := models[0]

	t.Run("Parse @id attribute", func(t *testing.T) {
		idField := model.Fields[0]
		assert.True(t, parser.IsID(idField))
	})

	t.Run("Parse @unique attribute", func(t *testing.T) {
		emailField := model.Fields[1]
		assert.True(t, parser.IsUnique(emailField))
	})

	t.Run("Parse @updatedAt attribute", func(t *testing.T) {
		updatedAtField := model.Fields[5]
		assert.True(t, parser.IsUpdatedAt(updatedAtField))
	})

	t.Run("Parse @default attribute", func(t *testing.T) {
		ageField := model.Fields[3]
		defaultVal := parser.GetDefaultValue(ageField)
		assert.NotNil(t, defaultVal)
	})
}

func TestAttributeParser_ParseRelationAttribute(t *testing.T) {
	schema := `
model User {
  id    String @id
  posts Post[]
}

model Post {
  id       String @id
  title    String
  authorId String
  author   User   @relation(fields: [authorId], references: [id], onDelete: Cascade)
}
`

	ast, err := pslparser.ParseSchema("test.prisma", strings.NewReader(schema))
	require.NoError(t, err)

	parser := NewAttributeParser()
	models := ast.Models()
	require.Len(t, models, 2)

	postModel := models[1]
	authorField := postModel.Fields[3]

	t.Run("Parse relation with fields and references", func(t *testing.T) {
		relation := parser.ParseRelationAttribute(authorField)
		require.NotNil(t, relation)

		assert.Equal(t, "author", relation.Name)
		assert.Equal(t, []string{"authorId"}, relation.FromFields)
		assert.Equal(t, []string{"id"}, relation.ToFields)
		assert.Equal(t, domain.Cascade, relation.OnDelete)
	})
}

func TestAttributeParser_ParseBlockAttributes(t *testing.T) {
	schema := `
model User {
  id    String @id
  email String @unique

  @@map("users")
  @@index([email])
}
`

	ast, err := pslparser.ParseSchema("test.prisma", strings.NewReader(schema))
	require.NoError(t, err)

	parser := NewAttributeParser()
	models := ast.Models()
	require.Len(t, models, 1)

	model := models[0]

	t.Run("Parse @@map attribute", func(t *testing.T) {
		mappedName := parser.GetMappedName(convertToInterfaces(model.BlockAttributes))
		assert.Equal(t, "users", mappedName)
	})

	t.Run("Parse block attributes", func(t *testing.T) {
		attrs := parser.ParseBlockAttributes(model)
		assert.NotEmpty(t, attrs)

		// Check for @@map
		hasMap := false
		for _, attr := range attrs {
			if attr.Name == "map" {
				hasMap = true
				break
			}
		}
		assert.True(t, hasMap)
	})
}

func TestAttributeParser_ParseDefaultFunctions(t *testing.T) {
	schema := `
model User {
  id        String   @id @default(uuid())
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
  count     Int      @default(autoincrement())
}
`

	ast, err := pslparser.ParseSchema("test.prisma", strings.NewReader(schema))
	require.NoError(t, err)

	parser := NewAttributeParser()
	models := ast.Models()
	require.Len(t, models, 1)

	model := models[0]

	t.Run("Parse uuid() function", func(t *testing.T) {
		idField := model.Fields[0]
		defaultVal := parser.GetDefaultValue(idField)
		assert.NotNil(t, defaultVal)

		// Should be a function call map
		if fnMap, ok := defaultVal.(map[string]interface{}); ok {
			assert.Equal(t, "uuid", fnMap["function"])
		}
	})

	t.Run("Parse now() function", func(t *testing.T) {
		createdAtField := model.Fields[1]
		defaultVal := parser.GetDefaultValue(createdAtField)
		assert.NotNil(t, defaultVal)

		if fnMap, ok := defaultVal.(map[string]interface{}); ok {
			assert.Equal(t, "now", fnMap["function"])
		}
	})
}

func TestAttributeParser_ParseMapAttribute(t *testing.T) {
	schema := `
model User {
  id        String @id
  email     String @map("user_email")
  firstName String @map("first_name")
  
  @@map("users_table")
}
`

	ast, err := pslparser.ParseSchema("test.prisma", strings.NewReader(schema))
	require.NoError(t, err)

	parser := NewAttributeParser()
	models := ast.Models()
	require.Len(t, models, 1)

	model := models[0]

	t.Run("Parse field @map", func(t *testing.T) {
		emailField := model.Fields[1]
		attrs := parser.ParseFieldAttributes(emailField)

		hasMap := false
		for _, attr := range attrs {
			if attr.Name == "map" && len(attr.Arguments) > 0 {
				hasMap = true
				assert.Equal(t, "user_email", attr.Arguments[0])
				break
			}
		}
		assert.True(t, hasMap, "Should have @map attribute")
	})

	t.Run("Parse block @@map", func(t *testing.T) {
		mappedName := parser.GetMappedName(convertToInterfaces(model.BlockAttributes))
		assert.Equal(t, "users_table", mappedName)
	})
}

// Helper function to convert PSL block attributes to interface slice
func convertToInterfaces(attrs []*pslast.BlockAttribute) []interface{} {
	result := make([]interface{}, len(attrs))
	for i, attr := range attrs {
		result[i] = attr
	}
	return result
}
