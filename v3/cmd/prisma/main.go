// Package main is the entry point for the Prisma CLI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/satishbabariya/prisma-go/v3/cmd/prisma/commands"
	"github.com/satishbabariya/prisma-go/v3/internal/config"
	"github.com/satishbabariya/prisma-go/v3/internal/utils/container"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by build)
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Create root command
	rootCmd := &cobra.Command{
		Use:     "prisma",
		Short:   "Prisma CLI for Go",
		Long:    "Prisma is a next-generation ORM for Go",
		Version: fmt.Sprintf("%s (commit: %s)", Version, Commit),
	}

	// Load configuration
	cfg := loadConfig()

	// Create dependency injection container
	c, err := container.NewContainer(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize container: %w", err)
	}
	defer c.Close(ctx)

	// Add commands
	rootCmd.AddCommand(commands.NewInitCommand())
	rootCmd.AddCommand(commands.NewGenerateCommand(c))
	rootCmd.AddCommand(commands.NewDBCommand()) // Add DB commands
	rootCmd.AddCommand(commands.NewMigrateCommand(c))
	rootCmd.AddCommand(commands.NewFormatCommand(c))
	rootCmd.AddCommand(commands.NewValidateCommand(c))

	// Execute root command
	return rootCmd.Execute()
}

func loadConfig() *config.Config {
	// Load configuration from environment variables or config file
	// For now, return default configuration
	return &config.Config{
		Database: config.DatabaseConfig{
			Provider:       "postgresql",
			URL:            os.Getenv("DATABASE_URL"),
			MaxConnections: 10,
		},
		Generator: config.GeneratorConfig{
			Output:  "./generated",
			Package: "db",
		},
	}
}
