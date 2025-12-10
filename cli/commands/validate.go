package commands

import (
	"fmt"
	"os"
	"path/filepath"

	psl "github.com/satishbabariya/prisma-go/psl"
)

func validateCommand(args []string) error {
	if len(args) == 0 {
		// Default to schema.prisma in current directory
		args = []string{"schema.prisma"}
	}

	schemaPath := args[0]

	// Check if file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", schemaPath)
	}

	// Read the schema file
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Create source file and parse
	sourceFile := psl.NewSourceFile(schemaPath, string(content))
	ast, diags := psl.ParseSchemaFromFile(sourceFile)

	// Check for parsing errors
	if diags.HasErrors() {
		fmt.Fprintf(os.Stderr, "Schema parsing failed:\n\n")
		fmt.Fprintf(os.Stderr, "%s\n", diags.ToPrettyString(schemaPath, string(content)))
		return fmt.Errorf("schema has parsing errors")
	}

	// Check for warnings
	if len(diags.Warnings()) > 0 {
		fmt.Fprintf(os.Stderr, "Schema parsed with warnings:\n\n")
		fmt.Fprintf(os.Stderr, "%s\n", diags.WarningsToPrettyString(schemaPath, string(content)))
	}

	absPath, _ := filepath.Abs(schemaPath)
	fmt.Printf("✓ Schema is valid: %s\n", absPath)

	// Count models, enums, datasources, generators
	modelCount := 0
	enumCount := 0
	datasourceCount := 0
	generatorCount := 0

	for _, top := range ast.Tops {
		if top.AsModel() != nil {
			modelCount++
		} else if top.AsEnum() != nil {
			enumCount++
		} else if top.AsSource() != nil {
			datasourceCount++
		} else if top.AsGenerator() != nil {
			generatorCount++
		}
	}

	// Print summary
	fmt.Println("\nSchema Summary:")
	fmt.Printf("  • %d datasource(s)\n", datasourceCount)
	fmt.Printf("  • %d generator(s)\n", generatorCount)
	fmt.Printf("  • %d model(s)\n", modelCount)
	fmt.Printf("  • %d enum(s)\n", enumCount)

	// List models with field counts
	if modelCount > 0 {
		fmt.Println("\nModels:")
		for _, top := range ast.Tops {
			if model := top.AsModel(); model != nil {
				fmt.Printf("  • %s (%d fields)\n", model.Name.Name, len(model.Fields))
			}
		}
	}

	return nil
}
