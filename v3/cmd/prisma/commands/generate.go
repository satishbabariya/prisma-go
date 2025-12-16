// Package commands implements CLI commands.
package commands

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/service"
	"github.com/satishbabariya/prisma-go/v3/internal/utils/container"
	"github.com/spf13/cobra"
)

// NewGenerateCommand creates the generate command.
func NewGenerateCommand(c *container.Container) *cobra.Command {
	var schemaPath string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Prisma Client",
		Long:  "Generate type-safe database client from your Prisma schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(c, schemaPath)
		},
	}

	cmd.Flags().StringVar(&schemaPath, "schema", "prisma/schema.prisma", "Path to schema file")

	return cmd
}

func runGenerate(c *container.Container, schemaPath string) error {
	fmt.Println("Generating Prisma Client...")

	generateService := c.GenerateService()
	if generateService == nil {
		return fmt.Errorf("generate service not initialized")
	}

	input := service.GenerateInput{
		SchemaPath: schemaPath,
		Output:     "./generated",
	}

	if err := generateService.Generate(context.Background(), input); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	fmt.Println("✓ Generated Prisma Client successfully")

	return nil
}
