// Package service implements application services.
package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pslparser "github.com/satishbabariya/prisma-go/psl/parsing/v2"
	generatorSvc "github.com/satishbabariya/prisma-go/v3/internal/core/generator/service"
	"github.com/satishbabariya/prisma-go/v3/internal/repository"
)

// GenerateService handles code generation.
type GenerateService struct {
	schemaRepo repository.SchemaRepository
}

// NewGenerateService creates a new generate service.
func NewGenerateService(
	schemaRepo repository.SchemaRepository,
	analyzer interface{},
	engine interface{},
	writer interface{},
) *GenerateService {
	return &GenerateService{
		schemaRepo: schemaRepo,
	}
}

// GenerateInput contains generation input parameters.
type GenerateInput struct {
	SchemaPath string
	Output     string
}

// Generate generates code from a Prisma schema.
func (s *GenerateService) Generate(ctx context.Context, input GenerateInput) error {
	// Read schema file directly
	content, err := os.ReadFile(input.SchemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse directly to PSL AST using PSL parser
	pslAst, parseErr := pslparser.ParseSchema("schema.prisma", strings.NewReader(string(content)))
	if parseErr != nil {
		return fmt.Errorf("failed to parse schema: %w", parseErr)
	}

	// Determine output directory
	outputDir := input.Output
	if outputDir == "" {
		// Try to find output in generator config
		for _, gen := range pslAst.Generators() {
			if provider := gen.GetProperty("provider"); provider != nil {
				// Check if provider is "go"
				if strVal, ok := provider.Value.AsStringValue(); ok && strVal.GetValue() == "go" {
					// Found go generator
					if out := gen.GetProperty("output"); out != nil {
						if outStr, ok := out.Value.AsStringValue(); ok {
							outputDir = outStr.GetValue()
						}
					}
					break
				}
			}
		}
	}

	if outputDir == "" {
		// Default to directory of schema file
		outputDir = filepath.Join(filepath.Dir(input.SchemaPath), "generated")
	} else {
		// If outputDir is relative, make it relative to schema path
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(filepath.Dir(input.SchemaPath), outputDir)
		}
	}

	// Create generator and run
	generator := generatorSvc.NewGenerator(outputDir)
	if err := generator.Generate(ctx, pslAst); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	return nil
}
