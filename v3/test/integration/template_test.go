//go:build integration

package test

import (
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/ir"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/template"
)

// TestTemplateGenerationEndToEnd tests complete template generation workflow
func TestTemplateGenerationEndToEnd(t *testing.T) {
	// Setup test IR with a simple User model
	ir := &ir.IR{
		Models: []ir.Model{
			{
				Name:      "User",
				TableName: "users",
				Fields: []ir.Field{
					{Name: "id", GoName: "ID", Type: ir.FieldType{PrismaType: "Int", GoType: "int", IsScalar: true}, IsID: true},
					{Name: "name", GoName: "Name", Type: ir.FieldType{PrismaType: "String", GoType: "string", IsScalar: true}},
					{Name: "email", GoName: "Email", Type: ir.FieldType{PrismaType: "String", GoType: "string", IsScalar: true}, IsUnique: true},
					{Name: "age", GoName: "Age", Type: ir.FieldType{PrismaType: "Int", GoType: "int", IsScalar: true}},
				},
			},
		},
	}

	// Generate client code using template engine
	engine := template.NewEngine()
	generatedFiles, err := engine.RenderAll(ir)
	if err != nil {
		t.Fatalf("Failed to generate client code: %v", err)
	}

	if len(generatedFiles) == 0 {
		t.Fatal("Generated code is empty")
	}

	// Check that we generated expected files
	expectedFiles := []string{"models.go", "client.go", "queries.go"}
	for _, filename := range expectedFiles {
		if content, exists := generatedFiles[filename]; !exists || len(content) == 0 {
			t.Errorf("Generated file missing or empty: %s", filename)
		} else {
			t.Logf("Generated %s: %d bytes", filename, len(content))
		}
	}

	// Test that generated client code contains expected methods
	clientCode := string(generatedFiles["client.go"])
	expectedMethods := []string{
		"FindManyUser",
		"FindFirstUser",
		"CreateUser",
		"UpdateUser",
		"DeleteUser",
	}

	for _, method := range expectedMethods {
		if len(clientCode) == 0 || !contains(clientCode, method) {
			t.Errorf("Generated client code missing method: %s", method)
		} else {
			t.Logf("Found expected method: %s", method)
		}
	}
}

// TestModelGeneration tests model template generation
func TestModelGeneration(t *testing.T) {
	ir := &ir.IR{
		Models: []ir.Model{
			{
				Name:      "Post",
				TableName: "posts",
				Fields: []ir.Field{
					{Name: "id", GoName: "ID", Type: ir.FieldType{PrismaType: "Int", GoType: "int", IsScalar: true}, IsID: true},
					{Name: "title", GoName: "Title", Type: ir.FieldType{PrismaType: "String", GoType: "string", IsScalar: true}},
					{Name: "content", GoName: "Content", Type: ir.FieldType{PrismaType: "String", GoType: "string", IsScalar: true}},
					{Name: "published", GoName: "Published", Type: ir.FieldType{PrismaType: "Boolean", GoType: "bool", IsScalar: true}},
				},
			},
		},
	}

	engine := template.NewEngine()
	modelsCode, err := engine.RenderModels(ir)
	if err != nil {
		t.Fatalf("Failed to generate models: %v", err)
	}

	if len(modelsCode) == 0 {
		t.Fatal("Generated models code is empty")
	}

	// Check that generated model contains expected struct
	modelsStr := string(modelsCode)
	if !contains(modelsStr, "type Post struct") {
		t.Error("Generated model missing Post struct")
	}

	if !contains(modelsStr, "ID int") {
		t.Error("Generated model missing ID field")
	}

	if !contains(modelsStr, "Title string") {
		t.Error("Generated model missing Title field")
	}

	if !contains(modelsStr, "Published bool") {
		t.Error("Generated model missing Published field")
	}

	t.Logf("Generated models code successfully: %d bytes", len(modelsCode))
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
