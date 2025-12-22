// Package commands implements the db seed command.
package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// NewDBSeedCommand creates the db seed command.
func NewDBSeedCommand() *cobra.Command {
	var seedScript string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed your database with initial data",
		Long: `Run a seed script to populate your database with initial data.

The seed script can be:
- A Go file (will be executed with 'go run')
- An SQL file (will be executed directly)
- A shell script (will be executed)

By default, looks for prisma/seed.go, prisma/seed.sql, or prisma/seed.sh`,
		Example: `  prisma db seed
  prisma db seed --script=custom_seed.go
  prisma db seed --script=seeds/initial_data.sql`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDBSeed(seedScript)
		},
	}

	cmd.Flags().StringVar(&seedScript, "script", "", "Path to seed script (auto-detected if not specified)")

	return cmd
}

func runDBSeed(scriptPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("🌱 Seeding database...")

	// Auto-detect seed script if not provided
	if scriptPath == "" {
		scriptPath = findSeedScript()
		if scriptPath == "" {
			return fmt.Errorf("no seed script found. Create prisma/seed.go, prisma/seed.sql, or prisma/seed.sh")
		}
	}

	// Check if file exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("seed script not found: %s", scriptPath)
	}

	fmt.Printf("📄 Using seed script: %s\n", scriptPath)

	// Execute based on file extension
	ext := filepath.Ext(scriptPath)
	var execCmd *exec.Cmd

	switch ext {
	case ".go":
		fmt.Println("▶️  Running Go seed script...")
		execCmd = exec.CommandContext(ctx, "go", "run", scriptPath)

	case ".sql":
		fmt.Println("▶️  Running SQL seed script...")
		// Get database URL
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL environment variable not set")
		}

		// Use psql for PostgreSQL (can be extended for other databases)
		execCmd = exec.CommandContext(ctx, "psql", dbURL, "-f", scriptPath)

	case ".sh":
		fmt.Println("▶️  Running shell seed script...")
		execCmd = exec.CommandContext(ctx, "sh", scriptPath)

	default:
		return fmt.Errorf("unsupported seed script type: %s (use .go, .sql, or .sh)", ext)
	}

	// Set environment and output
	execCmd.Env = os.Environ()
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	// Execute
	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("seed script failed: %w", err)
	}

	fmt.Println("✅ Database seeded successfully!")
	return nil
}

// findSeedScript looks for common seed script locations.
func findSeedScript() string {
	candidates := []string{
		"prisma/seed.go",
		"prisma/seed.sql",
		"prisma/seed.sh",
		"seeds/seed.go",
		"seeds/seed.sql",
		"db/seed.go",
		"db/seed.sql",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
