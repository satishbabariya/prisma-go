// Package generator provides the code generation service.
package generator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	pslast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/analyzer"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/astgen"
	"github.com/satishbabariya/prisma-go/v3/internal/core/generator/writer"
)

// Generator generates Go code from Prisma schemas.
type Generator struct {
	outputDir string
}

// NewGenerator creates a new generator.
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
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

	// Step 2: Build AST from IR
	builder := astgen.NewBuilder(ir)
	fileAST := builder.BuildFile()

	// Step 3: Write to file
	astWriter := writer.NewASTWriter()
	var buf bytes.Buffer

	// Write header
	header := writer.NewFileHeader()
	if err := header.Write(&buf, nil); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write AST
	if err := astWriter.Write(&buf, fileAST); err != nil {
		return fmt.Errorf("failed to write AST: %w", err)
	}

	// Step 4: Write to output file
	outputPath := filepath.Join(g.outputDir, "models.gen.go")
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// GenerateToString generates code and returns it as a string (for testing).
func (g *Generator) GenerateToString(ctx context.Context, schema *pslast.SchemaAst) (string, error) {
	// Analyze schema
	schemaAnalyzer := analyzer.NewSchemaAnalyzer(schema)
	ir, err := schemaAnalyzer.Analyze()
	if err != nil {
		return "", fmt.Errorf("failed to analyze schema: %w", err)
	}

	// Build AST
	builder := astgen.NewBuilder(ir)
	fileAST := builder.BuildFile()

	// Write to buffer
	astWriter := writer.NewASTWriter()
	var buf bytes.Buffer

	// Write header
	header := writer.NewFileHeader()
	if err := header.Write(&buf, nil); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	// Write AST
	if err := astWriter.Write(&buf, fileAST); err != nil {
		return "", fmt.Errorf("failed to write AST: %w", err)
	}

	return buf.String(), nil
}
