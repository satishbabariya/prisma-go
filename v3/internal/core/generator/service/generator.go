// Package generator provides the code generation service.
package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pslast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/analyzer"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/template"
)

// Generator generates Go code from Prisma schemas.
type Generator struct {
	outputDir      string
	templateEngine *template.Engine
}

// NewGenerator creates a new generator.
func NewGenerator(outputDir string) *Generator {
	templateEngine := template.NewEngine()

	// Load templates from embedded templates
	templatesDir := filepath.Join("internal", "core", "generator", "template", "templates")
	templateEngine.LoadTemplates(templatesDir) // Ignore error, templates are built-in

	return &Generator{
		outputDir:      outputDir,
		templateEngine: templateEngine,
	}
}

// Generate generates code from a Prisma schema.
func (g *Generator) Generate(ctx context.Context, schema *pslast.SchemaAst) error {
	// Step 1: Analyze schema and produce IR
	schemaAnalyzer := analyzer.NewSchemaAnalyzer(schema)
	ir, err := schemaAnalyzer.Analyze()
	if err != nil {
		return fmt.Errorf("failed to analyze schema: %w", err)
	}

	// Step 2: Generate code using template engine
	if g.templateEngine != nil {
		files, err := g.templateEngine.RenderAll(ir)
		if err != nil {
			return fmt.Errorf("failed to render templates: %w", err)
		}

		// Step 3: Write generated files to output directory
		if err := os.MkdirAll(g.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		for filename, content := range files {
			outputPath := filepath.Join(g.outputDir, filename)
			if err := os.WriteFile(outputPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write output file %s: %w", filename, err)
			}
		}

		return nil
	}

	// Fallback to basic generation
	return fmt.Errorf("template engine not available")
}

// GenerateToString generates code and returns it as a string (for testing).
func (g *Generator) GenerateToString(ctx context.Context, schema *pslast.SchemaAst) (string, error) {
	// Analyze schema
	schemaAnalyzer := analyzer.NewSchemaAnalyzer(schema)
	ir, err := schemaAnalyzer.Analyze()
	if err != nil {
		return "", fmt.Errorf("failed to analyze schema: %w", err)
	}

	// Generate using template engine
	if g.templateEngine != nil {
		files, err := g.templateEngine.RenderAll(ir)
		if err != nil {
			return "", fmt.Errorf("failed to render templates: %w", err)
		}

		// Combine all files into one output for testing
		var result strings.Builder
		for filename, content := range files {
			result.WriteString("// --- " + filename + " ---\n")
			result.Write(content)
			result.WriteString("\n\n")
		}

		return result.String(), nil
	}

	return "", fmt.Errorf("template engine not available")
}
