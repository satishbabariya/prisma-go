// Package generator generates Go code from Prisma schemas.
package generator

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/generator/codegen"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// Generator generates Go client code from schema
type Generator struct {
	ast      *ast.SchemaAst
	provider string
}

// NewGenerator creates a new code generator
func NewGenerator(ast *ast.SchemaAst, provider string) *Generator {
	return &Generator{
		ast:      ast,
		provider: provider,
	}
}

// GenerateClient generates the complete Prisma Go client
func (g *Generator) GenerateClient(outputDir string) error {
	// Validate schema before generation
	if err := g.validateSchema(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Generate model information from AST
	models := codegen.GenerateModelsFromAST(g.ast)

	if len(models) == 0 {
		return fmt.Errorf("no models found in schema")
	}

	// Validate generated models
	if err := validateModels(models); err != nil {
		return fmt.Errorf("model validation failed: %w", err)
	}

	// Generate models.go
	if err := codegen.GenerateModelsFile(models, outputDir); err != nil {
		return fmt.Errorf("failed to generate models: %w", err)
	}

	// Generate client.go
	if err := codegen.GenerateClientFile(models, g.provider, outputDir); err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}

	return nil
}

// validateSchema performs basic validation on the schema AST
func (g *Generator) validateSchema() error {
	// Check that we have at least one model
	hasModel := false
	for _, top := range g.ast.Tops {
		if top.AsModel() != nil {
			hasModel = true
			break
		}
	}

	if !hasModel {
		return fmt.Errorf("schema must contain at least one model")
	}

	return nil
}

// validateModels validates the generated model information
func validateModels(models []codegen.ModelInfo) error {
	// Check for duplicate table names
	tableNames := make(map[string]string) // table name -> model name
	for _, model := range models {
		if existingModel, exists := tableNames[model.TableName]; exists {
			return fmt.Errorf("duplicate table name %q: models %q and %q both map to the same table", model.TableName, existingModel, model.Name)
		}
		tableNames[model.TableName] = model.Name
	}

	return nil
}
