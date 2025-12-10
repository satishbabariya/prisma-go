package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/satishbabariya/prisma-go/generator"
	psl "github.com/satishbabariya/prisma-go/psl"
)

func generateCommand(args []string) error {
	schemaPath := "schema.prisma"
	if len(args) > 0 {
		schemaPath = args[0]
	}

	// Check if schema file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s (use: prisma-go generate <schema-path>)", schemaPath)
	}

	// Read schema
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	// Parse schema
	sourceFile := psl.NewSourceFile(schemaPath, string(content))
	ast, diags := psl.ParseSchemaFromFile(sourceFile)

	if diags.HasErrors() {
		fmt.Fprintf(os.Stderr, "Schema parsing failed:\n\n")
		fmt.Fprintf(os.Stderr, "%s\n", diags.ToPrettyString(schemaPath, string(content)))
		return fmt.Errorf("cannot generate from invalid schema")
	}

	// Determine output directory from generator config or use default
	outputDir := "./generated"
	provider := "postgresql" // Default provider

	// Extract provider from datasource
	for _, top := range ast.Tops {
		if datasource := top.AsSource(); datasource != nil {
			for _, prop := range datasource.Properties {
				if prop.Name.Name == "provider" {
					if value, _ := prop.Value.AsStringValue(); value != nil {
						provider = value.Value
					}
				}
			}
		}
		if gen := top.AsGenerator(); gen != nil {
			for _, prop := range gen.Properties {
				if prop.Name.Name == "output" {
					if value, _ := prop.Value.AsStringValue(); value != nil {
						outputDir = value.Value
					}
				}
			}
		}
	}

	fmt.Printf("🚀 Generating Prisma Client Go...\n\n")
	fmt.Printf("Schema: %s\n", schemaPath)
	fmt.Printf("Output: %s\n", outputDir)
	fmt.Printf("Provider: %s\n\n", provider)

	// Create generator
	gen := generator.NewGenerator(ast, provider)

	// Generate client code
	if err := gen.GenerateClient(outputDir); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	absPath, _ := filepath.Abs(outputDir)
	fmt.Printf("✓ Generated Prisma Client at %s\n\n", absPath)
	fmt.Println("Files generated:")
	fmt.Println("  • models.go  - Model structs")
	fmt.Println("  • client.go  - Prisma client")

	fmt.Println("\n💡 Next steps:")
	fmt.Println("  1. Import the generated package in your code")
	fmt.Println("  2. Create a client: client, _ := generated.NewPrismaClient(\"connection-string\")")
	fmt.Println("  3. Use the client: users, _ := client.User.FindMany(ctx)")

	return nil
}
