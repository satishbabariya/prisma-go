// Package commands implements CLI commands.
package commands

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/utils/container"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates the validate command.
func NewValidateCommand(c *container.Container) *cobra.Command {
	var schemaPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Prisma schema",
		Long:  "Validate your Prisma schema for errors",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(c, schemaPath)
		},
	}

	cmd.Flags().StringVar(&schemaPath, "schema", "prisma/schema.prisma", "Path to schema file")

	return cmd
}

func runValidate(c *container.Container, schemaPath string) error {
	fmt.Println("Validating schema...")

	schemaService := c.SchemaService()
	if err := schemaService.ValidateSchema(context.Background(), schemaPath); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("✓ Schema %s is valid\n", schemaPath)

	return nil
}
