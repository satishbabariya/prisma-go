package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/satishbabariya/prisma-go/cli/internal/ui"
	psl "github.com/satishbabariya/prisma-go/psl"
)

var formatCmd = &cobra.Command{
	Use:   "format [schema-path]",
	Short: "Format a Prisma schema file",
	Long: `Format a Prisma schema file according to Prisma formatting rules.

This command will:
- Parse and validate the schema
- Format it according to Prisma style guide
- Write the formatted schema back to the file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFormat,
}

var (
	formatSchemaPath string
	formatCheck      bool
	formatWrite      bool
)

func init() {
	formatCmd.Flags().StringVarP(&formatSchemaPath, "schema", "s", "schema.prisma", "Path to schema file")
	formatCmd.Flags().BoolVarP(&formatCheck, "check", "c", false, "Check if schema is formatted (exit with non-zero if not)")
	formatCmd.Flags().BoolVarP(&formatWrite, "write", "w", true, "Write formatted schema to file")

	// Aliases
	formatCmd.Aliases = []string{"fmt"}

	rootCmd.AddCommand(formatCmd)
}

func runFormat(cmd *cobra.Command, args []string) error {
	schemaPath := formatSchemaPath
	if len(args) > 0 {
		schemaPath = args[0]
	}

	// Check if file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", schemaPath)
	}

	// Read the schema file
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse and validate first
	_, diags := psl.ParseSchema(string(content))
	if diags.HasErrors() {
		ui.PrintError("Schema validation failed:")
		fmt.Fprintf(os.Stderr, "\n%s\n", diags.ToPrettyString(schemaPath, string(content)))
		return fmt.Errorf("cannot format schema with errors")
	}

	// Format the schema
	formatted, err := psl.Reformat(string(content), 2)
	if err != nil {
		return fmt.Errorf("failed to format schema: %w", err)
	}

	// Check mode - just verify formatting
	if formatCheck {
		if string(content) != formatted {
			ui.PrintError("Schema is not properly formatted")
			return fmt.Errorf("schema formatting check failed")
		}
		ui.PrintSuccess("Schema is properly formatted")
		return nil
	}

	// Write mode - write formatted content back
	if formatWrite {
		if err := os.WriteFile(schemaPath, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("failed to write formatted schema: %w", err)
		}

		absPath, _ := filepath.Abs(schemaPath)
		ui.PrintSuccess("Formatted %s", absPath)
	} else {
		// Just print formatted content
		fmt.Print(formatted)
	}

	return nil
}

