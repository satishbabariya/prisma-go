// Package commands implements CLI commands.
package commands

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/utils/container"
	"github.com/spf13/cobra"
)

// NewFormatCommand creates the format command.
func NewFormatCommand(c *container.Container) *cobra.Command {
	var schemaPath string

	cmd := &cobra.Command{
		Use:   "format",
		Short: "Format Prisma schema",
		Long:  "Format your Prisma schema file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFormat(c, schemaPath)
		},
	}

	cmd.Flags().StringVar(&schemaPath, "schema", "prisma/schema.prisma", "Path to schema file")

	return cmd
}

func runFormat(c *container.Container, schemaPath string) error {
	fmt.Println("Formatting schema...")

	schemaService := c.SchemaService()
	if err := schemaService.FormatSchema(context.Background(), schemaPath); err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	fmt.Printf("✓ Formatted %s\n", schemaPath)

	return nil
}
