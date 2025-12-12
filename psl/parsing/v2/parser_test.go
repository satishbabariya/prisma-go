package schema

import (
	"testing"
)

func TestParseBasicModel(t *testing.T) {
	input := `
model User {
  id    Int    @id @default(autoincrement())
  email String @unique
  name  String?
  posts Post[]
}
`
	schema, err := ParseSchemaString("test.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	models := schema.Models()
	if len(models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(models))
	}

	model := models[0]
	if model.GetName() != "User" {
		t.Errorf("Expected model name 'User', got '%s'", model.GetName())
	}

	if len(model.Fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(model.Fields))
	}
}

func TestParseEnum(t *testing.T) {
	input := `
enum Role {
  USER
  ADMIN
  MODERATOR @map("mod")
}
`
	schema, err := ParseSchemaString("test.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	enums := schema.Enums()
	if len(enums) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(enums))
	}

	enum := enums[0]
	if enum.GetName() != "Role" {
		t.Errorf("Expected enum name 'Role', got '%s'", enum.GetName())
	}

	if len(enum.Values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(enum.Values))
	}
}

func TestParseDatasource(t *testing.T) {
	input := `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}
`
	schema, err := ParseSchemaString("test.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	sources := schema.Sources()
	if len(sources) != 1 {
		t.Fatalf("Expected 1 datasource, got %d", len(sources))
	}

	source := sources[0]
	if source.GetName() != "db" {
		t.Errorf("Expected datasource name 'db', got '%s'", source.GetName())
	}

	if len(source.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(source.Properties))
	}
}

func TestParseGenerator(t *testing.T) {
	input := `
generator client {
  provider = "prisma-client-js"
  output   = "./generated/client"
}
`
	schema, err := ParseSchemaString("test.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	generators := schema.Generators()
	if len(generators) != 1 {
		t.Fatalf("Expected 1 generator, got %d", len(generators))
	}

	gen := generators[0]
	if gen.GetName() != "client" {
		t.Errorf("Expected generator name 'client', got '%s'", gen.GetName())
	}
}

func TestParseCompleteSchema(t *testing.T) {
	input := `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-js"
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  role      Role     @default(USER)
  posts     Post[]
  profile   Profile?
  createdAt DateTime @default(now())
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  author    User     @relation(fields: [authorId], references: [id])
  authorId  Int

  @@index([authorId])
}

model Profile {
  id     Int    @id @default(autoincrement())
  bio    String
  user   User   @relation(fields: [userId], references: [id])
  userId Int    @unique
}

enum Role {
  USER
  ADMIN
}
`
	schema, err := ParseSchemaString("test.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	if len(schema.Sources()) != 1 {
		t.Errorf("Expected 1 datasource, got %d", len(schema.Sources()))
	}

	if len(schema.Generators()) != 1 {
		t.Errorf("Expected 1 generator, got %d", len(schema.Generators()))
	}

	if len(schema.Models()) != 3 {
		t.Errorf("Expected 3 models, got %d", len(schema.Models()))
	}

	if len(schema.Enums()) != 1 {
		t.Errorf("Expected 1 enum, got %d", len(schema.Enums()))
	}
}

func TestParseView(t *testing.T) {
	input := `
view UserStats {
  userId     Int
  postCount  Int
  totalViews Int
}
`
	schema, err := ParseSchemaString("test.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	models := schema.Models()
	if len(models) != 1 {
		t.Fatalf("Expected 1 model/view, got %d", len(models))
	}

	view := models[0]
	if !view.IsView() {
		t.Error("Expected IsView() to be true")
	}

	if view.GetName() != "UserStats" {
		t.Errorf("Expected view name 'UserStats', got '%s'", view.GetName())
	}
}

func TestParseDocumentation(t *testing.T) {
	input := `
/// This is a user model
/// It stores user information
model User {
  /// The unique identifier
  id Int @id
}
`
	schema, err := ParseSchemaString("test.prisma", input)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	models := schema.Models()
	if len(models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(models))
	}

	model := models[0]
	doc := model.GetDocumentation()
	if doc == "" {
		t.Error("Expected model to have documentation")
	}
}
