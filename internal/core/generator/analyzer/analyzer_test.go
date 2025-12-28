package analyzer

import (
	"strings"
	"testing"

	pslparser "github.com/satishbabariya/prisma-go/psl/parsing/v2"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
)

func TestAnalyze_Relations(t *testing.T) {
	schema := `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    String @id
  email String @unique
  posts Post[]
}

model Post {
  id        String  @id
  title     String
  authorId  String
  author    User    @relation(fields: [authorId], references: [id])
}
`

	// Parse schema
	ast, err := pslparser.ParseSchema("schema.prisma", strings.NewReader(schema))
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Analyze
	analyzer := NewSchemaAnalyzer(ast)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Failed to analyze schema: %v", err)
	}

	// Verify User model
	var userModel ir.Model
	found := false
	for _, m := range result.Models {
		if m.Name == "User" {
			userModel = m
			found = true
			break
		}
	}
	if !found {
		t.Fatal("User model not found")
	}

	// Check User.posts field
	var postsField ir.Field
	found = false
	for _, f := range userModel.Fields {
		if f.Name == "posts" {
			postsField = f
			found = true
			break
		}
	}
	if !found {
		t.Fatal("User.posts field not found")
	}

	if !postsField.Type.IsModel {
		t.Error("User.posts should be a model type")
	}
	if postsField.Type.PrismaType != "Post" {
		t.Errorf("User.posts type expected Post, got %s", postsField.Type.PrismaType)
	}
	// This is one of the things we expect to be populated by relation analysis
	if postsField.Relation == nil {
		t.Error("User.posts RelationInfo is nil")
	} else if postsField.Relation.RelationType != ir.OneToMany {
		t.Errorf("User.posts relation type expected OneToMany, got %s", postsField.Relation.RelationType)
	}

	// Checker Post model
	var postModel ir.Model
	found = false
	for _, m := range result.Models {
		if m.Name == "Post" {
			postModel = m
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Post model not found")
	}

	// Check Post.author field
	var authorField ir.Field
	found = false
	for _, f := range postModel.Fields {
		if f.Name == "author" {
			authorField = f
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Post.author field not found")
	}

	if authorField.Relation == nil {
		t.Error("Post.author RelationInfo is nil")
	} else {
		if authorField.Relation.RelationType != ir.OneToMany {
			t.Errorf("Post.author relation type expected OneToMany, got %s", authorField.Relation.RelationType)
		}
		if authorField.Relation.ForeignKey != "authorId" {
			t.Errorf("Post.author foreign key expected authorId, got %s", authorField.Relation.ForeignKey)
		}
		if authorField.Relation.References != "id" {
			t.Errorf("Post.author references expected id, got %s", authorField.Relation.References)
		}
	}
}
