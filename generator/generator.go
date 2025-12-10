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
	// Generate model information from AST
	models := codegen.GenerateModelsFromAST(g.ast)

	if len(models) == 0 {
		return fmt.Errorf("no models found in schema")
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

