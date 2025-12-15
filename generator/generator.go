// Package generator generates Go code from Prisma schemas.
package generator

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/generator/codegen"
	"github.com/satishbabariya/prisma-go/internal/debug"
	ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// Generator generates Go client code from schema
type Generator struct {
	ast      *ast.SchemaAst
	provider string
}

// NewGenerator creates a new code generator
func NewGenerator(ast *ast.SchemaAst, provider string) *Generator {
	debug.Debug("Creating new generator", "provider", provider)
	return &Generator{
		ast:      ast,
		provider: provider,
	}
}

// GenerateClient generates the complete Prisma Go client
func (g *Generator) GenerateClient(outputDir string) error {
	debug.Debug("Starting client generation", "outputDir", outputDir, "provider", g.provider)

	// Validate schema before generation
	debug.Debug("Validating schema")
	if err := g.validateSchema(); err != nil {
		debug.Error("Schema validation failed", "error", err)
		return fmt.Errorf("schema validation failed: %w", err)
	}
	debug.Debug("Schema validation passed")

	// Generate model information from AST
	debug.Debug("Generating models from AST")
	models := codegen.GenerateModelsFromAST(g.ast)
	debug.Debug("Models generated", "count", len(models))

	if len(models) == 0 {
		debug.Error("No models found in schema")
		return fmt.Errorf("no models found in schema")
	}

	// Validate generated models
	debug.Debug("Validating generated models")
	if err := validateModels(models); err != nil {
		debug.Error("Model validation failed", "error", err)
		return fmt.Errorf("model validation failed: %w", err)
	}
	debug.Debug("Model validation passed")

	// Generate models.go
	debug.Debug("Generating models.go file", "outputDir", outputDir)
	if err := codegen.GenerateModelsFile(g.ast, models, outputDir); err != nil {
		debug.Error("Failed to generate models file", "error", err)
		return fmt.Errorf("failed to generate models: %w", err)
	}
	debug.Debug("models.go generated successfully")

	// Generate client.go
	debug.Debug("Generating client.go file", "outputDir", outputDir)
	if err := codegen.GenerateClientFile(models, g.provider, outputDir); err != nil {
		debug.Error("Failed to generate client file", "error", err)
		return fmt.Errorf("failed to generate client: %w", err)
	}
	debug.Debug("client.go generated successfully")
	debug.Info("Client generation completed", "outputDir", outputDir, "models", len(models))

	return nil
}

// validateSchema performs basic validation on the schema AST
func (g *Generator) validateSchema() error {
	// Check that we have at least one model
	models := g.ast.Models()
	modelCount := len(models)

	debug.Debug("Schema validation check", "totalTops", len(g.ast.Tops), "modelCount", modelCount)

	if modelCount == 0 {
		debug.Error("Schema validation failed: no models found")
		return fmt.Errorf("schema must contain at least one model")
	}

	return nil
}

// validateModels validates the generated model information
func validateModels(models []codegen.ModelInfo) error {
	// Check for duplicate table names
	tableNames := make(map[string]string) // table name -> model name
	for _, model := range models {
		debug.Debug("Validating model", "model", model.Name, "table", model.TableName)
		if existingModel, exists := tableNames[model.TableName]; exists {
			debug.Error("Duplicate table name detected", "table", model.TableName, "models", []string{existingModel, model.Name})
			return fmt.Errorf("duplicate table name %q: models %q and %q both map to the same table", model.TableName, existingModel, model.Name)
		}
		tableNames[model.TableName] = model.Name
	}
	debug.Debug("Model validation completed", "uniqueTables", len(tableNames))

	return nil
}
