// Package commands implements CLI commands.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewInitCommand creates the init command.
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Prisma project",
		Long:  "Create a new Prisma schema file and configuration",
		RunE:  runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	schemaPath := "prisma/schema.prisma"

	// Create prisma directory
	if err := os.MkdirAll("prisma", 0755); err != nil {
		return fmt.Errorf("failed to create prisma directory: %w", err)
	}

	// Check if schema already exists
	if _, err := os.Stat(schemaPath); err == nil {
		return fmt.Errorf("schema file already exists: %s", schemaPath)
	}

	// Create default schema
	defaultSchema := `// This is your Prisma schema file,
// learn more about it in the docs: https://pris.ly/d/prisma-schema

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-go"
  output   = "./generated"
}

model User {
  id    Int     @id @default(autoincrement())
  email String  @unique
  name  String?
}
`

	if err := os.WriteFile(schemaPath, []byte(defaultSchema), 0644); err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}

	fmt.Printf("✓ Created Prisma schema at %s\n", schemaPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Set the DATABASE_URL in your .env file")
	fmt.Println("2. Run `prisma generate` to generate the client")
	fmt.Println("3. Start using Prisma in your application")

	return nil
}
