package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/satishbabariya/prisma-go/cli/internal/ui"
	psl "github.com/satishbabariya/prisma-go/psl"
)

var validateCmd = &cobra.Command{
	Use:   "validate [schema-path]",
	Short: "Validate a Prisma schema file",
	Long: `Validate a Prisma schema file for syntax and semantic errors.

This command will:
- Parse the schema file
- Check for syntax errors
- Check for semantic errors
- Display validation results`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

var (
	validateSchemaPath string
)

func init() {
	validateCmd.Flags().StringVarP(&validateSchemaPath, "schema", "s", "schema.prisma", "Path to schema file")

	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	schemaPath := getSchemaPath(validateSchemaPath, args)

	ui.PrintHeader("Prisma-Go", "Validate Schema")

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
		ui.PrintError("Schema parsing failed:")
		fmt.Fprintf(os.Stderr, "\n%s\n", diags.ToPrettyString(schemaPath, string(content)))
		return fmt.Errorf("schema has parsing errors")
	}

	// Check for warnings
	if len(diags.Warnings()) > 0 {
		ui.PrintWarning("Schema parsed with warnings:")
		fmt.Fprintf(os.Stderr, "\n%s\n", diags.WarningsToPrettyString(schemaPath, string(content)))
	}

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

	// Validate minimum requirements: datasource and generator are required
	if datasourceCount == 0 {
		ui.PrintError("Schema validation failed:")
		fmt.Fprintf(os.Stderr, "\n❌ A datasource must be defined in the schema.\n")
		return fmt.Errorf("schema missing required datasource")
	}

	if generatorCount == 0 {
		ui.PrintError("Schema validation failed:")
		fmt.Fprintf(os.Stderr, "\n❌ A generator must be defined in the schema.\n")
		return fmt.Errorf("schema missing required generator")
	}

	absPath, _ := filepath.Abs(schemaPath)
	ui.PrintSuccess("Schema is valid: %s", absPath)

	// Print summary
	fmt.Println()
	ui.PrintSection("Schema Summary")
	summary := []string{
		fmt.Sprintf("%d datasource(s)", datasourceCount),
		fmt.Sprintf("%d generator(s)", generatorCount),
		fmt.Sprintf("%d model(s)", modelCount),
		fmt.Sprintf("%d enum(s)", enumCount),
	}
	ui.PrintList(summary)

	// List models with field counts
	if modelCount > 0 {
		fmt.Println()
		ui.PrintSection("Models")
		for _, top := range ast.Tops {
			if model := top.AsModel(); model != nil {
				ui.PrintInfo("%s (%d fields)", model.Name.Name, len(model.Fields))
			}
		}
	}

	return nil
}
