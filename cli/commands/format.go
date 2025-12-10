package commands

import (
	"fmt"
	"os"
	"path/filepath"

	psl "github.com/satishbabariya/prisma-go/psl"
)

func formatCommand(args []string) error {
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

	// Parse and validate first
	_, diags := psl.ParseSchema(string(content))
	if diags.HasErrors() {
		fmt.Fprintf(os.Stderr, "Schema validation failed:\n\n")
		fmt.Fprintf(os.Stderr, "%s\n", diags.ToPrettyString(schemaPath, string(content)))
		return fmt.Errorf("cannot format schema with errors")
	}

	// Format the schema
	formatted, err := psl.Reformat(string(content), 2)
	if err != nil {
		return fmt.Errorf("failed to format schema: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(schemaPath, []byte(formatted), 0644); err != nil {
		return fmt.Errorf("failed to write formatted schema: %w", err)
	}

	absPath, _ := filepath.Abs(schemaPath)
	fmt.Printf("✓ Formatted %s\n", absPath)

	return nil
}
